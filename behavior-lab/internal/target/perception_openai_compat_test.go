package target

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompletePerceptionUsesSupportedImageDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		messages, ok := body["messages"].([]any)
		if !ok || len(messages) != 2 {
			t.Fatalf("unexpected messages: %#v", body["messages"])
		}
		user := messages[1].(map[string]any)
		content := user["content"].([]any)
		if len(content) != 2 {
			t.Fatalf("unexpected multimodal content: %#v", content)
		}
		imageURL := content[1].(map[string]any)["image_url"].(map[string]any)
		if detail := imageURL["detail"]; detail != "high" {
			t.Fatalf("unsupported image detail: %#v", detail)
		}
		url, _ := imageURL["url"].(string)
		if !strings.HasPrefix(url, "data:image/png;base64,") {
			t.Fatalf("missing PNG data URL: %q", url)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"VISION_OK"}}],"usage":{"prompt_tokens":12,"completion_tokens":3}}`)
	}))
	defer server.Close()

	client := OpenAICompat{BaseURL: server.URL + "/v1", Model: "fixture-vlm", Temperature: 0}
	result, err := client.CompletePerception(context.Background(), PerceptionInput{
		SystemPrompt: "",
		Question:     "Inspect the image.",
		Image:        []byte("png-fixture"),
		MediaType:    "image/png",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "VISION_OK" {
		t.Fatalf("unexpected content: %q", result.Content)
	}
	if result.PromptTokensReported != 12 || result.CompletionTokensReported != 3 {
		t.Fatalf("unexpected usage: %+v", result)
	}
}
