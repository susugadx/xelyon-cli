package tools

type parseToolCallPhaseFunc func(ctx parseToolCallContext, current []*ToolCall) []*ToolCall

func (f parseToolCallPhaseFunc) Run(ctx parseToolCallContext, current []*ToolCall) []*ToolCall {
	return f(ctx, current)
}

type fixedParseToolCallPhase struct {
	tool string
}

func (p fixedParseToolCallPhase) Run(_ parseToolCallContext, _ []*ToolCall) []*ToolCall {
	return []*ToolCall{{Tool: p.tool}}
}
