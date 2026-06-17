package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// ExecutionResult はツール実行結果と、結果表示に必要な実行メタデータを保持する。
type ExecutionResult struct {
	Result      string
	Change      *FileChange
	Observation *RuntimeObservation
	// ObservationGroups は batched result を元の tool call へ分配するための任意グループ。
	ObservationGroups map[string]*RuntimeObservation
	StartedAt         time.Time
	Duration          time.Duration
	Error             bool
}

// ExecuteWithContext は実行コンテキスト付きでツールを実行し、結果を公開する。
func ExecuteWithContext(execCtx ExecutionContext, tc *ToolCall) (string, *FileChange) {
	execResult := ExecuteUnpublishedWithContext(execCtx, tc)
	PublishResultWithContext(execCtx, tc, execResult)
	return execResult.Result, execResult.Change
}

// ExecuteUnpublishedWithContext はツールを実行し、wrapper 層の結果表示は行わない。
func ExecuteUnpublishedWithContext(execCtx ExecutionContext, tc *ToolCall) ExecutionResult {
	execCtx = normalizeExecutionContext(execCtx)
	startTime := time.Now()
	execResult := executeCoreWithContext(execCtx, tc)
	elapsed := time.Since(startTime)

	execResult.StartedAt = startTime
	execResult.Duration = elapsed
	return execResult
}

// ExecuteQuietWithContext は実行コンテキスト付きでツールを実行するが、wrapper 出力を抑制する。
func ExecuteQuietWithContext(execCtx ExecutionContext, tc *ToolCall) (string, *FileChange) {
	execResult := ExecuteQuietUnpublishedWithContext(execCtx, tc)
	return execResult.Result, execResult.Change
}

// ExecuteQuietUnpublishedWithContext は quiet mode でツールを実行し、wrapper 層の結果表示は行わない。
func ExecuteQuietUnpublishedWithContext(execCtx ExecutionContext, tc *ToolCall) ExecutionResult {
	execCtx = normalizeExecutionContext(execCtx)
	restoreQuiet := common.PushQuietMode()
	defer restoreQuiet()
	return ExecuteUnpublishedWithContext(execCtx, tc)
}

func executeCoreWithContext(execCtx ExecutionContext, tc *ToolCall) ExecutionResult {
	if err := execCtx.EffectiveContext().Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			return ExecutionResult{Result: "Error: context cancelled", Error: true}
		}
		return ExecutionResult{Result: fmt.Sprintf("Error: %v", err), Error: true}
	}

	applyToolCallDefaults(tc)
	execResult := executeToolCallCore(execCtx, tc)
	execResult.Result = normalizeToolExecutionOutput(execResult.Result)
	execResult.Error = execResult.Error || IsErrorResult(execResult.Result)

	return execResult
}

func applyToolCallDefaults(tc *ToolCall) {
	if tc == nil {
		return
	}
	if tc.Args == nil {
		tc.Args = make(map[string]string)
	}
	if tc.Args["path"] != "" {
		return
	}
	switch tc.Tool {
	case "list_dir", "git_add":
		tc.Args["path"] = "."
	}
}

func executeToolCallCore(execCtx ExecutionContext, tc *ToolCall) ExecutionResult {
	execResult := execCtx.EffectiveRegistry().ExecuteDetailedWithContext(execCtx, tc)
	invalidateToolCache(execCtx, tc, execResult.Change)
	return execResult
}

func normalizeToolExecutionOutput(result string) string {
	if strings.TrimSpace(result) == "" {
		return "(no output)"
	}
	return result
}

// IsErrorResult はツール結果が失敗メッセージかどうかを判定する。
func IsErrorResult(result string) bool {
	return strings.HasPrefix(strings.TrimSpace(result), "Error:")
}
