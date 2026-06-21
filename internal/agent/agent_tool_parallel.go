package agent

import (
	"context"
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/file/readtool"
)

type ToolExecCallback func(idx int, tc *tools.ToolCall, result toolruntime.Result)

type readFileBatchExecution struct {
	Execution toolruntime.Result
	Sections  []readtool.ReadExecutionSection
}

// executeToolForParallel は並列実行用のツール実行関数。
// goroutine から安全に呼び出せるよう、以下を省略している:
//   - spinner / SetGlobalSpinner: goroutine 内で global spinner を操作すると競合する
//   - stdout/stderr: io.Discard を使って補助出力を抑制する
//
// stdout 出力の抑制:
//
//	tools.ExecuteQuietWithContext を使用し、wrapper 層の出力（ヘッダー・引数・折りたたみ結果）を抑制する。
//	Tool.Run() 内部の補助的な出力は execCtx.Stdout=io.Discard に流し、process stdout へは出さない。
//
// cancel の実効性（best effort）:
//
//	ctx.Err() を実行前にチェックして早期リターンする。
//	Tool.Run() まで request context を伝播するが、各ツールがそれを使うかは個別実装次第。
//	現状は bash など context-aware なツールが実行中キャンセルを拾える。
func (a *Agent) executeToolForParallelResult(ctx context.Context, tc *tools.ToolCall) tools.ExecutionResult {
	// wait_agent はリアルタイムイベント表示を使用（parallel path でも live view を優先）
	if tc.Tool == "wait_agent" {
		output, change := a.executeWaitAgentWithLiveView(ctx, tc)
		return tools.ExecutionResult{
			Result: output,
			Change: change,
			Error:  tools.IsErrorResult(output),
		}
	}

	if ctx.Err() != nil {
		return tools.ExecutionResult{
			Result: "Error: context cancelled",
			Error:  true,
		}
	}

	// ネガティブキャッシュチェック（ToolCache は thread-safe）
	if a.ToolCache != nil {
		if _, hit := a.ToolCache.CheckNegativeCache(tc.Tool, tc.RawArgs); hit {
			a.addOptimizationMetrics(OptimizationMetrics{NegativeCacheHits: 1})
		}
	}

	// ヘッダー・引数・折りたたみ出力と補助 stdout を抑制（parallel path 用）
	execResult := a.executeQuietToolResult(ctx, tc, strings.NewReader(""), io.Discard, io.Discard, false)
	a.recordToolResultOptimizations(tc, execResult.Result)

	if a.ToolCache != nil {
		a.ToolCache.SetNegativeCache(tc.Tool, tc.RawArgs, execResult.Result)
	}

	return execResult
}

func (a *Agent) executeReadFileBatchResult(ctx context.Context, paths []string) readFileBatchExecution {
	tc := toolruntime.BuildReadFileBatchToolCall(paths, true)
	if ctx.Err() != nil {
		return readFileBatchExecution{Execution: toolruntime.Result{Result: "Error: context cancelled", Error: true}}
	}

	if a.ToolCache != nil {
		if _, hit := a.ToolCache.CheckNegativeCache(tc.Tool, tc.RawArgs); hit {
			a.addOptimizationMetrics(OptimizationMetrics{NegativeCacheHits: 1})
		}
	}

	execCtx := a.toolExecutionContext(ctx, strings.NewReader(""), io.Discard, io.Discard)
	sections := readtool.ExecuteReadPathsWithDetailSections(execCtx, paths, "")
	result := readtool.RenderReadExecutionSections(sections)
	a.recordToolResultOptimizations(tc, result)

	if a.ToolCache != nil {
		a.ToolCache.SetNegativeCache(tc.Tool, tc.RawArgs, result)
	}

	return readFileBatchExecution{
		Execution: toolruntime.Result{
			Result:      result,
			Observation: readtool.MergeReadExecutionSectionObservations(sections),
			Error:       tools.IsErrorResult(result),
		},
		Sections: sections,
	}
}
