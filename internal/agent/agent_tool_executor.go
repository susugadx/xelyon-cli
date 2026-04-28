package agent

import (
	"context"
	"io"
	"os"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func (a *Agent) toolExecutionContext(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) tools.ExecutionContext {
	runtimeUI := a.ui()
	invocationCWD, _ := os.Getwd()
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

func (a *Agent) executeToolWithSpinner(ctx context.Context, toolCall *tools.ToolCall) (string, *tools.FileChange) {
	if a.ToolCache != nil {
		if result, hit := a.ToolCache.CheckNegativeCache(toolCall.Tool, toolCall.RawArgs); hit {
			yellow.Fprintf(a.output(), "⚠ Negative cache hit: %s previously returned: %s\n", toolCall.Tool, result)
			a.addOptimizationMetrics(OptimizationMetrics{NegativeCacheHits: 1})
		}
	}

	if toolCall.Tool == "wait_agent" {
		return a.executeWaitAgentWithLiveView(ctx, toolCall)
	}

	spinner := a.ui().NewSpinner()
	spinnerMsg := ui.SpinnerMessageForTool(toolCall.Tool)
	spinner.Start(spinnerMsg)
	a.ui().SetSpinner(spinner)

	execCtx := a.toolExecutionContext(ctx, nil, nil, nil)
	var result string
	var change *tools.FileChange
	if a.shouldRepairGeminiApplyPatch(toolCall) {
		execResult := tools.ExecuteUnpublishedWithContext(execCtx, toolCall)
		a.ui().StopSpinner()
		execResult = a.maybeRepairGeminiApplyPatchExecution(ctx, toolCall, execResult, execCtx, false)
		tools.PublishResultWithContext(execCtx, toolCall, execResult)
		result, change = execResult.Result, execResult.Change
	} else {
		result, change = tools.ExecuteWithContext(execCtx, toolCall)
		a.ui().StopSpinner()
	}
	a.recordToolResultOptimizations(toolCall, result)

	if a.ToolCache != nil {
		a.ToolCache.SetNegativeCache(toolCall.Tool, toolCall.RawArgs, result)
	}

	return result, change
}
