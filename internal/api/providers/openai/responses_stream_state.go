package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// responsesFunctionCallAccumulator は Responses API の function_call を累積
type responsesFunctionCallAccumulator struct {
	CallID    string
	Name      string
	Arguments strings.Builder
}

type responsesStreamState struct {
	spinner       *ui.Spinner
	errOut        io.Writer
	debug         bool
	responseID    string
	functionCalls map[string]*responsesFunctionCallAccumulator
	toolCallsOut  strings.Builder
	lastUsage     *api.Usage
}

func newResponsesStreamState(spinner *ui.Spinner, errOut io.Writer) *responsesStreamState {
	return &responsesStreamState{
		spinner:       spinner,
		errOut:        errOut,
		debug:         os.Getenv("XELYON_DEBUG_OPENAI") == "1",
		functionCalls: make(map[string]*responsesFunctionCallAccumulator),
	}
}

func (s *responsesStreamState) parseLine(line string) (string, bool, error) {
	if s.debug && line != "" {
		s.debugf("[DEBUG OpenAI Responses] SSE line: %s\n", line)
	}

	data, done, handled := parseResponsesSSEDataLine(line)
	if !handled {
		return "", false, nil
	}
	if done {
		return "", true, nil
	}

	chunk, err := decodeResponsesStreamChunk(data)
	if err != nil {
		return "", false, nil // パースエラーはスキップ
	}

	return s.handleChunk(chunk, data)
}

func parseResponsesSSEDataLine(line string) (data string, done bool, handled bool) {
	if !strings.HasPrefix(line, "data: ") {
		return "", false, false
	}

	data = strings.TrimPrefix(line, "data: ")
	if data == "[DONE]" {
		return "", true, true
	}

	return data, false, true
}

func decodeResponsesStreamChunk(data string) (ResponsesStreamChunk, error) {
	var chunk ResponsesStreamChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return ResponsesStreamChunk{}, err
	}
	return chunk, nil
}

func (s *responsesStreamState) debugf(format string, args ...interface{}) {
	if !s.debug {
		return
	}
	fmt.Fprintf(s.errOut, format, args...)
}

func (s *responsesStreamState) handleChunk(chunk ResponsesStreamChunk, rawData string) (string, bool, error) {
	s.debugf("[DEBUG OpenAI Responses] event: %s\n", chunk.Type)
	if chunk.Type == "response.completed" {
		s.debugf("[DEBUG OpenAI Responses] raw data: %s\n", rawData)
	}

	if chunk.Type == "response.created" && chunk.Response != nil {
		s.responseID = chunk.Response.ID
	}

	if chunk.Type == "error" {
		errMsg := "OpenAI API error"
		if chunk.Error != nil {
			if chunk.Error.Message != "" {
				errMsg = chunk.Error.Message
			} else if chunk.Error.Code != "" {
				errMsg = fmt.Sprintf("OpenAI API error: %s", chunk.Error.Code)
			}
		}
		return "", true, fmt.Errorf("%s", errMsg)
	}

	if chunk.Type == "response.failed" {
		return "", true, fmt.Errorf("OpenAI Responses API request failed")
	}

	if chunk.Type == "response.output_item.added" && chunk.Item != nil && chunk.Item.Type == "function_call" {
		if s.spinner != nil {
			s.spinner.Stop()
			s.spinner.Start(ui.SpinnerMessageForTool(chunk.Item.Name))
		}
		acc := &responsesFunctionCallAccumulator{
			CallID: chunk.Item.CallID,
			Name:   chunk.Item.Name,
		}
		s.functionCalls[chunk.Item.CallID] = acc
	}

	if chunk.Type == "response.function_call_arguments.delta" {
		callID := ""
		if chunk.Item != nil {
			callID = chunk.Item.CallID
		}

		if callID != "" {
			if acc, ok := s.functionCalls[callID]; ok {
				acc.Arguments.WriteString(chunk.Delta)
			}
		} else if len(s.functionCalls) == 1 {
			for _, acc := range s.functionCalls {
				acc.Arguments.WriteString(chunk.Delta)
				break
			}
		}
	}

	if chunk.Type == "response.function_call_arguments.done" && chunk.Item != nil {
		if acc, ok := s.functionCalls[chunk.Item.CallID]; ok {
			if chunk.Item.Arguments != "" {
				acc.Arguments.Reset()
				acc.Arguments.WriteString(chunk.Item.Arguments)
			}
		}
	}

	if chunk.Type == "response.output_text.delta" {
		return chunk.Delta, false, nil
	}

	if chunk.Type == "response.completed" || chunk.Type == "response.done" {
		s.captureUsage(chunk)
		s.appendFunctionCallsToOutput()
		return "", true, nil
	}

	return "", false, nil
}

func (s *responsesStreamState) captureUsage(chunk ResponsesStreamChunk) {
	var usage *ResponsesUsage
	if chunk.Response != nil && chunk.Response.Usage != nil {
		usage = chunk.Response.Usage
	} else if chunk.Usage != nil {
		usage = chunk.Usage
	}

	if usage == nil {
		s.debugf("[DEBUG OpenAI Responses] %s event but usage is nil\n", chunk.Type)
		return
	}

	cachedTokens := 0
	if usage.InputTokensDetails != nil {
		cachedTokens = usage.InputTokensDetails.CachedTokens
	}
	reasoningTokens := 0
	if usage.OutputTokensDetails != nil {
		reasoningTokens = usage.OutputTokensDetails.ReasoningTokens
	}
	s.lastUsage = &api.Usage{
		InputTokens:       usage.InputTokens,
		OutputTokens:      usage.OutputTokens,
		ThinkingTokens:    reasoningTokens,
		CachedInputTokens: cachedTokens,
	}
	s.debugf("[DEBUG OpenAI Responses] usage received: input=%d, output=%d, cached=%d\n",
		usage.InputTokens, usage.OutputTokens, cachedTokens)
}

func (s *responsesStreamState) appendFunctionCallsToOutput() {
	for _, acc := range s.functionCalls {
		tc := &api.OpenAIToolCall{
			ID:   acc.CallID,
			Type: "function",
			Function: api.OpenAIToolCallFunction{
				Name:      acc.Name,
				Arguments: acc.Arguments.String(),
			},
		}
		if toolJSON, err := ConvertToolCallToToolJSON(tc); err == nil {
			s.toolCallsOut.WriteString(toolJSON)
		}
	}
}

// handleResponsesStreaming は Responses API のストリーミングを処理
// Response ID も抽出して返却（content, responseID, error）
func (p *Provider) handleResponsesStreaming(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, string, error) {
	state := newResponsesStreamState(spinner, api.ErrorWriterFromContext(ctx))
	content, err := api.ParseStreamingResponse(ctx, resp, spinner, state.parseLine)
	if err != nil {
		return "", state.responseID, err
	}

	// usage コールバックを呼び出し
	if state.lastUsage != nil && p.usageCallback != nil {
		p.usageCallback(*state.lastUsage)
	}

	// tool_calls がある場合はそれを返す
	if state.toolCallsOut.Len() > 0 {
		if content != "" {
			return content + state.toolCallsOut.String(), state.responseID, nil
		}
		return state.toolCallsOut.String(), state.responseID, nil
	}
	return content, state.responseID, nil
}
