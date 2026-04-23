package tools

type parseToolCallContext struct {
	response        string
	codeBlockRanges [][2]int
	registry        *Registry
	startFinder     jsonToolCallStartFinder
	logger          *parseDebugLogger
}

type parseToolCallExecutionPlan struct {
	codeBlockPolicy markdownCodeBlockPolicy
	phases          []parseToolCallPhase
}

type parseToolCallExecutionPlanBuilder struct {
	codeBlockPolicy markdownCodeBlockPolicy
	phases          []parseToolCallPhase
}

type parseToolCallPhase interface {
	Run(ctx parseToolCallContext, current []*ToolCall) []*ToolCall
}

type parseJSONToolCallPhase struct{}

type parseXMLRescueToolCallPhase struct{}

func defaultParseToolCallExecutionPlan() parseToolCallExecutionPlan {
	return newParseToolCallExecutionPlanBuilder().Build()
}

func (p parseToolCallExecutionPlan) ResolveCodeBlockRanges(response string) [][2]int {
	return findCodeBlockRangesWithPolicy(response, p.codeBlockPolicy)
}

func (p parseToolCallExecutionPlan) ResolvePhases() []parseToolCallPhase {
	return cloneParseToolCallPhases(p.phases)
}

func newParseToolCallContext(response string, options parseRunOptions) parseToolCallContext {
	return parseToolCallContext{
		response:        response,
		codeBlockRanges: options.plan.ResolveCodeBlockRanges(response),
		registry:        options.registry,
		startFinder:     options.startFinder,
		logger:          options.logger,
	}
}

func newParseToolCallExecutionPlanBuilder() parseToolCallExecutionPlanBuilder {
	return parseToolCallExecutionPlanBuilder{
		codeBlockPolicy: defaultMarkdownCodeBlockPolicy(),
		phases:          defaultParseToolCallPhases(),
	}
}

func (b parseToolCallExecutionPlanBuilder) WithCodeBlockPolicy(policy markdownCodeBlockPolicy) parseToolCallExecutionPlanBuilder {
	b.codeBlockPolicy = policy
	return b
}

func (b parseToolCallExecutionPlanBuilder) WithPhases(phases []parseToolCallPhase) parseToolCallExecutionPlanBuilder {
	b.phases = cloneParseToolCallPhases(phases)
	return b
}

func (b parseToolCallExecutionPlanBuilder) Build() parseToolCallExecutionPlan {
	return parseToolCallExecutionPlan{
		codeBlockPolicy: b.codeBlockPolicy,
		phases:          resolveParseToolCallPhases(b.phases),
	}
}

func defaultParseToolCallPhases() []parseToolCallPhase {
	return []parseToolCallPhase{
		parseJSONToolCallPhase{},
		parseXMLRescueToolCallPhase{},
	}
}

func resolveParseToolCallPhases(phases []parseToolCallPhase) []parseToolCallPhase {
	if len(phases) == 0 {
		return defaultParseToolCallPhases()
	}
	return cloneParseToolCallPhases(phases)
}

func cloneParseToolCallPhases(phases []parseToolCallPhase) []parseToolCallPhase {
	if len(phases) == 0 {
		return nil
	}
	cloned := make([]parseToolCallPhase, len(phases))
	copy(cloned, phases)
	return cloned
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
