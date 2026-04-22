package openai

import (
	"sort"
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

func (s *responsesStreamState) handleFunctionCallAdded(item *ResponsesItem) {
	if item == nil || item.Type != "function_call" {
		return
	}

	if s.spinner != nil {
		s.spinner.Stop()
		s.spinner.Start(ui.SpinnerMessageForTool(item.Name))
	}
	acc, exists := s.functionCalls[item.CallID]
	if !exists {
		acc = &responsesFunctionCallAccumulator{
			CallID: item.CallID,
			Name:   item.Name,
		}
		s.functionCalls[item.CallID] = acc
		s.callOrder = append(s.callOrder, item.CallID)
		return
	}
	if item.Name != "" {
		acc.Name = item.Name
	}
}

func (s *responsesStreamState) handleFunctionCallArgumentsDelta(chunk ResponsesStreamChunk) {
	callID := ""
	if chunk.Item != nil {
		callID = chunk.Item.CallID
	}

	if callID != "" {
		if acc, ok := s.functionCalls[callID]; ok {
			acc.Arguments.WriteString(chunk.Delta)
		}
		return
	}

	if len(s.functionCalls) != 1 {
		return
	}
	for _, acc := range s.functionCalls {
		acc.Arguments.WriteString(chunk.Delta)
		return
	}
}

func (s *responsesStreamState) handleFunctionCallArgumentsDone(chunk ResponsesStreamChunk) {
	if chunk.Item == nil {
		return
	}
	acc, ok := s.functionCalls[chunk.Item.CallID]
	if !ok || chunk.Item.Arguments == "" {
		return
	}
	acc.Arguments.Reset()
	acc.Arguments.WriteString(chunk.Item.Arguments)
}

func (s *responsesStreamState) appendFunctionCallsToOutput() {
	emitted := make(map[string]struct{}, len(s.functionCalls))
	for _, callID := range s.callOrder {
		acc, ok := s.functionCalls[callID]
		if !ok {
			continue
		}
		s.appendFunctionCallToolJSON(acc)
		emitted[callID] = struct{}{}
	}

	// フォールバック: 順序情報のない call は call_id 昇順で安定出力する。
	if len(emitted) == len(s.functionCalls) {
		return
	}
	remaining := make([]string, 0, len(s.functionCalls)-len(emitted))
	for callID := range s.functionCalls {
		if _, ok := emitted[callID]; ok {
			continue
		}
		remaining = append(remaining, callID)
	}
	sort.Strings(remaining)
	for _, callID := range remaining {
		acc := s.functionCalls[callID]
		s.appendFunctionCallToolJSON(acc)
	}
}

func (s *responsesStreamState) appendFunctionCallToolJSON(acc *responsesFunctionCallAccumulator) {
	if acc == nil {
		return
	}
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
