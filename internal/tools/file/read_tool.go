package file

import "github.com/susugadx/xelyon-cli/internal/tools"

// ReadFileTool は1回の呼び出しで1個以上のファイルを読み込むツール。
type ReadFileTool struct{}

func (t *ReadFileTool) Name() string { return "read_file" }

func (t *ReadFileTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

func (t *ReadFileTool) Parameters() map[string]interface{} {
	return schemaReadFileParameters(MaxReadFilesPaths)
}

func (t *ReadFileTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	result, err := t.RunResult(execCtx, args)
	return result.Output, result.Change, err
}

// RunResult は read_file の描画結果と runtime observation を返す。
func (t *ReadFileTool) RunResult(execCtx tools.ExecutionContext, args map[string]string) (tools.ToolRunResult, error) {
	detail, result := resolveReadDetail(args["detail"], args["_full_budget"])
	if result != "" {
		return tools.ToolRunResult{Output: result}, nil
	}
	budgetOverride := resolveReadBudgetOverride(args["detail"], args["_full_budget"])

	requests, reg, result, err := resolveReadTargets(execCtx, args["targets"], args["paths"], detail)
	if result != "" || err != nil {
		return tools.ToolRunResult{Output: result}, err
	}

	sections := executeReadFilesRequestsSections(
		execCtx.Output(),
		execCtx.EffectiveConfig(),
		execCtx.EffectiveToolCache(),
		requests,
		budgetOverride,
		reg,
	)
	return tools.ToolRunResult{
		Output:      renderReadExecutionSections(sections),
		Observation: MergeReadExecutionSectionObservations(sections),
	}, nil
}
