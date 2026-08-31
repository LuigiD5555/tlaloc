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

type PerceptionResult struct {
	Content                  string `json:"content"`
	PromptTokensReported     int    `json:"prompt_tokens_reported"`
	CompletionTokensReported int    `json:"completion_tokens_reported"`
}

// CompletePerception sends only the declared system prompt, user question and
// one visual carrier. It intentionally exposes no private evaluator manifest,
// expected probe bits, canonical decode or registry data.
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
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = "http://127.0.0.1:1234/v1"
	}
	client := c.httpClient()

	imageURL := "data:" + input.MediaType + ";base64," + base64.StdEncoding.EncodeToString(input.Image)
	imagePart := c.multimodalCompatibility().ImageURLPart(imageURL)
	messages := []map[string]any{
		{"role": "system", "content": input.SystemPrompt},
		{"role": "user", "content": []map[string]any{
			{"type": "text", "text": input.Question},
			{"type": "image_url", "image_url": imagePart},
		}},
	}
	body, err := json.Marshal(perceptionRequest{Model: c.Model, Messages: messages, Temperature: c.Temperature})
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
	resp, err := client.Do(req)
	if err != nil {
		return PerceptionResult{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return PerceptionResult{}, err
	}
	if resp.StatusCode/100 != 2 {
		return PerceptionResult{}, fmt.Errorf("target status %s: %s", resp.Status, string(raw))
	}
	var out perceptionResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return PerceptionResult{}, err
	}
	if len(out.Choices) == 0 || out.Choices[0].Message.Content == nil {
		return PerceptionResult{}, fmt.Errorf("target returned no perception content")
	}
	content := strings.TrimSpace(*out.Choices[0].Message.Content)
	if content == "" {
		return PerceptionResult{}, fmt.Errorf("target returned empty perception content")
	}
	return PerceptionResult{Content: content, PromptTokensReported: out.Usage.PromptTokens, CompletionTokensReported: out.Usage.CompletionTokens}, nil
}
