package history

import (
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// ToAPIMessages はAPI形式に変換
func (s *Session) ToAPIMessages() []api.Message {
	msgs := make([]api.Message, 0, len(s.Messages))
	for _, m := range s.Messages {
		if !isConversationMessageEntry(m) {
			continue
		}
		msg := api.Message{
			Role:             m.Role,
			Content:          m.Content,
			ReasoningContent: m.ReasoningContent,
			ToolCalls:        m.ToolCalls,
			ToolCallID:       m.ToolCallID,
			ToolName:         m.ToolName,
		}
		if m.ProviderMetadata != nil {
			if len(m.ProviderMetadata.AnthropicContentBlocks) > 0 {
				msg.SetAnthropicContentBlocks(m.ProviderMetadata.AnthropicContentBlocks)
			} else {
				msg.SetAnthropicThinkingBlocks(m.ProviderMetadata.AnthropicThinkingBlocks)
			}
		}
		msgs = append(msgs, msg)
	}
	return msgs
}

func providerMetadataFromAPIMessage(msg api.Message) *MessageProviderMetadata {
	contentBlocks := msg.AnthropicContentBlocks()
	if len(contentBlocks) > 0 {
		return &MessageProviderMetadata{
			AnthropicContentBlocks: contentBlocks,
		}
	}

	thinkingBlocks := msg.AnthropicThinkingBlocks()
	if len(thinkingBlocks) == 0 {
		return nil
	}
	return &MessageProviderMetadata{
		AnthropicThinkingBlocks: thinkingBlocks,
	}
}

func newMessageEntryFromAPI(msg api.Message, model string, ts time.Time) MessageEntry {
	return newMessageEntryFromAPIWithStoredContent(msg, msg.Content, model, ts)
}

func newMessageEntryFromAPIWithStoredContent(msg api.Message, content, model string, ts time.Time) MessageEntry {
	return MessageEntry{
		Timestamp:        ts,
		Role:             msg.Role,
		Content:          content,
		ReasoningContent: msg.ReasoningContent,
		Model:            model,
		ToolCalls:        msg.ToolCalls,
		ToolCallID:       msg.ToolCallID,
		ToolName:         msg.ToolName,
		ProviderMetadata: providerMetadataFromAPIMessage(msg),
	}
}
