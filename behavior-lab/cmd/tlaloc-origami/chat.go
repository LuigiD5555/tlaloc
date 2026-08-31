package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"tlaloc.local/behaviorlab/internal/target"
)

func chatCmd(args []string) {
	fs := flag.NewFlagSet("chat", flag.ExitOnError)
	carrier := fs.String("carrier", "origami.png", "carrier PNG")
	store := fs.String("store", "store", "Tlaloc R2 memory plane")
	prompt := fs.String("prompt", "MASTER_PROMPT.txt", "Master Prompt R2")
	origamiBin := fs.String("origami-bin", "origami-fixed-carrier", "Origami fixed carrier decoder")
	base := fs.String("base", "http://127.0.0.1:1234/v1", "OpenAI-compatible base URL")
	model := fs.String("model", "", "model name")
	compatibilityName := fs.String("compatibility", target.CompatibilityLMStudio, "multimodal compatibility strategy: lm-studio|openai|minimal")
	apiKey := fs.String("api-key", "", "optional API key")
	question := fs.String("question", "", "question")
	turns := fs.Int("turns", 12, "max model/tool turns")
	toolMode := fs.String("tool-mode", "functions", "functions|text")
	_ = fs.Parse(args)
	if *model == "" || *question == "" {
		die(fmt.Errorf("-model and -question are required"))
	}
	compatibility, err := target.ResolveMultimodalCompatibility(*compatibilityName)
	die(err)
	image, err := os.ReadFile(*carrier)
	die(err)
	system, err := os.ReadFile(*prompt)
	die(err)
	ex := target.FixedOrigamiExecutor{OrigamiBinary: *origamiBin, Carrier: *carrier, StoreDir: *store}
	client := target.OpenAICompat{BaseURL: *base, Model: *model, APIKey: *apiKey, Temperature: 0, Compatibility: compatibility}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	input := target.HybridInput{SystemPrompt: string(system), Question: *question, ImagePNG: image, Tools: target.OrigamiFixedTools(), Executor: ex, MaxTurns: *turns}
	var result target.HybridResult
	if *toolMode == "text" {
		result, err = client.CompleteHybridTextBridge(ctx, input)
	} else if *toolMode == "functions" {
		result, err = client.CompleteHybrid(ctx, input)
	} else {
		die(fmt.Errorf("unsupported -tool-mode %q", *toolMode))
	}
	die(err)
	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b))
}
