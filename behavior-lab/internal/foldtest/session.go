package foldtest

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"tlaloc.local/behaviorlab/internal/pdfmemory"
	"tlaloc.local/behaviorlab/internal/target"
)

// SessionConfig holds configuration for a test session
type SessionConfig struct {
	WorkDir  string
	StoreDir string
	Manifest pdfmemory.Manifest
	Cover    string // pre-built cover text
	Model    string
	BaseURL  string
	MaxTurns int
	Budget   int
	// ExtraContext, when non-empty, is appended to the system prompt as a
	// "specialist" hint (e.g. a page suggestion from a blackboard
	// observation) alongside the cover. Empty by default, so existing
	// callers are unaffected.
	ExtraContext string
}

// SessionResult holds the results of running a question through the fold-bench harness
type SessionResult struct {
	Question             string                    `json:"question"`
	Answer               string                    `json:"answer"`
	Turns                int                       `json:"turns"`
	Unfolds              []UnfoldMetric            `json:"unfolds"`
	TotalTokensPrompt    int                       `json:"total_tokens_prompt"`
	TotalTokensCompletion int                      `json:"total_tokens_completion"`
	TotalLatencyMs       int64                     `json:"total_latency_ms"`
	LearnedCommandUsed   bool                      `json:"learned_command_used"`
	Status               string                    `json:"status"`
}

// UnfoldMetric records metrics about a single unfold operation
type UnfoldMetric struct {
	Address         string `json:"address"`
	Fidelity        string `json:"fidelity,omitempty"`
	TokenCost       int    `json:"token_cost"`
	Verified        bool   `json:"verified"`
	LatencyMs       int64  `json:"latency_ms"`
	ContentLength   int    `json:"content_length"`
}

// RunSession executes a single question through the fold-bench harness
func RunSession(ctx context.Context, config SessionConfig, question string) (SessionResult, error) {
	result := SessionResult{
		Question: question,
		Unfolds:  []UnfoldMetric{},
		Status:   "RUNNING",
	}

	// Build system prompt: cover + few-shot examples of new command syntax
	systemPrompt := buildSystemPrompt(config.Cover, config.ExtraContext)

	// Initialize conversation
	userTurns := []string{question}
	var allMessages []string

	learnedCommands := false

	for turn := 0; turn < config.MaxTurns; turn++ {
		result.Turns = turn + 1

		// Build context for this turn: system + all prior messages + current question
		userMessage := userTurns[len(userTurns)-1]
		fullPrompt := systemPrompt + "\n\n" + strings.Join(allMessages, "\n\n") + "\n\nUser: " + userMessage

		// Call model (plain text, no vision)
		response, err := target.OpenAICompat{
			BaseURL: config.BaseURL,
			Model:   config.Model,
		}.Complete(ctx, systemPrompt, fullPrompt)
		if err != nil {
			return result, fmt.Errorf("model call failed: %w", err)
		}

		allMessages = append(allMessages, "User: "+userMessage)
		allMessages = append(allMessages, "Assistant: "+response)

		// Track token usage (rough estimate: ~4 chars per token)
		result.TotalTokensPrompt += len(fullPrompt) / 4
		result.TotalTokensCompletion += len(response) / 4

		// Try to parse a command from the response
		cmd, ok := ParseCommand(response)
		if !ok {
			// No command found; treat as final answer
			result.Answer = response
			result.Status = "ANSWERED"
			break
		}

		learnedCommands = true

		// Execute the command
		if cmd.Op == "UNFOLD" {
			metric, nextTurn, err := executeUnfold(ctx, config, cmd)
			if err != nil {
				result.Status = "UNFOLD_ERROR"
				return result, err
			}
			result.Unfolds = append(result.Unfolds, metric)
			userTurns = append(userTurns, nextTurn)
		} else if cmd.Op == "GROUP" {
			nextTurn, err := executeGroup(config, cmd)
			if err != nil {
				result.Status = "GROUP_ERROR"
				return result, err
			}
			userTurns = append(userTurns, nextTurn)
		}
	}

	result.LearnedCommandUsed = learnedCommands

	if result.Status == "" {
		result.Status = "MAX_TURNS_REACHED"
	}

	return result, nil
}

