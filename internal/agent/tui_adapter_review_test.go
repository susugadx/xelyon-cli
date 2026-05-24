package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/review"
	"github.com/susugadx/xelyon-cli/internal/tui"
)

func TestTUIAdapterRunReviewReturnsUsageSummaryAndRecordsLastReview(t *testing.T) {
	repo := setupReviewGitRepo(t)
	t.Chdir(repo)
	t.Setenv(reviewRunArtifactsEnv, "")

	provider := &scriptedChatProvider{name: "openai"}
	provider.chatWithToolsFn = func(call int, _ context.Context, _ string, _ []api.Message, _ string) (string, error) {
		if provider.usageCallback == nil {
			t.Fatal("provider usage callback was not configured")
		}
		provider.usageCallback(api.Usage{
			InputTokens:  1000 * (call + 1),
			OutputTokens: 100 * (call + 1),
		})
		switch call {
		case 0:
			return mustMarshalReviewValueForAgentTest(t, newAgentNoProbeReviewPlanForTest(
				"Agent review runner path.",
				"Agent runner could omit review usage from TUI status.",
			)), nil
		case 1:
			return mustMarshalReviewValueForAgentTest(t, newAgentCleanReviewReportForTest()), nil
		case 2:
			return mustMarshalReviewValueForAgentTest(t, newAgentSaturatedReviewCheckForTest()), nil
		default:
			t.Fatalf("unexpected provider call %d", call)
			return "", nil
		}
	}
	agent := newReviewAgentForTest(t, provider)
	agent.CurrentModel = "gpt-5.4-nano"
	agent.Model = "gpt-5.4-nano"

	result, err := NewTUIAdapter(agent, nil).RunReview(context.Background(), review.NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("RunReview() error = %v", err)
	}
	if result.Usage.Tokens != "6.6k tok" {
		t.Fatalf("usage tokens = %q, want 6.6k tok", result.Usage.Tokens)
	}
	if !strings.HasPrefix(result.Usage.Cost, "~$") {
		t.Fatalf("usage cost = %q, want compact estimated cost", result.Usage.Cost)
	}

	agent.statsMu.Lock()
	lastTurnUsage := agent.Stats.LastTurnUsage
	lastReviewUsage := agent.Stats.LastReviewUsage
	lastReviewCost := agent.Stats.LastReviewCost
	lastReviewCostUnknown := agent.Stats.LastReviewCostUnknown
	reviewUsage := agent.Stats.ReviewUsage
	reviewCost := agent.Stats.ReviewAccumulatedCost
	agent.statsMu.Unlock()
	if lastTurnUsage != nil {
		t.Fatalf("LastTurnUsage = %+v, want nil because review usage is tracked separately", lastTurnUsage)
	}
	if lastReviewUsage == nil {
		t.Fatal("LastReviewUsage = nil, want review run usage for /status")
	}
	if lastReviewUsage.InputTokens != 6000 || lastReviewUsage.OutputTokens != 600 {
		t.Fatalf("LastReviewUsage = %+v, want review run token totals", lastReviewUsage)
	}
	if reviewUsage.InputTokens != 6000 || reviewUsage.OutputTokens != 600 {
		t.Fatalf("ReviewUsage = %+v, want cumulative review token totals", reviewUsage)
	}
	if lastReviewCost <= 0 || lastReviewCostUnknown {
		t.Fatalf("LastReviewCost = %f unknown=%t, want known review run cost", lastReviewCost, lastReviewCostUnknown)
	}
	if reviewCost != lastReviewCost {
		t.Fatalf("ReviewAccumulatedCost = %f, want last review cost %f", reviewCost, lastReviewCost)
	}
}

