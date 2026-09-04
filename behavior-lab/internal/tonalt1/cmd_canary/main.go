// Command t1-infra-canary performs exactly ONE non-T1, infrastructure-only
// live call against LM Studio to verify OpenAICompat -> /chat/completions ->
// lfm2-vl-1.6b -> non-empty valid response, using a synthetic image that is
// NOT part of any T1 workflow, operand, bridge page, or gold benchmark.
//
// This call is recorded separately as T1_INFRA_CANARY_CALLS = 1 and does not
// count toward the T1 primary or counterfactual call budgets.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"tlaloc.local/behaviorlab/internal/target"
)

func main() {
	imgPath := "/tmp/t1_canary_synthetic.png"
	imgBytes, err := os.ReadFile(imgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CANARY FAILED: cannot read synthetic image: %v\n", err)
		os.Exit(1)
	}

	client := target.OpenAICompat{
		BaseURL:     "http://127.0.0.1:1234/v1",
		Model:       "lfm2-vl-1.6b",
		Temperature: 0,
		MaxTokens:   32,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	start := time.Now()
	result, err := client.CompletePerception(ctx, target.PerceptionInput{
		SystemPrompt: "",
		Question:     "What digit is shown in this image? Answer with only the digit.",
		Image:        imgBytes,
		MediaType:    "image/png",
	})
	elapsed := time.Since(start)

	report := map[string]interface{}{
		"T1_INFRA_CANARY_CALLS": 1,
		"purpose":               "infrastructure_only_non_T1",
		"endpoint":              "http://127.0.0.1:1234/v1/chat/completions",
		"model":                 "lfm2-vl-1.6b",
		"image_source":          "synthetic_non_primary_non_gold",
		"image_bytes_len":       len(imgBytes),
		"elapsed_ms":            elapsed.Milliseconds(),
		"timestamp":             time.Now().UTC().Format(time.RFC3339),
	}

	if err != nil {
		report["status"] = "FAIL"
		report["error"] = err.Error()
		out, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(out))
		os.Exit(1)
	}

	report["status"] = "PASS"
	report["raw_content"] = result.Content
	report["prompt_tokens_reported"] = result.PromptTokensReported
	report["completion_tokens_reported"] = result.CompletionTokensReported
	report["non_empty_valid_response"] = result.Content != ""

	out, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(out))
}
