package target

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const textBridgeSuffix = `

PLAIN-TEXT TOOL BRIDGE — R2
If native function/tool calling is unavailable, request exactly one Origami operation by outputting one ORIGAMI_CALL envelope and nothing else. Before origami_boot, read T0 from the image and independently read BOTH T1 probe rows. Pass visual_probe_top, visual_probe_bottom and truthful host capabilities. Never guess a probe and never fabricate ORIGAMI_TOOL_RESULT. Valid names are only the tools declared by the receiver prompt.`

// CompleteHybridTextBridge provides a fallback for OpenAI-compatible multimodal
// models that cannot emit function-tool calls. Tlaloc intercepts explicit
// <ORIGAMI_CALL> JSON envelopes and executes them through the same ToolExecutor.
func (c OpenAICompat) CompleteHybridTextBridge(ctx context.Context, input HybridInput) (HybridResult, error) {
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
	imageURL := "data:image/png;base64," + base64Encode(input.ImagePNG)
	messages := []map[string]any{{"role": "system", "content": input.SystemPrompt + textBridgeSuffix}, {"role": "user", "content": []map[string]any{{"type": "text", "text": input.Question}, {"type": "image_url", "image_url": map[string]string{"url": imageURL, "detail": "original"}}}}}
	result := HybridResult{}
	for turn := 0; turn < input.MaxTurns; turn++ {
		response, err := c.hybridChat(ctx, messages, nil)
		if err != nil {
			return result, err
		}
		result.PromptTokensReported += response.Usage.PromptTokens
		result.CompletionTokensReported += response.Usage.CompletionTokens
		if len(response.Choices) == 0 {
			return result, fmt.Errorf("target returned no choices")
		}
		content := ""
		if response.Choices[0].Message.Content != nil {
			content = strings.TrimSpace(*response.Choices[0].Message.Content)
		}
		result.Turns = append(result.Turns, HybridTurn{Index: turn, Content: content})
		call, ok, err := parseTextCall(content)
		if err != nil {
			return result, err
		}
		if !ok {
			if content == "" {
				return result, fmt.Errorf("target returned empty content")
			}
			result.Answer = content
			return result, nil
		}
		args := call.Arguments
		if len(args) == 0 {
			args = json.RawMessage(`{}`)
		}
		toolResult, err := input.Executor.Execute(ctx, call.Name, args)
		if err != nil {
			return result, fmt.Errorf("execute tool %s: %w", call.Name, err)
		}
		result.ToolCalls++
		result.ToolOutputBytes += len(toolResult)
		result.ToolOutputTokenEq = (result.ToolOutputBytes + 3) / 4
		messages = append(messages, map[string]any{"role": "assistant", "content": content}, map[string]any{"role": "user", "content": "<ORIGAMI_TOOL_RESULT name=\"" + call.Name + "\">" + toolResult + "</ORIGAMI_TOOL_RESULT>"})
	}
	return result, fmt.Errorf("Hybrid text tool loop exceeded max turns %d", input.MaxTurns)
}

type textCall struct {
	Name      string
	Arguments json.RawMessage
}

func parseTextCall(content string) (textCall, bool, error) {
	const open = "<ORIGAMI_CALL>"
	const close = "</ORIGAMI_CALL>"
	start := strings.Index(content, open)
	if start < 0 {
		return textCall{}, false, nil
	}
	end := strings.Index(content[start+len(open):], close)
	if end < 0 {
		return textCall{}, false, fmt.Errorf("unterminated ORIGAMI_CALL")
	}
	payload := strings.TrimSpace(content[start+len(open) : start+len(open)+end])
	var raw struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return textCall{}, false, fmt.Errorf("invalid ORIGAMI_CALL: %w", err)
	}
	if raw.Name == "" {
		return textCall{}, false, fmt.Errorf("ORIGAMI_CALL missing name")
	}
	return textCall{Name: raw.Name, Arguments: raw.Arguments}, true, nil
}