func TestTUIAdapterRunReviewEmitsProgressToolResults(t *testing.T) {
	repo := setupReviewGitRepo(t)
	t.Chdir(repo)
	t.Setenv(reviewRunArtifactsEnv, "")

	provider := &scriptedChatProvider{name: "openai"}
	provider.chatWithToolsFn = func(call int, _ context.Context, _ string, _ []api.Message, _ string) (string, error) {
		switch call {
		case 0:
			return mustMarshalReviewValueForAgentTest(t, newAgentNoProbeReviewPlanForTest(
				"Agent review runner path.",
				"Agent runner should expose review progress to TUI.",
			)), nil
		case 1:
			return mustMarshalReviewValueForAgentTest(t, newAgentCleanReviewReportForTest()), nil
		case 2:
			return mustMarshalReviewValueForAgentTest(t, newAgentSaturatedReviewCheckForTest()), nil
		default:
			t.Fatalf("unexpected provider call %d", call)
			return "", nil
		}
	}
	agent := newReviewAgentForTest(t, provider)
	adapter := NewTUIAdapter(agent, nil)
	var toolMessages []tui.AppendToolResultMsg
	adapter.sendToolResult = func(msg tui.AppendToolResultMsg) {
		toolMessages = append(toolMessages, msg)
	}

	if _, err := adapter.RunReview(context.Background(), review.NewCurrentChangesRequest("")); err != nil {
		t.Fatalf("RunReview() error = %v", err)
	}

	assertReviewProgressToolMessage(t, toolMessages, "review:evidence", tui.ToolStatusRunning, "collecting current changes", "")
	assertReviewProgressToolMessage(t, toolMessages, "review:evidence", tui.ToolStatusOK, "evidence collected", "staged")
	assertReviewProgressToolMessage(t, toolMessages, "review:probe_plan", tui.ToolStatusRunning, "planning probes", "")
	assertReviewProgressToolMessage(t, toolMessages, "review:probe_plan", tui.ToolStatusOK, "planned probes", "0 checks")
	assertReviewProgressToolMessage(t, toolMessages, "review:report", tui.ToolStatusRunning, "writing report", "")
	assertReviewProgressToolMessage(t, toolMessages, "review:report", tui.ToolStatusOK, "report drafted", "")
	assertReviewProgressToolMessage(t, toolMessages, "review:saturation_check", tui.ToolStatusRunning, "checking review coverage", "")
	assertReviewProgressToolMessage(t, toolMessages, "review:saturation_check", tui.ToolStatusOK, "coverage checked", string(review.ReviewSaturationStatusSaturated))
}

func TestTUIAdapterRunReviewEmitsRunScopedProgress(t *testing.T) {
	repo := setupReviewGitRepo(t)
	t.Chdir(repo)
	t.Setenv(reviewRunArtifactsEnv, "")

	provider := &scriptedChatProvider{name: "openai"}
	provider.chatWithToolsFn = func(call int, _ context.Context, _ string, _ []api.Message, _ string) (string, error) {
		switch call {
		case 0:
			return mustMarshalReviewValueForAgentTest(t, newAgentNoProbeReviewPlanForTest(
				"Agent review runner path.",
				"Agent runner should expose review progress to TUI.",
			)), nil
		case 1:
			return mustMarshalReviewValueForAgentTest(t, newAgentCleanReviewReportForTest()), nil
		case 2:
			return mustMarshalReviewValueForAgentTest(t, newAgentSaturatedReviewCheckForTest()), nil
		default:
			t.Fatalf("unexpected provider call %d", call)
			return "", nil
		}
	}
	agent := newReviewAgentForTest(t, provider)
	adapter := NewTUIAdapter(agent, nil)
	var progressMessages []tui.ReviewProgressMsg
	var fallbackToolMessages []tui.AppendToolResultMsg
	adapter.sendReviewProgress = func(msg tui.ReviewProgressMsg) {
		progressMessages = append(progressMessages, msg)
	}
	adapter.sendToolResult = func(msg tui.AppendToolResultMsg) {
		fallbackToolMessages = append(fallbackToolMessages, msg)
	}

	ctx := tui.ContextWithReviewRunID(context.Background(), 42)
	if _, err := adapter.RunReview(ctx, review.NewCurrentChangesRequest("")); err != nil {
		t.Fatalf("RunReview() error = %v", err)
	}

	if len(fallbackToolMessages) != 0 {
		t.Fatalf("fallback tool messages = %#v, want none when review run id is present", fallbackToolMessages)
	}
	assertReviewProgressMessage(t, progressMessages, 42, "review:evidence", tui.ToolStatusRunning, "collecting current changes", "")
	assertReviewProgressMessage(t, progressMessages, 42, "review:saturation_check", tui.ToolStatusOK, "coverage checked", string(review.ReviewSaturationStatusSaturated))
}

func assertReviewProgressToolMessage(t *testing.T, messages []tui.AppendToolResultMsg, id string, status tui.ToolStatus, name string, targetFragment string) {
	t.Helper()

	for _, msg := range messages {
		tool := msg.Tool
		if tool.ID != id || tool.Status != status || tool.Name != name {
			continue
		}
		if targetFragment == "" || strings.Contains(tool.Target, targetFragment) {
			return
		}
	}
	t.Fatalf("progress tool message %s/%s/%s target %q not found in %#v", id, status, name, targetFragment, messages)
}

func assertReviewProgressMessage(t *testing.T, messages []tui.ReviewProgressMsg, runID int, id string, status tui.ToolStatus, name string, targetFragment string) {
	t.Helper()

	for _, msg := range messages {
		tool := msg.Tool
		if msg.RunID != runID || tool.ID != id || tool.Status != status || tool.Name != name {
			continue
		}
		if targetFragment == "" || strings.Contains(tool.Target, targetFragment) {
			return
		}
	}
	t.Fatalf("review progress message %d/%s/%s/%s target %q not found in %#v", runID, id, status, name, targetFragment, messages)
}
