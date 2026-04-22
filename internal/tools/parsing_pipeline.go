package tools

type parseToolCallContext struct {
	response        string
	codeBlockRanges [][2]int
	registry        *Registry
	startFinder     jsonToolCallStartFinder
	logger          *parseDebugLogger
}

type parseToolCallPhase interface {
	Run(ctx parseToolCallContext, current []*ToolCall) []*ToolCall
}

type parseJSONToolCallPhase struct{}

type parseXMLRescueToolCallPhase struct{}

func newParseToolCallContext(response string, options parseRunOptions) parseToolCallContext {
	return parseToolCallContext{
		response:        response,
		codeBlockRanges: findCodeBlockRangesWithPolicy(response, options.codeBlockPolicy),
		registry:        options.registry,
		startFinder:     options.startFinder,
		logger:          options.logger,
	}
}

func defaultParseToolCallPhases() []parseToolCallPhase {
	return []parseToolCallPhase{
		parseJSONToolCallPhase{},
		parseXMLRescueToolCallPhase{},
	}
}

func runParseToolCallPhases(ctx parseToolCallContext, phases []parseToolCallPhase) []*ToolCall {
	var current []*ToolCall
	for _, phase := range phases {
		current = phase.Run(ctx, current)
	}
	return current
}

func (parseJSONToolCallPhase) Run(ctx parseToolCallContext, _ []*ToolCall) []*ToolCall {
	return parseJSONToolCalls(ctx.response, ctx.codeBlockRanges, ctx.startFinder, ctx.logger)
}

func (parseXMLRescueToolCallPhase) Run(ctx parseToolCallContext, current []*ToolCall) []*ToolCall {
	// 既存契約: JSONで1件以上抽出できた場合はXML rescueを走らせない。
	if len(current) > 0 {
		return current
	}
	return parseXMLToolCalls(ctx.response, ctx.codeBlockRanges, ctx.registry, ctx.logger)
}
