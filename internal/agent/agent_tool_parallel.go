package agent

import (
	"context"
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
	filetool "github.com/susugadx/xelyon-cli/internal/tools/file"
)

type ToolExecCallback func(idx int, tc *tools.ToolCall, result string, change *tools.FileChange)

type toolExecResult struct {
	result string
	change *tools.FileChange
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
func (a *Agent) executeToolForParallel(ctx context.Context, tc *tools.ToolCall) (string, *tools.FileChange) {
	// wait_agent はリアルタイムイベント表示を使用（parallel path でも live view を優先）
	if tc.Tool == "wait_agent" {
		return a.executeWaitAgentWithLiveView(ctx, tc)
	}

	if ctx.Err() != nil {
		return "Error: context cancelled", nil
	}

	// ネガティブキャッシュチェック（ToolCache は thread-safe）
	if a.ToolCache != nil {
		if _, hit := a.ToolCache.CheckNegativeCache(tc.Tool, tc.RawArgs); hit {
			a.addOptimizationMetrics(OptimizationMetrics{NegativeCacheHits: 1})
		}
	}

	// ExecuteQuietWithContext: ヘッダー・引数・折りたたみ出力と補助 stdout を抑制（parallel path 用）
	result, change := tools.ExecuteQuietWithContext(a.toolExecutionContext(ctx, strings.NewReader(""), io.Discard, io.Discard), tc)
	a.recordToolResultOptimizations(tc, result)

	if a.ToolCache != nil {
		a.ToolCache.SetNegativeCache(tc.Tool, tc.RawArgs, result)
	}

	return result, change
}

func (a *Agent) executeReadFileBatch(ctx context.Context, paths []string) string {
	tc := buildReadFileBatchToolCall(paths, true)
	if ctx.Err() != nil {
		return "Error: context cancelled"
	}

	if a.ToolCache != nil {
		if _, hit := a.ToolCache.CheckNegativeCache(tc.Tool, tc.RawArgs); hit {
			a.addOptimizationMetrics(OptimizationMetrics{NegativeCacheHits: 1})
		}
	}

	execCtx := a.toolExecutionContext(ctx, strings.NewReader(""), io.Discard, io.Discard)
	result := filetool.ExecuteReadFilesWithLocator(execCtx.Output(), execCtx.EffectiveConfig(), execCtx.EffectiveToolCache(), paths, filetool.DefaultFullLines, execCtx.EffectiveLocatorRegistry())
	a.recordToolResultOptimizations(tc, result)

	if a.ToolCache != nil {
		a.ToolCache.SetNegativeCache(tc.Tool, tc.RawArgs, result)
	}

	return result
}
