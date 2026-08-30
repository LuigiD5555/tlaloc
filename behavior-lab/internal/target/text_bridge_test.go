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

type textMockExec struct{ calls int }

func (m *textMockExec) Execute(_ context.Context, name string, args json.RawMessage) (string, error) {
	m.calls++
	if name != "origami_boot" {
		return "", fmt.Errorf("bad tool %s", name)
	}
	return `{"status":"BOOT_OK"}`, nil
}
func TestCompleteHybridTextBridge(t *testing.T) {
	requests := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		defer r.Body.Close()
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		msgs := body["messages"].([]any)
		if requests == 1 {
			user := msgs[1].(map[string]any)
			parts := user["content"].([]any)
			image := parts[1].(map[string]any)["image_url"].(map[string]any)["url"].(string)
			if !strings.HasPrefix(image, "data:image/png;base64,") {
				t.Fatal("image missing")
			}
			fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"<ORIGAMI_CALL>{\"name\":\"origami_boot\",\"arguments\":{}}</ORIGAMI_CALL>"}}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`)
			return
		}
		fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"ANSWER: ready\nSTATUS: SEMANTIC"}}],"usage":{"prompt_tokens":20,"completion_tokens":5}}`)
	}))
	defer s.Close()
	ex := &textMockExec{}
	c := OpenAICompat{BaseURL: s.URL + "/v1", Model: "fixture"}
	r, err := c.CompleteHybridTextBridge(context.Background(), HybridInput{SystemPrompt: "boot", Question: "test", ImagePNG: []byte("png"), Executor: ex, MaxTurns: 3})
	if err != nil {
		t.Fatal(err)
	}
	if ex.calls != 1 || r.ToolCalls != 1 || requests != 2 {
		t.Fatalf("bad counts %+v calls=%d requests=%d", r, ex.calls, requests)
	}
	if !strings.Contains(r.Answer, "ready") {
		t.Fatalf("bad answer %s", r.Answer)
	}
}
