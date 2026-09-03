package target

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAICompat struct {
	BaseURL        string
	Model          string
	APIKey         string
	Client         *http.Client
	Temperature    float64
	Compatibility  MultimodalCompatibilityStrategy
	RequestTimeout time.Duration
	MaxTokens      int
	Observer       ModelTraceObserver
	Guard          GenerationGuard
}

type chatRequest struct {
	Model       string              `json:"model"`
	Messages    []map[string]string `json:"messages"`
	Temperature float64             `json:"temperature"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// TextResult is a text completion plus the provider-reported token usage,
// mirroring PerceptionResult so the campaign runner can record tokens on the
// text path as well as the perception path.
type TextResult struct {
	Content                  string
	PromptTokensReported     int
	CompletionTokensReported int
}

func (c OpenAICompat) httpClient(ctx context.Context) *http.Client {
	return httpClientForTimeout(ctx, c.Client, c.RequestTimeout)
}

// Complete returns just the completion text. It is a thin wrapper over
// CompleteText for callers that do not need token usage.
func (c OpenAICompat) Complete(ctx context.Context, systemPrompt, user string) (string, error) {
	result, err := c.CompleteText(ctx, systemPrompt, user)
	return result.Content, err
}

func (c OpenAICompat) CompleteText(ctx context.Context, systemPrompt, user string) (TextResult, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = "http://127.0.0.1:1234/v1"
	}
	if c.Model == "" {
		return TextResult{}, fmt.Errorf("model is required")
	}
	client := c.httpClient(ctx)
	body, _ := json.Marshal(chatRequest{Model: c.Model, Messages: []map[string]string{{"role": "system", "content": systemPrompt}, {"role": "user", "content": user}}, Temperature: c.Temperature, MaxTokens: c.MaxTokens})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return TextResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return TextResult{}, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return TextResult{}, err
	}
	if resp.StatusCode/100 != 2 {
		return TextResult{}, fmt.Errorf("target status %s: %s", resp.Status, string(b))
	}
	var out chatResponse
	if err := json.Unmarshal(b, &out); err != nil {
		return TextResult{}, err
	}
	if len(out.Choices) == 0 {
		return TextResult{}, fmt.Errorf("target returned no choices")
	}
	return TextResult{
		Content:                  strings.TrimSpace(out.Choices[0].Message.Content),
		PromptTokensReported:     out.Usage.PromptTokens,
		CompletionTokensReported: out.Usage.CompletionTokens,
	}, nil
}
