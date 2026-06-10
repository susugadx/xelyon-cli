package openairesponses

import (
	"fmt"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

type responsesReplayRef struct {
	kind string
	key  string
}

type responsesMessageAccumulator struct {
	ID     string
	Status string
	Text   strings.Builder
}

const (
	responsesReplayKindMessage      = "message"
	responsesReplayKindReasoning    = "reasoning"
	responsesReplayKindFunctionCall = "function_call"
)

func (s *responsesStreamState) handleOutputItemAdded(chunk StreamChunk) {
	if chunk.Item == nil {
		return
	}
	switch chunk.Item.Type {
	case "message":
		s.captureMessageItem(chunk.Item, chunk.OutputIndex)
	case "reasoning":
		s.captureReasoningItem(chunk.Item, chunk.OutputIndex)
	case "function_call":
		s.handleFunctionCallAdded(chunk.Item, chunk.OutputIndex)
	}
}

func (s *responsesStreamState) handleOutputItemDone(chunk StreamChunk) {
	if chunk.Item == nil {
		return
	}
	switch chunk.Item.Type {
	case "message":
		s.captureMessageItem(chunk.Item, chunk.OutputIndex)
	case "reasoning":
		s.captureReasoningItem(chunk.Item, chunk.OutputIndex)
	case "function_call":
		s.handleFunctionCallAdded(chunk.Item, chunk.OutputIndex)
		if chunk.Item.Arguments != "" {
			s.handleFunctionCallArgumentsDone(chunk)
		}
	}
}

func (s *responsesStreamState) handleTextDelta(chunk StreamChunk) {
	s.textOut.WriteString(chunk.Delta)
	if chunk.Delta == "" {
		return
	}
	acc := s.ensureCurrentMessageAccumulator(chunk.OutputIndex)
	acc.Text.WriteString(chunk.Delta)
}

func (s *responsesStreamState) captureMessageItem(item *Item, outputIndex *int) {
	key := s.messageReplayKey(item, outputIndex)
	acc, exists := s.messages[key]
	if !exists {
		acc = &responsesMessageAccumulator{}
		s.messages[key] = acc
	}
	if item.ID != "" {
		acc.ID = item.ID
	}
	if item.Status != "" {
		acc.Status = item.Status
	}
	if outputIndex != nil {
		s.messageKeysByOutputIndex[*outputIndex] = key
	}
	s.currentMessageKey = key
	s.addReplayOrder(responsesReplayKindMessage, key)
}

func (s *responsesStreamState) captureReasoningItem(item *Item, outputIndex *int) {
	key := s.reasoningReplayKey(item, outputIndex)
	existing := s.reasoningItems[key]
	if item.ID != "" {
		existing.ID = item.ID
	}
	if item.Status != "" {
		existing.Status = item.Status
	}
	if len(item.Summary) > 0 {
		existing.Summary = cloneResponsesSummary(item.Summary)
	}
	if item.EncryptedContent != "" {
		existing.EncryptedContent = item.EncryptedContent
	}
	existing.Type = "reasoning"
	s.reasoningItems[key] = existing
	s.addReplayOrder(responsesReplayKindReasoning, key)
}

func (s *responsesStreamState) ensureCurrentMessageAccumulator(outputIndex *int) *responsesMessageAccumulator {
	if outputIndex != nil {
		if key := s.messageKeysByOutputIndex[*outputIndex]; key != "" {
			s.currentMessageKey = key
			return s.messages[key]
		}
		key := fmt.Sprintf("message:index:%d", *outputIndex)
		acc := s.ensureMessageAccumulator(key)
		s.messageKeysByOutputIndex[*outputIndex] = key
		s.currentMessageKey = key
		s.addReplayOrder(responsesReplayKindMessage, key)
		return acc
	}
	if s.currentMessageKey != "" {
		return s.messages[s.currentMessageKey]
	}
	key := "message:default"
	acc := s.ensureMessageAccumulator(key)
	s.currentMessageKey = key
	s.addReplayOrder(responsesReplayKindMessage, key)
	return acc
}

func (s *responsesStreamState) ensureMessageAccumulator(key string) *responsesMessageAccumulator {
	if acc, ok := s.messages[key]; ok {
		return acc
	}
	acc := &responsesMessageAccumulator{}
	s.messages[key] = acc
	return acc
}

func (s *responsesStreamState) messageReplayKey(item *Item, outputIndex *int) string {
	if item != nil && item.ID != "" {
		return "message:id:" + item.ID
	}
	if outputIndex != nil {
		return fmt.Sprintf("message:index:%d", *outputIndex)
	}
	return "message:default"
}

func (s *responsesStreamState) reasoningReplayKey(item *Item, outputIndex *int) string {
	if item != nil && item.ID != "" {
		return "reasoning:id:" + item.ID
	}
	if outputIndex != nil {
		return fmt.Sprintf("reasoning:index:%d", *outputIndex)
	}
	return fmt.Sprintf("reasoning:auto:%d", len(s.reasoningItems))
}

func (s *responsesStreamState) addReplayOrder(kind, key string) {
	if key == "" {
		return
	}
	seenKey := kind + "\x00" + key
	if _, ok := s.replayOrderSeen[seenKey]; ok {
		return
	}
	s.replayOrderSeen[seenKey] = struct{}{}
	s.replayOrder = append(s.replayOrder, responsesReplayRef{kind: kind, key: key})
}

func (s *responsesStreamState) orderedOpenAIResponsesReplayItems() []api.InputItem {
	if s == nil {
		return nil
	}
	if len(s.replayOrder) == 0 {
		return s.legacyOpenAIResponsesReplayItems()
	}

	items := make([]api.InputItem, 0, len(s.replayOrder))
	emittedFunctions := make(map[string]struct{}, len(s.functionCalls))
	for _, ref := range s.replayOrder {
		switch ref.kind {
		case responsesReplayKindMessage:
			if item, ok := s.messageReplayItem(ref.key); ok {
				items = append(items, item)
			}
		case responsesReplayKindReasoning:
			if item, ok := s.reasoningItems[ref.key]; ok {
				items = append(items, item)
			}
		case responsesReplayKindFunctionCall:
			if acc, ok := s.functionCalls[ref.key]; ok {
				items = append(items, acc.openAIResponsesReplayItem())
				emittedFunctions[ref.key] = struct{}{}
			}
		}
	}
	items = append(items, s.unorderedFunctionReplayItems(emittedFunctions)...)
	return items
}

func (s *responsesStreamState) messageReplayItem(key string) (api.InputItem, bool) {
	acc, ok := s.messages[key]
	if !ok {
		return api.InputItem{}, false
	}
	text := acc.Text.String()
	if text == "" {
		return api.InputItem{}, false
	}
	return api.InputItem{
		Type:    "message",
		Role:    "assistant",
		ID:      acc.ID,
		Status:  acc.Status,
		Content: text,
	}, true
}

func (s *responsesStreamState) legacyOpenAIResponsesReplayItems() []api.InputItem {
	items := make([]api.InputItem, 0, len(s.functionCalls)+1)
	if text := s.textOut.String(); text != "" {
		items = append(items, api.InputItem{
			Type:    "message",
			Role:    "assistant",
			Content: text,
		})
	}
	items = append(items, s.openAIResponsesFunctionCallReplayItems()...)
	return items
}

func (s *responsesStreamState) unorderedFunctionReplayItems(emitted map[string]struct{}) []api.InputItem {
	if len(emitted) == len(s.functionCalls) {
		return nil
	}
	items := make([]api.InputItem, 0, len(s.functionCalls)-len(emitted))
	keys := make([]string, 0, len(s.functionCalls)-len(emitted))
	for key := range s.functionCalls {
		if _, ok := emitted[key]; ok {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		items = append(items, s.functionCalls[key].openAIResponsesReplayItem())
	}
	return items
}

func cloneResponsesSummary(src []map[string]any) []map[string]any {
	if len(src) == 0 {
		return nil
	}
	out := make([]map[string]any, len(src))
	for i, entry := range src {
		if entry == nil {
			continue
		}
		out[i] = make(map[string]any, len(entry))
		for k, v := range entry {
			out[i][k] = v
		}
	}
	return out
}
