package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/dev"
)

type investigationLoopAction int

const (
	investigationLoopContinue investigationLoopAction = iota
	investigationLoopDone
)

type planInvestigationRunner struct {
	agent           *Agent
	ctx             context.Context
	hardLimit       int
	lastToolCall    *tools.ToolCall
	sameCallCount   int
	autoCompression planInvestigationAutoCompression
}

type planInvestigationOptions struct {
	autoCompression planInvestigationAutoCompression
}

type planInvestigationAutoCompression struct {
	currentTurnStartIndex int
	turnState             *autoCompressionTurnState
	persistHistory        func() []api.Message
	onSuccess             func(currentTurnStartIndex int)
}

func (c planInvestigationAutoCompression) shouldAttempt() bool {
	return c.turnState != nil && c.currentTurnStartIndex > 0
}

func (c planInvestigationAutoCompression) compressionOptions() inTurnAutoCompressionOptions {
	if !c.shouldAttempt() || c.persistHistory == nil {
		return inTurnAutoCompressionOptions{}
	}
	return inTurnAutoCompressionOptions{persistHistory: c.persistHistory()}
}

func (c *planInvestigationAutoCompression) applyResult(result inTurnAutoCompressionResult) {
	if !result.compressed || result.compressedCurrentTurnStartIndex <= 0 {
		return
	}
	c.currentTurnStartIndex = result.compressedCurrentTurnStartIndex
	if c.onSuccess != nil {
		c.onSuccess(result.compressedCurrentTurnStartIndex)
	}
}

// runInvestigationPhase は調査フェーズを実行する。
// テキスト出力された計画（Plan JSON）を ExtractPlanJSON/ParsePlan でパースし、Plan を返す。
//
// NOTE: 本フェーズは executeToolCallsWithParallel を使用しない。
// SafetyHigh ツール制約のある独自ループ（executeToolOnly）を持ち、
// 並列実行の対象外である。ツールは1つずつ順次実行される。
func (a *Agent) runInvestigationPhase(ctx context.Context) (*plan.Plan, error) {
	return newPlanInvestigationRunner(a, ctx).Run()
}

func (a *Agent) runInvestigationPhaseWithOptions(ctx context.Context, opts planInvestigationOptions) (*plan.Plan, error) {
	return newPlanInvestigationRunnerWithOptions(a, ctx, opts).Run()
}

func newPlanInvestigationRunner(agent *Agent, ctx context.Context) *planInvestigationRunner {
	return newPlanInvestigationRunnerWithOptions(agent, ctx, planInvestigationOptions{
		autoCompression: planInvestigationAutoCompression{currentTurnStartIndex: len(agent.History)},
	})
}

func newPlanInvestigationRunnerWithOptions(agent *Agent, ctx context.Context, opts planInvestigationOptions) *planInvestigationRunner {
	cfg := agent.cfg()
	return &planInvestigationRunner{
		agent:           agent,
		ctx:             ctx,
		hardLimit:       normalizeToolLoopLimit(cfg.General.ToolLoopLimit),
		autoCompression: opts.autoCompression,
	}
}

func (r *planInvestigationRunner) Run() (*plan.Plan, error) {
	for iteration := 0; ; iteration++ {
		if err := r.beforeIteration(iteration); err != nil {
			return nil, err
		}

		response, err := r.requestResponse()
		if err != nil {
			return nil, err
		}

		toolCalls := r.agent.parseToolCalls(response)
		r.appendAssistantTurn(response)

		if len(toolCalls) == 0 {
			p, action, err := r.handleNoToolResponse(response)
			if err != nil {
				return nil, err
			}
			if action == investigationLoopContinue {
				continue
			}
			return p, nil
		}

		if err := r.executeToolCalls(toolCalls); err != nil {
			return nil, err
		}
		if err := r.afterToolResults(); err != nil {
			return nil, err
		}
	}
}

func (r *planInvestigationRunner) beforeIteration(iteration int) error {
	if r.hardLimit > 0 && iteration >= r.hardLimit {
		return fmt.Errorf("investigation phase exceeded max iterations (%d)", r.hardLimit)
	}
	if r.hardLimit == 0 {
		emitLoopWarning(r.agent, iteration)
	}
	return nil
}

func (r *planInvestigationRunner) requestResponse() (string, error) {
	response, err := r.agent.CurrentProvider.ChatWithTools(
		r.agent.requestContext(r.ctx),
		r.agent.SystemPrompt,
		r.agent.History,
		r.agent.CurrentModel,
	)
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}
	return response, nil
}

func (r *planInvestigationRunner) appendAssistantTurn(response string) {
	r.agent.recordAssistantAPITurn(response)
}

