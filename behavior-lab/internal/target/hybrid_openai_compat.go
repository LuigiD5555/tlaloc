package target

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ToolDefinition struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type ToolExecutor interface {
	Execute(ctx context.Context, name string, arguments json.RawMessage) (string, error)
}

type HybridInput struct {
	SystemPrompt string
	Question     string
	ImagePNG     []byte
	Tools        []ToolDefinition
	Executor     ToolExecutor
	MaxTurns     int
}

type HybridTurn struct {
	Index     int        `json:"index"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Content   string     `json:"content,omitempty"`
}

type HybridResult struct {
	Answer                   string       `json:"answer"`
	Turns                    []HybridTurn `json:"turns"`
	ToolCalls                int          `json:"tool_calls"`
	ToolOutputBytes          int          `json:"tool_output_bytes"`
	ToolOutputTokenEq        int          `json:"tool_output_token_eq"`
	PromptTokensReported     int          `json:"prompt_tokens_reported"`
	CompletionTokensReported int          `json:"completion_tokens_reported"`
}

type hybridChatRequest struct {
	Model       string           `json:"model"`
	Messages    []map[string]any `json:"messages"`
	Temperature float64          `json:"temperature"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
}

type hybridChatResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Role      string     `json:"role"`
			Content   *string    `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// CompleteHybrid executes the OpenAI-compatible multimodal tool loop used by
// LM Studio and other compatible servers. Provider-specific payload details
// are delegated to MultimodalCompatibilityStrategy.
func (c OpenAICompat) CompleteHybrid(ctx context.Context, input HybridInput) (HybridResult, error) {
	if c.Model == "" {
		return HybridResult{}, fmt.Errorf("model is required")
	}
	if input.Executor == nil {
		return HybridResult{}, fmt.Errorf("tool executor is required")
	}
	if len(input.ImagePNG) == 0 {
		return HybridResult{}, fmt.Errorf("carrier image is required")
	}
	if input.MaxTurns <= 0 {
		input.MaxTurns = 16
	}
	if len(input.Tools) == 0 {
		return HybridResult{}, fmt.Errorf("at least one declared tool is required")
	}

	imageURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(input.ImagePNG)
	imagePart := c.multimodalCompatibility().ImageURLPart(imageURL)
	messages := []map[string]any{
		{"role": "system", "content": input.SystemPrompt},
		{"role": "user", "content": []map[string]any{
			{"type": "text", "text": input.Question},
			{"type": "image_url", "image_url": imagePart},
		}},
	}
	result := HybridResult{}

	for turn := 0; turn < input.MaxTurns; turn++ {
		response, err := c.hybridChat(ctx, messages, input.Tools)
		if err != nil {
			return result, err
		}
		result.PromptTokensReported += response.Usage.PromptTokens
		result.CompletionTokensReported += response.Usage.CompletionTokens
		if len(response.Choices) == 0 {
			return result, fmt.Errorf("target returned no choices")
		}
		choice := response.Choices[0]
		content := ""
		if choice.Message.Content != nil {
			content = strings.TrimSpace(*choice.Message.Content)
		}
		result.Turns = append(result.Turns, HybridTurn{Index: turn, ToolCalls: choice.Message.ToolCalls, Content: content})

		if len(choice.Message.ToolCalls) == 0 {
			if content == "" {
				return result, fmt.Errorf("target returned neither content nor tool calls")
			}
			result.Answer = content
			return result, nil
		}

		assistant := map[string]any{"role": "assistant", "tool_calls": choice.Message.ToolCalls}
		if content != "" {
			assistant["content"] = content
		}
		messages = append(messages, assistant)

		for _, call := range choice.Message.ToolCalls {
			if call.ID == "" || call.Function.Name == "" {
				return result, fmt.Errorf("invalid tool call")
			}
			args := json.RawMessage(call.Function.Arguments)
			if len(args) == 0 {
				args = json.RawMessage(`{}`)
			}
			var probe any
			if err := json.Unmarshal(args, &probe); err != nil {
				return result, fmt.Errorf("tool %s arguments: %w", call.Function.Name, err)
			}
			toolResult, err := input.Executor.Execute(ctx, call.Function.Name, args)
			if err != nil {
				return result, fmt.Errorf("execute tool %s: %w", call.Function.Name, err)
			}
			result.ToolCalls++
			result.ToolOutputBytes += len(toolResult)
			messages = append(messages, map[string]any{
				"role": "tool", "tool_call_id": call.ID, "name": call.Function.Name, "content": toolResult,
			})
		}
		result.ToolOutputTokenEq = (result.ToolOutputBytes + 3) / 4
	}
	return result, fmt.Errorf("Hybrid tool loop exceeded max turns %d", input.MaxTurns)
}

func (c OpenAICompat) hybridChat(ctx context.Context, messages []map[string]any, tools []ToolDefinition) (hybridChatResponse, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = "http://127.0.0.1:1234/v1"
	}
	client := c.httpClient()
	body, err := json.Marshal(hybridChatRequest{Model: c.Model, Messages: messages, Temperature: c.Temperature, Tools: tools})
	if err != nil {
		return hybridChatResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return hybridChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return hybridChatResponse{}, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return hybridChatResponse{}, err
	}
	if resp.StatusCode/100 != 2 {
		return hybridChatResponse{}, fmt.Errorf("target status %s: %s", resp.Status, string(b))
	}
	var out hybridChatResponse
	if err := json.Unmarshal(b, &out); err != nil {
		return hybridChatResponse{}, err
	}
	return out, nil
}
