package agent

import (
	"context"
	"io"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func (a *Agent) toolExecutionContext(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) tools.ExecutionContext {
	runtimeUI := a.ui()
	invocationCWD := a.invocationCWD()
	if ctx == nil {
		ctx = a.currentRequestContext()
	}
	if stdin == nil {
		stdin = runtimeUI.Input()
	}
	if stdout == nil {
		stdout = runtimeUI.Output()
	}
	if stderr == nil {
		stderr = runtimeUI.ErrorOutput()
	}
	ec := tools.ExecutionContext{
		Context:            ctx,
		PromptContext:      a.requestToolPromptContext(ctx),
		Provider:           a.CurrentProvider,
		ProviderName:       a.ProviderName,
		ProviderConfigKey:  a.activeModelProviderConfigKey(a.cfg()),
		Model:              a.CurrentModel,
		Stdin:              stdin,
		Stdout:             stdout,
		Stderr:             stderr,
		PromptReader:       runtimeUI.PromptReader(),
		Runtime:            runtimeUI,
		Registry:           a.registry(),
		ToolCache:          a.ToolCache,
		LSPClient:          a.GetLSPClient(),
		Config:             a.cfg(),
		ProjectMap:         a.projectMap,
		ProjectMapRootPath: a.projectMapRootPath,
		ProjectMapStateKey: a.projectMapStateKey,
		InvocationCWD:      invocationCWD,
		AutoApprove:        a.autoApprove(),
		AuditLogger:        a.auditLogger(),
		LocatorRegistry:    a.LocatorRegistry,
	}

	if a.tuiToolResultCh != nil {
		ch := a.tuiToolResultCh
		ec.ToolResultCallback = func(info tools.ToolResultInfo) {
			if a.tuiToolResultClosed.Load() {
				return
			}
			select {
			case ch <- info:
			default:
			}
		}
	}

	return ec
}

func (a *Agent) currentRequestContext() context.Context {
	if a != nil && a.requestCtx != nil {
		return a.requestCtx
	}
	return context.Background()
}

func (a *Agent) executeQuietToolResult(ctx context.Context, toolCall *tools.ToolCall, stdin io.Reader, stdout, stderr io.Writer, repair bool) tools.ExecutionResult {
	execCtx := a.toolExecutionContext(ctx, stdin, stdout, stderr)
	execResult := tools.ExecuteQuietUnpublishedWithContext(execCtx, toolCall)
	if repair {
		execResult = a.maybeRepairGeminiApplyPatchExecution(ctx, toolCall, execResult, execCtx, true)
	}
	return execResult
}

func (a *Agent) executeToolWithSpinner(ctx context.Context, toolCall *tools.ToolCall) (string, *tools.FileChange) {
	execResult := a.executeToolWithSpinnerResult(ctx, toolCall)
	return execResult.Result, execResult.Change
}

func (a *Agent) executeToolWithSpinnerResult(ctx context.Context, toolCall *tools.ToolCall) tools.ExecutionResult {
	a.reportNegativeCacheHit(toolCall)

	if execResult, handled := a.executeImmediateToolResult(ctx, toolCall); handled {
		return execResult
	}

	execResult := a.executePublishedToolWithSpinner(ctx, toolCall)
	a.recordToolResultOptimizations(toolCall, execResult.Result)

	if a.ToolCache != nil {
		a.ToolCache.SetNegativeCache(toolCall.Tool, toolCall.RawArgs, execResult.Result)
	}

	return execResult
}

func (a *Agent) reportNegativeCacheHit(toolCall *tools.ToolCall) {
	if a == nil || a.ToolCache == nil || toolCall == nil {
		return
	}
	if result, hit := a.ToolCache.CheckNegativeCache(toolCall.Tool, toolCall.RawArgs); hit {
		yellow.Fprintf(a.output(), "⚠ Negative cache hit: %s previously returned: %s\n", toolCall.Tool, result)
		a.addOptimizationMetrics(OptimizationMetrics{NegativeCacheHits: 1})
	}
}

func (a *Agent) executeImmediateToolResult(ctx context.Context, toolCall *tools.ToolCall) (tools.ExecutionResult, bool) {
	if a == nil || toolCall == nil {
		return tools.ExecutionResult{}, false
	}
	if toolCall.Tool != "wait_agent" {
		return tools.ExecutionResult{}, false
	}
	output, change := a.executeWaitAgentWithLiveView(ctx, toolCall)
	return tools.ExecutionResult{
		Result: output,
		Change: change,
		Error:  tools.IsErrorResult(output),
	}, true
}

func (a *Agent) executePublishedToolWithSpinner(ctx context.Context, toolCall *tools.ToolCall) tools.ExecutionResult {
	spinner := a.ui().NewSpinner()
	spinner.Start(ui.SpinnerMessageForTool(toolCall.Tool))
	a.ui().SetSpinner(spinner)

	execCtx := a.toolExecutionContext(ctx, nil, nil, nil)
	if a.shouldRepairGeminiApplyPatch(toolCall) {
		execResult := tools.ExecuteUnpublishedWithContext(execCtx, toolCall)
		a.ui().StopSpinner()
		execResult = a.maybeRepairGeminiApplyPatchExecution(ctx, toolCall, execResult, execCtx, false)
		tools.PublishResultWithContext(execCtx, toolCall, execResult)
		return execResult
	}

	execResult := tools.ExecuteUnpublishedWithContext(execCtx, toolCall)
	tools.PublishResultWithContext(execCtx, toolCall, execResult)
	a.ui().StopSpinner()
	return execResult
}
