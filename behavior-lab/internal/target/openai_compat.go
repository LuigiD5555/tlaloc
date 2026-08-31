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
	BaseURL       string
	Model         string
	APIKey        string
	Client        *http.Client
	Temperature   float64
	Compatibility MultimodalCompatibilityStrategy
}

type chatRequest struct {
	Model       string              `json:"model"`
	Messages    []map[string]string `json:"messages"`
	Temperature float64             `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (c OpenAICompat) Complete(ctx context.Context, systemPrompt, user string) (string, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = "http://127.0.0.1:1234/v1"
	}
	if c.Model == "" {
		return "", fmt.Errorf("model is required")
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	body, _ := json.Marshal(chatRequest{Model: c.Model, Messages: []map[string]string{{"role": "system", "content": systemPrompt}, {"role": "user", "content": user}}, Temperature: c.Temperature})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("target status %s: %s", resp.Status, string(b))
	}
	var out chatResponse
	if err := json.Unmarshal(b, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("target returned no choices")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}
