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
		s.registerFunctionCallAliases(key, item, outputIndex)
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
	s.registerFunctionCallAliases(key, item, outputIndex)
}

func (s *responsesStreamState) functionCallReplayKey(item *Item, outputIndex *int) string {
	if item != nil && item.ID != "" {
		if key := s.functionKeysByItemID[item.ID]; key != "" {
			return key
		}
	}
	if outputIndex != nil {
		if key := s.functionKeysByOutputIndex[*outputIndex]; key != "" {
			return key
		}
	}
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
	key := s.functionCallKeyFromChunk(chunk)
	if key != "" {
		acc := s.functionCalls[key]
		s.registerFunctionCallChunkAliases(key, chunk)
		acc.Arguments.WriteString(chunk.Delta)
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
	key := s.functionCallKeyFromChunk(chunk)
	if key == "" {
		if len(s.functionCalls) != 1 {
			return
		}
		for onlyKey := range s.functionCalls {
			key = onlyKey
			break
		}
	}
	acc, ok := s.functionCalls[key]
	if !ok {
		return
	}
	s.registerFunctionCallChunkAliases(key, chunk)
	s.updateFunctionCallAccumulatorFromChunk(acc, chunk)
	arguments := chunk.Arguments
	if chunk.Item != nil && chunk.Item.Arguments != "" {
		arguments = chunk.Item.Arguments
	}
	if arguments == "" {
		return
	}
	acc.Arguments.Reset()
	acc.Arguments.WriteString(arguments)
}

func (s *responsesStreamState) functionCallKeyFromChunk(chunk StreamChunk) string {
	if chunk.Item != nil && chunk.Item.CallID != "" {
		if _, ok := s.functionCalls[chunk.Item.CallID]; ok {
			return chunk.Item.CallID
		}
	}
	if chunk.CallID != "" {
		if _, ok := s.functionCalls[chunk.CallID]; ok {
			return chunk.CallID
		}
	}
	if chunk.Item != nil && chunk.Item.ID != "" {
		if key := s.functionKeysByItemID[chunk.Item.ID]; key != "" {
			return key
		}
		if key := "item:" + chunk.Item.ID; s.functionCalls[key] != nil {
			return key
		}
	}
	if chunk.ItemID != "" {
		if key := s.functionKeysByItemID[chunk.ItemID]; key != "" {
			return key
		}
		if key := "item:" + chunk.ItemID; s.functionCalls[key] != nil {
			return key
		}
	}
	if chunk.OutputIndex != nil {
		if key := s.functionKeysByOutputIndex[*chunk.OutputIndex]; key != "" {
			return key
		}
		if key := "index:" + strconv.Itoa(*chunk.OutputIndex); s.functionCalls[key] != nil {
			return key
		}
	}
	return ""
}

func (s *responsesStreamState) registerFunctionCallAliases(key string, item *Item, outputIndex *int) {
	if key == "" {
		return
	}
	if item != nil && item.ID != "" {
		s.functionKeysByItemID[item.ID] = key
	}
	if outputIndex != nil {
		s.functionKeysByOutputIndex[*outputIndex] = key
	}
}

func (s *responsesStreamState) registerFunctionCallChunkAliases(key string, chunk StreamChunk) {
	if chunk.ItemID != "" {
		s.functionKeysByItemID[chunk.ItemID] = key
	}
	s.registerFunctionCallAliases(key, chunk.Item, chunk.OutputIndex)
}

func (s *responsesStreamState) updateFunctionCallAccumulatorFromChunk(acc *responsesFunctionCallAccumulator, chunk StreamChunk) {
	if chunk.ItemID != "" && acc.ID == "" {
		acc.ID = chunk.ItemID
	}
	if chunk.CallID != "" {
		acc.CallID = chunk.CallID
	}
	if chunk.Name != "" {
		acc.Name = chunk.Name
	}
	if chunk.Item == nil {
		return
	}
	if chunk.Item.ID != "" {
		acc.ID = chunk.Item.ID
	}
	if chunk.Item.CallID != "" {
		acc.CallID = chunk.Item.CallID
	}
	if chunk.Item.Name != "" {
		acc.Name = chunk.Item.Name
	}
	if chunk.Item.Status != "" {
		acc.Status = chunk.Item.Status
	}
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