func (r *planInvestigationRunner) handleNoToolResponse(response string) (*plan.Plan, investigationLoopAction, error) {
	r.debugNoToolResponse(response)

	p, hadPlanJSON := r.extractPlan(response)
	if p != nil {
		return p, investigationLoopDone, nil
	}
	if hadPlanJSON {
		r.requestPlanJSONRetry()
		return nil, investigationLoopContinue, nil
	}

	r.agent.showAssistantResponse(response)
	return nil, investigationLoopDone, nil
}

func (r *planInvestigationRunner) debugNoToolResponse(response string) {
	if os.Getenv("XELYON_DEBUG_PARSE") != "1" {
		return
	}

	_, _ = fmt.Fprintf(r.agent.errorOutput(), "[DEBUG runInvestigationPhase] ParseToolCalls returned 0 tools\n")
	if strings.Contains(response, `{"tool"`) || strings.Contains(response, `{ "tool"`) {
		_, _ = fmt.Fprintf(r.agent.errorOutput(), "[DEBUG runInvestigationPhase] WARNING: tool pattern exists but not parsed!\n")
		if len(response) > 200 {
			_, _ = fmt.Fprintf(r.agent.errorOutput(), "[DEBUG runInvestigationPhase] tail: ...%s\n", response[len(response)-200:])
		}
	}
}

func (r *planInvestigationRunner) extractPlan(response string) (*plan.Plan, bool) {
	planJSON := plan.ExtractPlanJSON(response)
	if planJSON == "" {
		return nil, false
	}

	if os.Getenv("XELYON_DEBUG_PARSE") == "1" {
		_, _ = fmt.Fprintf(r.agent.errorOutput(), "[DEBUG runInvestigationPhase] found plan JSON (%d bytes)\n", len(planJSON))
	}

	p, err := plan.ParsePlan(planJSON)
	if err == nil && len(p.Steps) > 0 {
		return p, true
	}
	return nil, true
}

func (r *planInvestigationRunner) requestPlanJSONRetry() {
	r.agent.History = append(r.agent.History, api.Message{
		Role:    "user",
		Content: "[SYSTEM] Plan JSON を**必ず**次のスキーマ例に沿って、```json``` で囲んだ1つのJSONとして出力してください（箇条書き/番号付きリスト/文章のみは禁止）。\n\n例:\n```json\n{\n  \"title\": \"調査と実装計画\",\n  \"goal\": \"<最終的に達成したいこと>\",\n  \"assumptions\": [\"<前提>\"],\n  \"steps\": [\n    {\n      \"id\": 1,\n      \"title\": \"<手順タイトル>\",\n      \"description\": \"<この手順でやること>\",\n      \"expected_output\": \"<完了条件/成果物>\"\n    }\n  ]\n}\n```",
	})
}

func (r *planInvestigationRunner) executeToolCalls(toolCalls []*tools.ToolCall) error {
	for _, tc := range toolCalls {
		if r.isAllowedTool(tc) {
			if r.shouldAbortToolLoop(tc) {
				return fmt.Errorf("tool loop detected during investigation")
			}
			r.lastToolCall = tc
			r.agent.executeToolOnly(tc)
			continue
		}

		r.rejectTool(tc)
	}
	return nil
}

func (r *planInvestigationRunner) afterToolResults() error {
	if err := requestContextErr(r.ctx); err != nil {
		return err
	}
	if !r.autoCompression.shouldAttempt() {
		return nil
	}
	result := r.agent.maybeAutoCompressDuringTurnWithOptions(
		r.ctx,
		r.autoCompression.currentTurnStartIndex,
		r.autoCompression.turnState,
		r.autoCompression.compressionOptions(),
	)
	if result.requestErr != nil {
		return result.requestErr
	}
	r.autoCompression.applyResult(result)
	return nil
}

func (r *planInvestigationRunner) isAllowedTool(tc *tools.ToolCall) bool {
	safety := tools.GetToolSafety(tc.Tool)
	if safety == tools.SafetyHigh {
		return true
	}

	if tc.Tool != "bash" && tc.Tool != "command" {
		return false
	}
	return dev.IsSafeCommand(tc.Args["command"], r.agent.cfg().Bash)
}

func (r *planInvestigationRunner) shouldAbortToolLoop(tc *tools.ToolCall) bool {
	return r.agent.shouldAbortToolLoop(tc, r.lastToolCall, &r.sameCallCount)
}

func (r *planInvestigationRunner) rejectTool(tc *tools.ToolCall) {
	r.agent.History = append(r.agent.History, api.Message{
		Role:    "user",
		Content: fmt.Sprintf("[SYSTEM] Tool '%s' is not allowed in investigation phase. Please only use read-only tools.", tc.Tool),
	})
}
