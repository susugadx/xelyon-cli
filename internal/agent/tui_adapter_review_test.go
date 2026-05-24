package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/review"
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
