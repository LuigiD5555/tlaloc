package target

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type PerceptionInput struct {
	SystemPrompt string
	Question     string
	Image        []byte
	MediaType    string
}

type perceptionRequest struct {
	Model       string           `json:"model"`
	Messages    []map[string]any `json:"messages"`
	Temperature float64          `json:"temperature"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Stream      bool             `json:"stream,omitempty"`
}

type perceptionResponse struct {
	Choices []struct {
		Message struct {
			Content *string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type perceptionStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content *string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type PerceptionResult struct {
	Content                  string `json:"content"`
	PromptTokensReported     int    `json:"prompt_tokens_reported"`
	CompletionTokensReported int    `json:"completion_tokens_reported"`
}

// CompletePerception sends only the declared system prompt, user question and
// one visual carrier. It intentionally exposes no private evaluator manifest,
// expected probe bits, canonical decode or registry data. When an Observer or
// GenerationGuard is attached, the request uses OpenAI-compatible SSE
// streaming; deltas are accumulated into the same response used by scoring.
func (c OpenAICompat) CompletePerception(ctx context.Context, input PerceptionInput) (PerceptionResult, error) {
	if c.Model == "" {
		return PerceptionResult{}, fmt.Errorf("model is required")
	}
	if len(input.Image) == 0 {
		return PerceptionResult{}, fmt.Errorf("image is required")
	}
	if input.MediaType == "" {
		input.MediaType = "image/png"
	}
	if c.Observer == nil {
		c.Observer = ModelTraceObserverFromContext(ctx)
	}
	policy := GenerationPolicyFromContext(ctx)
	if c.MaxTokens <= 0 {
		c.MaxTokens = policy.MaxTokens
	}
	if c.Guard == nil {
		c.Guard = policy.Guard
	}
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = "http://127.0.0.1:1234/v1"
	}
	client := c.httpClient(ctx)

	imageURL := "data:" + input.MediaType + ";base64," + base64.StdEncoding.EncodeToString(input.Image)
	imagePart := c.multimodalCompatibility().ImageURLPart(imageURL)
	messages := []map[string]any{
		{"role": "system", "content": input.SystemPrompt},
		{"role": "user", "content": []map[string]any{
			{"type": "text", "text": input.Question},
			{"type": "image_url", "image_url": imagePart},
		}},
	}
	stream := c.Observer != nil || c.Guard != nil
	body, err := json.Marshal(perceptionRequest{Model: c.Model, Messages: messages, Temperature: c.Temperature, MaxTokens: c.MaxTokens, Stream: stream})
	if err != nil {
		return PerceptionResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return PerceptionResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	start := time.Now()
	c.observe(ModelTraceEvent{Type: TraceRequestStart, Model: c.Model, Question: input.Question})
	resp, err := client.Do(req)
	if err != nil {
		c.observe(ModelTraceEvent{Type: TraceRequestError, Model: c.Model, Elapsed: time.Since(start), Err: err})
		return PerceptionResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		if readErr != nil {
			c.observe(ModelTraceEvent{Type: TraceRequestError, Model: c.Model, Elapsed: time.Since(start), Err: readErr})
			return PerceptionResult{}, readErr
		}
		err := fmt.Errorf("target status %s: %s", resp.Status, string(raw))
		c.observe(ModelTraceEvent{Type: TraceRequestError, Model: c.Model, Elapsed: time.Since(start), Err: err})
		return PerceptionResult{}, err
	}

	if stream && strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		result, err := c.readPerceptionStream(resp.Body, start)
		if err != nil {
			c.observe(ModelTraceEvent{Type: TraceRequestError, Model: c.Model, Elapsed: time.Since(start), Err: err})
			return PerceptionResult{}, err
		}
		return result, nil
	}

	// Some compatible servers may ignore stream=true and return ordinary JSON.
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		c.observe(ModelTraceEvent{Type: TraceRequestError, Model: c.Model, Elapsed: time.Since(start), Err: err})
		return PerceptionResult{}, err
	}
	var out perceptionResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		c.observe(ModelTraceEvent{Type: TraceRequestError, Model: c.Model, Elapsed: time.Since(start), Err: err})
		return PerceptionResult{}, err
	}
	if len(out.Choices) == 0 || out.Choices[0].Message.Content == nil {
		err := fmt.Errorf("target returned no perception content")
		c.observe(ModelTraceEvent{Type: TraceRequestError, Model: c.Model, Elapsed: time.Since(start), Err: err})
		return PerceptionResult{}, err
	}
	content := *out.Choices[0].Message.Content
	if strings.TrimSpace(content) == "" {
		err := fmt.Errorf("target returned empty perception content")
		c.observe(ModelTraceEvent{Type: TraceRequestError, Model: c.Model, Elapsed: time.Since(start), Err: err})
		return PerceptionResult{}, err
	}
	if c.Observer != nil {
		c.observe(ModelTraceEvent{Type: TraceFirstDelta, Model: c.Model, Elapsed: time.Since(start)})
		c.observe(ModelTraceEvent{Type: TraceDelta, Model: c.Model, Delta: content})
	}
	if err := c.checkGeneration(content, start); err != nil {
		return PerceptionResult{}, err
	}
	c.observe(ModelTraceEvent{Type: TraceRequestDone, Model: c.Model, Elapsed: time.Since(start), Characters: len(content)})
	return PerceptionResult{Content: strings.TrimSpace(content), PromptTokensReported: out.Usage.PromptTokens, CompletionTokensReported: out.Usage.CompletionTokens}, nil
}

func (c OpenAICompat) readPerceptionStream(r io.Reader, start time.Time) (PerceptionResult, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 2<<20)
	var content strings.Builder
	promptTokens := 0
	completionTokens := 0
	first := true

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk perceptionStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return PerceptionResult{}, fmt.Errorf("decode streaming chunk: %w", err)
		}
		if chunk.Usage.PromptTokens != 0 {
			promptTokens = chunk.Usage.PromptTokens
		}
		if chunk.Usage.CompletionTokens != 0 {
			completionTokens = chunk.Usage.CompletionTokens
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content == nil || *choice.Delta.Content == "" {
				continue
			}
			delta := *choice.Delta.Content
			if first {
				c.observe(ModelTraceEvent{Type: TraceFirstDelta, Model: c.Model, Elapsed: time.Since(start)})
				first = false
			}
			content.WriteString(delta)
			c.observe(ModelTraceEvent{Type: TraceDelta, Model: c.Model, Delta: delta})
			if err := c.checkGeneration(content.String(), start); err != nil {
				return PerceptionResult{}, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return PerceptionResult{}, err
	}
	raw := content.String()
	if strings.TrimSpace(raw) == "" {
		return PerceptionResult{}, fmt.Errorf("target returned empty perception stream")
	}
	c.observe(ModelTraceEvent{Type: TraceRequestDone, Model: c.Model, Elapsed: time.Since(start), Characters: len(raw)})
	return PerceptionResult{Content: strings.TrimSpace(raw), PromptTokensReported: promptTokens, CompletionTokensReported: completionTokens}, nil
}

func (c OpenAICompat) checkGeneration(content string, start time.Time) error {
	if c.Guard == nil {
		return nil
	}
	if err := c.Guard.Check(content); err != nil {
		if degeneration, ok := AsGenerationDegeneration(err); ok && degeneration.Partial == "" {
			degeneration.Partial = content
		}
		c.observe(ModelTraceEvent{Type: TraceGuardTriggered, Model: c.Model, Elapsed: time.Since(start), Characters: len(content), Err: err})
		return err
	}
	return nil
}

func (c OpenAICompat) observe(event ModelTraceEvent) {
	if c.Observer != nil {
		c.Observer.Observe(event)
	}
}
