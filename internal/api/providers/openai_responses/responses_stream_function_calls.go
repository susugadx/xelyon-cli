package openairesponses

import (
	"sort"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// responsesFunctionCallAccumulator は Responses API の function_call を累積
type responsesFunctionCallAccumulator struct {
	ID        string
	CallID    string
	Name      string
	Status    string
	Arguments strings.Builder
}

func (s *responsesStreamState) handleFunctionCallAdded(item *Item, outputIndex *int) {
	if item == nil || item.Type != "function_call" {
		return
	}

	key := s.functionCallReplayKey(item, outputIndex)
	acc, exists := s.functionCalls[key]
	if !exists {
		acc = &responsesFunctionCallAccumulator{
			ID:     item.ID,
			CallID: item.CallID,
			Name:   item.Name,
			Status: item.Status,
		}
		s.functionCalls[key] = acc
		s.callOrder = append(s.callOrder, key)
		s.addReplayOrder(responsesReplayKindFunctionCall, key)
		return
	}
	if item.ID != "" {
		acc.ID = item.ID
	}
	if item.CallID != "" {
		acc.CallID = item.CallID
	}
	if item.Name != "" {
		acc.Name = item.Name
	}
	if item.Status != "" {
		acc.Status = item.Status
	}
}

func (s *responsesStreamState) functionCallReplayKey(item *Item, outputIndex *int) string {
	if item != nil && item.CallID != "" {
		return item.CallID
	}
	if item != nil && item.ID != "" {
		return "item:" + item.ID
	}
	if outputIndex != nil {
		return "index:" + strconv.Itoa(*outputIndex)
	}
	return "unknown"
}

func (s *responsesStreamState) handleFunctionCallArgumentsDelta(chunk StreamChunk) {
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

func (s *responsesStreamState) handleFunctionCallArgumentsDone(chunk StreamChunk) {
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
	if toolJSON, err := convertResponsesFunctionCallToToolJSON(acc.CallID, acc.Name, acc.Arguments.String()); err == nil {
		s.toolCallsOut.WriteString(toolJSON)
	}
}

func (s *responsesStreamState) openAIResponsesReplayItems() []api.InputItem {
	return s.orderedOpenAIResponsesReplayItems()
}

func (s *responsesStreamState) openAIResponsesFunctionCallReplayItems() []api.InputItem {
	if s == nil || len(s.functionCalls) == 0 {
		return nil
	}

	items := make([]api.InputItem, 0, len(s.functionCalls))
	emitted := make(map[string]struct{}, len(s.functionCalls))
	for _, callID := range s.callOrder {
		acc, ok := s.functionCalls[callID]
		if !ok {
			continue
		}
		items = append(items, acc.openAIResponsesReplayItem())
		emitted[callID] = struct{}{}
	}
	if len(emitted) == len(s.functionCalls) {
		return items
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
		items = append(items, s.functionCalls[callID].openAIResponsesReplayItem())
	}
	return items
}

func (acc *responsesFunctionCallAccumulator) openAIResponsesReplayItem() api.InputItem {
	if acc == nil {
		return api.InputItem{}
	}
	return api.InputItem{
		Type:      "function_call",
		ID:        acc.ID,
		CallID:    acc.CallID,
		Name:      acc.Name,
		Arguments: acc.Arguments.String(),
		Status:    acc.Status,
	}
}

func convertResponsesFunctionCallToToolJSON(callID, name, arguments string) (string, error) {
	tc := &api.OpenAIToolCall{
		ID:   callID,
		Type: "function",
		Function: api.OpenAIToolCallFunction{
			Name:      name,
			Arguments: arguments,
		},
	}
	return ConvertToolCallToToolJSON(tc)
}
