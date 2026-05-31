package tuiagent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/websearch"
	"github.com/susugadx/xelyon-cli/internal/config"
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

	stats := *agent.Stats
	lastTurnUsage := stats.LastTurnUsage
	lastReviewUsage := stats.LastReviewUsage
	lastReviewCost := stats.LastReviewCost
	lastReviewCostUnknown := stats.LastReviewCostUnknown
	reviewUsage := stats.ReviewUsage
	reviewCost := stats.ReviewAccumulatedCost
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

func TestTUIAdapterRunReviewIncludesWebSearchEvidenceUsage(t *testing.T) {
	repo := setupReviewGitRepo(t)
	t.Chdir(repo)
	t.Setenv(reviewRunArtifactsEnv, "")
	writeReviewWebSearchEvidenceTriggerForTest(t, repo)

	searchCalls := registerReviewWebSearchUsageProviderForTest(t, "gemini", "gemini-3.1-pro-preview-customtools", api.Usage{
		InputTokens:  7,
		OutputTokens: 3,
	})

	provider := newReviewModelUsageProviderForTest(t, api.Usage{InputTokens: 10, OutputTokens: 1})
	agent := newReviewAgentForTest(t, provider)
	cfg := agent.Runtime.Config
	cfg.WebSearch.Provider = "gemini"
	cfg.WebSearch.CacheEnabled = false
	cfg.SetProviderModelConfig("gemini", config.ProviderModelConfig{DefaultModel: "gemini-3.1-pro-preview-customtools"})
	cfg.Review.WebSearchEvidence.Enabled = true
	cfg.Review.WebSearchEvidence.MaxQueries = 1
	cfg.Review.WebSearchEvidence.MaxResultsPerQuery = 1

	result, err := NewTUIAdapter(agent, nil).RunReview(context.Background(), review.NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("RunReview() error = %v", err)
	}
	if got := searchCalls(); got != 1 {
		t.Fatalf("search calls = %d, want 1", got)
	}
	if result.Usage.Tokens != "43 tok" {
		t.Fatalf("usage tokens = %q, want 43 tok including web search evidence", result.Usage.Tokens)
	}

	stats := *agent.Stats
	lastReviewUsage := stats.LastReviewUsage
	if lastReviewUsage == nil {
		t.Fatal("LastReviewUsage = nil, want review run usage")
	}
	if lastReviewUsage.InputTokens != 37 || lastReviewUsage.OutputTokens != 6 {
		t.Fatalf("LastReviewUsage = %+v, want model usage plus web search evidence usage", lastReviewUsage)
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

func writeReviewWebSearchEvidenceTriggerForTest(t *testing.T, repo string) {
	t.Helper()
	content := []byte("package main\n\nfunc main() { println(\"OpenAI web_search\") }\n")
	if err := os.WriteFile(filepath.Join(repo, "main.go"), content, 0o644); err != nil {
		t.Fatalf("write changed file: %v", err)
	}
}

func registerReviewWebSearchUsageProviderForTest(t *testing.T, provider, model string, usage api.Usage) func() int {
	t.Helper()
	calls := 0
	websearch.RegisterWithContextForTest(t, provider, func(ctx context.Context, _ string, gotModel string) (string, error) {
		calls++
		if gotModel != model {
			t.Fatalf("model = %q, want %q", gotModel, model)
		}
		callback := websearch.UsageCallbackFromContext(ctx)
		if callback == nil {
			t.Fatal("UsageCallbackFromContext() = nil, want callback")
		}
		callback(usage)
		return "No URL-bearing external source was returned.", nil
	})
	return func() int { return calls }
}

func newReviewModelUsageProviderForTest(t *testing.T, usage api.Usage) *scriptedChatProvider {
	t.Helper()
	provider := &scriptedChatProvider{name: "openai"}
	provider.chatWithToolsFn = func(call int, _ context.Context, _ string, _ []api.Message, _ string) (string, error) {
		if provider.usageCallback == nil {
			t.Fatal("provider usage callback was not configured")
		}
		provider.usageCallback(usage)
		switch call {
		case 0:
			return mustMarshalReviewValueForAgentTest(t, newAgentNoProbeReviewPlanForTest(
				"Agent review web search evidence usage path.",
				"Agent runner could omit web search evidence usage from review usage.",
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
	return provider
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