func executeUnfold(ctx context.Context, config SessionConfig, cmd *Command) (UnfoldMetric, string, error) {
	metric := UnfoldMetric{
		Address:  cmd.Address,
		Fidelity: cmd.Fidelity,
	}

	// Call unfold
	_, packet, err := UnfoldToTempFile(config.WorkDir, config.StoreDir, config.Manifest, cmd.Address, cmd.Fidelity, config.Budget)
	if err != nil {
		return metric, "", fmt.Errorf("unfold failed: %w", err)
	}

	// Collect metrics
	metric.Verified = true // Placeholder: would check Merkle verification in packet
	metric.TokenCost = packet.Budget.UsedTokens
	if len(packet.Evidence) > 0 {
		metric.ContentLength = len(packet.Evidence[0].Content)
	}

	// Build next turn message with evidence
	var nextTurn strings.Builder
	nextTurn.WriteString(fmt.Sprintf("Evidence from %s:\n", cmd.Address))
	for _, ev := range packet.Evidence {
		nextTurn.WriteString(fmt.Sprintf("[%s]\n%s\n", ev.Address, ev.Content))
	}

	return metric, nextTurn.String(), nil
}

func executeGroup(config SessionConfig, cmd *Command) (string, error) {
	// Load graph
	graph, err := pdfmemory.LoadGraph(config.StoreDir)
	if err != nil {
		return "", fmt.Errorf("loading graph: %w", err)
	}

	// Find the term and its neighbors
	_, ok := graph.Nodes[cmd.Term]
	if !ok {
		return fmt.Sprintf("Term %q not found in graph.\n", cmd.Term), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Related terms for %q (depth %d):\n", cmd.Term, cmd.Depth))

	// Collect neighbors up to the requested depth
	neighbors := collectNeighbors(graph, cmd.Term, cmd.Depth)
	for _, neighbor := range neighbors {
		if neighborNode, ok := graph.Nodes[neighbor]; ok {
			result.WriteString(fmt.Sprintf("  - %s (document_freq: %d)\n", neighbor, neighborNode.DocumentFrequency))
		}
	}

	return result.String(), nil
}

func collectNeighbors(g pdfmemory.Graph, term string, maxDepth int) []string {
	visited := make(map[string]bool)
	var neighbors []string

	var collect func(t string, depth int)
	collect = func(t string, depth int) {
		if depth <= 0 || visited[t] {
			return
		}
		visited[t] = true

		node, ok := g.Nodes[t]
		if !ok {
			return
		}

		for _, edge := range node.Neighbors {
			neighbors = append(neighbors, edge.Term)
			collect(edge.Term, depth-1)
		}
	}

	collect(term, maxDepth)
	return neighbors
}

func buildSystemPrompt(cover, extraContext string) string {
	prompt := `You are reading a document. You have been given a compact text cover of the entire document, showing key terms per page.

To learn more details about a specific page or block, use the UNFOLD command:
  UNFOLD page:N        (e.g., UNFOLD page:5)
  UNFOLD block:doc/page-N/blocks/M  (e.g., UNFOLD block:doc/page-3/blocks/2)

To see related terms, use the GROUP command:
  GROUP term:<term> depth:<N>  (e.g., GROUP term:authentication depth:2)

After using UNFOLD or GROUP, you will receive evidence content. Continue asking questions or unfold more sections until you can provide your final answer.

When you have enough information to answer the original question, provide your final answer in this format:
  ANSWER: <your answer>

Here is the document cover:
---
` + cover + `
---
`

	if extraContext != "" {
		prompt += `
Additional context from specialists:
---
` + extraContext + `
---
`
	}

	return prompt + `
Begin.`
}

// extractAnswerFromResponse tries to extract "ANSWER: ..." from response
func extractAnswerFromResponse(response string) string {
	re := regexp.MustCompile(`ANSWER:\s*(.+?)(?:\n|$)`)
	matches := re.FindStringSubmatch(response)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return response // Fallback to full response if no ANSWER: found
}
