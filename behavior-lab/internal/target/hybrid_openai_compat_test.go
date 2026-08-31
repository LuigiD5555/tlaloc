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

type mockExecutor struct{ calls int }

func (m *mockExecutor) Execute(_ context.Context, name string, arguments json.RawMessage) (string, error) {
	m.calls++
	if name != "origami_follow" {
		return "", fmt.Errorf("unexpected tool %s", name)
	}
	var args struct {
		Query    string `json:"query"`
		Relation string `json:"relation"`
		Depth    int    `json:"depth"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return "", err
	}
	if args.Query != "K7F91" || args.Relation != "depends" || args.Depth != 2 {
		return "", fmt.Errorf("unexpected args %+v", args)
	}
	return `{"result":{"operation":"FOLLOW","entries":[{"value":"AMBER-10593"}],"evidence":"carrier:path:depends"}}`, nil
}

func TestCompleteHybridUsesLMStudioStrategyExecutesToolAndReturnsAnswer(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		defer r.Body.Close()
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		messages, ok := body["messages"].([]any)
		if !ok || len(messages) < 2 {
			t.Fatalf("unexpected messages: %#v", body["messages"])
		}
		if requests == 1 {
			if _, ok := body["tools"].([]any); !ok {
				t.Fatalf("tools missing from first request: %#v", body)
			}
			user := messages[1].(map[string]any)
			content := user["content"].([]any)
			if len(content) != 2 {
				t.Fatalf("unexpected multimodal content: %#v", content)
			}
			imageURL := content[1].(map[string]any)["image_url"].(map[string]any)
			image := imageURL["url"].(string)
			if !strings.HasPrefix(image, "data:image/png;base64,") {
				t.Fatalf("missing image data URL: %s", image)
			}
			if imageURL["detail"] != "high" {
				t.Fatalf("LM Studio strategy detail=%#v, want high", imageURL["detail"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","content":null,"tool_calls":[{"id":"call-1","type":"function","function":{"name":"origami_follow","arguments":"{\"query\":\"K7F91\",\"relation\":\"depends\",\"depth\":2}"}}]}}],"usage":{"prompt_tokens":100,"completion_tokens":20}}`)
			return
		}
		foundToolResult := false
		for _, raw := range messages {
			msg := raw.(map[string]any)
			if msg["role"] == "tool" {
				foundToolResult = true
				if msg["tool_call_id"] != "call-1" {
					t.Fatalf("wrong tool_call_id: %#v", msg)
				}
			}
		}
		if !foundToolResult {
			t.Fatalf("second request missing tool result: %#v", messages)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"ANSWER: AMBER-10593\nSTATUS: SEMANTIC","tool_calls":[]}}],"usage":{"prompt_tokens":130,"completion_tokens":15}}`)
	}))
	defer server.Close()

	compatibility, err := ResolveMultimodalCompatibility(CompatibilityLMStudio)
	if err != nil {
		t.Fatal(err)
	}
	executor := &mockExecutor{}
	client := OpenAICompat{BaseURL: server.URL + "/v1", Model: "fixture-vlm", Compatibility: compatibility}
	result, err := client.CompleteHybrid(context.Background(), HybridInput{
		SystemPrompt: "Find BOOT first.",
		Question:     "What is the second-order dependency of K7F91?",
		ImagePNG:     []byte("png-fixture"),
		Tools:        OrigamiHybridTools(),
		Executor:     executor,
		MaxTurns:     4,
	})
	if err != nil {
		t.Fatal(err)
	}
	if executor.calls != 1 || result.ToolCalls != 1 || requests != 2 {
		t.Fatalf("unexpected loop counts executor=%d tools=%d requests=%d", executor.calls, result.ToolCalls, requests)
	}
	if !strings.Contains(result.Answer, "AMBER-10593") {
		t.Fatalf("unexpected answer: %s", result.Answer)
	}
	if result.PromptTokensReported != 230 || result.CompletionTokensReported != 35 {
		t.Fatalf("unexpected usage: %+v", result)
	}
	if result.ToolOutputBytes == 0 || result.ToolOutputTokenEq == 0 {
		t.Fatalf("missing tool cost: %+v", result)
	}
}
