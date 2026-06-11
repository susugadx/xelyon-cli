package api

// CloneMessages は provider-facing history 用にチャットメッセージ群を defensive copy する。
func CloneMessages(messages []Message) []Message {
	if len(messages) == 0 {
		return nil
	}
	cloned := make([]Message, len(messages))
	for i, msg := range messages {
		cloned[i] = cloneMessage(msg)
	}
	return cloned
}

func cloneMessage(msg Message) Message {
	cloned := msg
	cloned.ToolCalls = cloneOpenAIToolCalls(msg.ToolCalls)
	cloned.providerState.anthropicContentBlocks = CloneAnthropicContentBlocks(msg.providerState.anthropicContentBlocks)
	cloned.providerState.anthropicThinkingBlocks = cloneAnthropicThinkingBlocks(msg.providerState.anthropicThinkingBlocks)
	cloned.providerState.openAIResponsesItems = CloneInputItems(msg.providerState.openAIResponsesItems)
	return cloned
}

func cloneOpenAIToolCalls(toolCalls []OpenAIToolCall) []OpenAIToolCall {
	if len(toolCalls) == 0 {
		return nil
	}
	cloned := make([]OpenAIToolCall, len(toolCalls))
	for i, toolCall := range toolCalls {
		cloned[i] = toolCall
		cloned[i].ThoughtParts = cloneInputItemThoughtParts(toolCall.ThoughtParts)
	}
	return cloned
}

func cloneAnthropicThinkingBlocks(blocks []AnthropicThinkingBlock) []AnthropicThinkingBlock {
	if len(blocks) == 0 {
		return nil
	}
	cloned := make([]AnthropicThinkingBlock, len(blocks))
	copy(cloned, blocks)
	return cloned
}
