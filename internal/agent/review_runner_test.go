package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/review"
)

func TestAgentReviewModelCompleteReviewPassesPromptAsSingleUserMessage(t *testing.T) {
	var captured struct {
		systemPrompt string
		history      []api.Message
		model        string
		updateMode   string
		toolsOff     bool
		toolCount    int
		mergedTools  int
		compacted    int
		cacheNS      string
	}
	provider := &scriptedChatProvider{
		name: "openai",
		chatWithToolsFn: func(_ int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			captured.systemPrompt = systemPrompt
			captured.history = append([]api.Message(nil), history...)
			captured.model = model
			captured.updateMode = api.AssistantUpdateModeFromContext(ctx)
			captured.toolsOff = api.IsToolUseDisabled(ctx)
			captured.toolCount = len(api.ToolDefinitionsFromContext(ctx))
			captured.mergedTools = len(api.ToolDefinitionsWithAdditional(ctx, []api.ToolDefinition{{
				Name:        "mcp_fake_tool",
				Description: "fake mcp tool",
				Parameters:  map[string]interface{}{"type": "object"},
			}}))
			captured.compacted = len(api.CompactedInputItemsFromContext(ctx))
			captured.cacheNS = api.ProviderCacheNamespaceFromContext(ctx)
			return `{"ok":true}`, nil
		},
	}
	agent := newReviewAgentForTest(t, provider)
	agent.isCompactedMode = true
	agent.compactedItems = []api.InputItem{{Type: "message", Role: "user", Content: "existing compacted chat"}}

	resp, err := (agentReviewModel{agent: agent}).CompleteReview(context.Background(), review.ReviewModelRequest{
		Phase:  review.ReviewModelPhaseReport,
		Prompt: "return review json",
	})
	if err != nil {
		t.Fatalf("CompleteReview() error = %v", err)
	}
	if resp.Content != `{"ok":true}` {
		t.Fatalf("Content = %q, want raw provider response", resp.Content)
	}
	if captured.systemPrompt != "" {
		t.Fatalf("systemPrompt = %q, want empty", captured.systemPrompt)
	}
	if len(captured.history) != 1 || captured.history[0].Role != "user" || captured.history[0].Content != "return review json" {
		t.Fatalf("history = %#v, want single user prompt", captured.history)
	}
	if captured.model != "review-model" {
		t.Fatalf("model = %q, want review-model", captured.model)
	}
	if captured.updateMode != api.AssistantUpdatesOff {
		t.Fatalf("assistant update mode = %q, want off", captured.updateMode)
	}
	if !captured.toolsOff {
		t.Fatal("tool use disabled = false, want true for review model call")
	}
	if captured.toolCount != 0 {
		t.Fatalf("tool definitions = %d, want 0 for review model call", captured.toolCount)
	}
	if captured.mergedTools != 0 {
		t.Fatalf("merged tool definitions = %d, want 0 for isolated review model call", captured.mergedTools)
	}
	if captured.compacted != 0 {
		t.Fatalf("compacted input items = %d, want 0 for isolated review model call", captured.compacted)
	}
	if captured.cacheNS != reviewModelProviderCacheNamespace {
		t.Fatalf("provider cache namespace = %q, want %q", captured.cacheNS, reviewModelProviderCacheNamespace)
	}
}

func TestAgentReviewModelCompleteReviewWrapsProviderErrorWithPhase(t *testing.T) {
	provider := &scriptedChatProvider{
		name: "openai",
		chatWithToolsFn: func(_ int, _ context.Context, _ string, _ []api.Message, _ string) (string, error) {
			return "", errors.New("provider failed")
		},
	}
	agent := newReviewAgentForTest(t, provider)

	_, err := (agentReviewModel{agent: agent}).CompleteReview(context.Background(), review.ReviewModelRequest{
		Phase:  review.ReviewModelPhaseProbePlan,
		Prompt: "plan",
	})
	if err == nil {
		t.Fatal("CompleteReview() error = nil, want provider error")
	}
	if got := err.Error(); !strings.Contains(got, "review model probe_plan") || !strings.Contains(got, "provider failed") {
		t.Fatalf("CompleteReview() error = %q, want phase and provider error", got)
	}
}

func TestAgentReviewModelCompleteReviewRestoresResponseID(t *testing.T) {
	provider := &reviewResponseIDProvider{}
	provider.name = "openai"
	provider.responseID = "resp_original"
	provider.chatWithToolsFn = func(_ int, _ context.Context, _ string, _ []api.Message, _ string) (string, error) {
		if got := provider.GetResponseID(); got != "" {
			t.Fatalf("response ID during review call = %q, want empty", got)
		}
		provider.SetResponseID("resp_review")
		return `{"ok":true}`, nil
	}
	agent := newReviewAgentForTest(t, provider)

	if _, err := (agentReviewModel{agent: agent}).CompleteReview(context.Background(), review.ReviewModelRequest{
		Phase:  review.ReviewModelPhaseReport,
		Prompt: "report",
	}); err != nil {
		t.Fatalf("CompleteReview() error = %v", err)
	}
	if got := provider.GetResponseID(); got != "resp_original" {
		t.Fatalf("response ID after review call = %q, want resp_original", got)
	}
}

func TestAgentRunReviewUsesRunnerAndDoesNotMutateConversation(t *testing.T) {
	repo := setupReviewGitRepo(t)
	t.Chdir(repo)

	provider := &scriptedChatProvider{name: "openai"}
	provider.chatWithToolsFn = func(call int, _ context.Context, _ string, history []api.Message, model string) (string, error) {
		if model != "review-model" {
			t.Fatalf("model = %q, want review-model", model)
		}
		if len(history) != 1 || history[0].Role != "user" {
			t.Fatalf("history = %#v, want single review prompt", history)
		}
		switch call {
		case 0:
			return mustMarshalReviewValueForAgentTest(t, review.ReviewProbePlan{
				SchemaVersion: review.ReviewProbePlanSchemaVersionV1,
				TargetKind:    review.TargetCurrentChanges,
				Probes:        []review.ReviewPlannedProbe{},
				NoProbeReason: "No additional probe is needed.",
			}), nil
		case 1:
			return mustMarshalReviewValueForAgentTest(t, review.ReviewReport{
				SchemaVersion:             review.ReviewReportSchemaVersionV1,
				TargetKind:                review.TargetCurrentChanges,
				GeneratedAt:               time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
				OverallVerificationStatus: review.ReviewVerificationVerified,
				Verdict:                   review.ReviewVerdictClean,
				Summary:                   "No findings.",
			}), nil
		default:
			t.Fatalf("unexpected provider call %d", call)
			return "", nil
		}
	}
	agent := newReviewAgentForTest(t, provider)
	agent.History = []api.Message{{Role: "user", Content: "existing chat"}}
	agent.session = history.NewSession("review-model")
	agent.session.AddMessage("user", "existing chat", "review-model")

	report, err := agent.RunReview(context.Background(), review.NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("RunReview() error = %v", err)
	}
	if report.Verdict != review.ReviewVerdictClean {
		t.Fatalf("report verdict = %q, want clean", report.Verdict)
	}
	if provider.callCount != 2 {
		t.Fatalf("provider calls = %d, want 2", provider.callCount)
	}
	if len(agent.History) != 1 || agent.History[0].Content != "existing chat" {
		t.Fatalf("agent history mutated: %#v", agent.History)
	}
	if got := len(agent.session.Messages); got != 1 {
		t.Fatalf("session messages = %d, want 1", got)
	}
}

func TestAgentRunReviewRepairsInvalidModelJSONAndPreservesReviewIsolation(t *testing.T) {
	repo := setupReviewGitRepo(t)
	t.Chdir(repo)

	var prompts []string
	var toolUseDisabled []bool
	var toolCounts []int
	var updateModes []string
	provider := &scriptedChatProvider{name: "openai"}
	provider.chatWithToolsFn = func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
		if systemPrompt != "" {
			t.Fatalf("systemPrompt = %q, want empty", systemPrompt)
		}
		if model != "review-model" {
			t.Fatalf("model = %q, want review-model", model)
		}
		if len(history) != 1 || history[0].Role != "user" {
			t.Fatalf("history = %#v, want single review prompt", history)
		}
		prompts = append(prompts, history[0].Content)
		toolUseDisabled = append(toolUseDisabled, api.IsToolUseDisabled(ctx))
		toolCounts = append(toolCounts, len(api.ToolDefinitionsFromContext(ctx)))
		updateModes = append(updateModes, api.AssistantUpdateModeFromContext(ctx))

		switch call {
		case 0:
			return `{not-json`, nil
		case 1:
			return mustMarshalReviewValueForAgentTest(t, review.ReviewProbePlan{
				SchemaVersion: review.ReviewProbePlanSchemaVersionV1,
				TargetKind:    review.TargetCurrentChanges,
				Probes:        []review.ReviewPlannedProbe{},
				NoProbeReason: "No additional probe is needed.",
			}), nil
		case 2:
			return `{"schema_version":"review_report.v1"`, nil
		case 3:
			return mustMarshalReviewValueForAgentTest(t, review.ReviewReport{
				SchemaVersion:             review.ReviewReportSchemaVersionV1,
				TargetKind:                review.TargetCurrentChanges,
				GeneratedAt:               time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
				OverallVerificationStatus: review.ReviewVerificationVerified,
				Verdict:                   review.ReviewVerdictClean,
				Summary:                   "No findings.",
			}), nil
		default:
			t.Fatalf("unexpected provider call %d", call)
			return "", nil
		}
	}
	agent := newReviewAgentForTest(t, provider)
	agent.History = []api.Message{{Role: "user", Content: "existing chat"}}
	agent.session = history.NewSession("review-model")
	agent.session.AddMessage("user", "existing chat", "review-model")

	report, err := agent.RunReview(context.Background(), review.NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("RunReview() error = %v", err)
	}
	if report.Verdict != review.ReviewVerdictClean {
		t.Fatalf("report verdict = %q, want clean", report.Verdict)
	}
	if provider.callCount != 4 {
		t.Fatalf("provider calls = %d, want 4", provider.callCount)
	}
	if len(prompts) != 4 {
		t.Fatalf("captured prompts = %d, want 4", len(prompts))
	}
	for _, want := range []string{"Probe Plan JSON Repair", "{not-json"} {
		if !strings.Contains(prompts[1], want) {
			t.Fatalf("probe plan repair prompt missing %q:\n%s", want, prompts[1])
		}
	}
	for _, want := range []string{"Report JSON Repair", `{"schema_version":"review_report.v1"`} {
		if !strings.Contains(prompts[3], want) {
			t.Fatalf("report repair prompt missing %q:\n%s", want, prompts[3])
		}
	}
	for i := range prompts {
		if !toolUseDisabled[i] {
			t.Fatalf("call %d tool use disabled = false, want true", i)
		}
		if toolCounts[i] != 0 {
			t.Fatalf("call %d tool definitions = %d, want 0", i, toolCounts[i])
		}
		if updateModes[i] != api.AssistantUpdatesOff {
			t.Fatalf("call %d assistant update mode = %q, want off", i, updateModes[i])
		}
	}
	if len(agent.History) != 1 || agent.History[0].Content != "existing chat" {
		t.Fatalf("agent history mutated: %#v", agent.History)
	}
	if got := len(agent.session.Messages); got != 1 {
		t.Fatalf("session messages = %d, want 1", got)
	}
}

func TestAgentRunReviewCanBeCanceledThroughActiveRequest(t *testing.T) {
	repo := setupReviewGitRepo(t)
	t.Chdir(repo)

	started := make(chan struct{})
	var startedOnce sync.Once
	provider := &scriptedChatProvider{name: "openai"}
	provider.chatWithToolsFn = func(_ int, ctx context.Context, _ string, _ []api.Message, _ string) (string, error) {
		startedOnce.Do(func() { close(started) })
		<-ctx.Done()
		return "", ctx.Err()
	}
	agent := newReviewAgentForTest(t, provider)

	done := make(chan error, 1)
	go func() {
		_, err := agent.RunReview(context.Background(), review.NewCurrentChangesRequest(""))
		done <- err
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("RunReview did not reach provider call")
	}

	agent.cancelActiveRequest("review cancel")

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("RunReview() error = nil, want cancellation error")
		}
		if !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("RunReview() error = %q, want context canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunReview did not stop after cancelActiveRequest")
	}
	if agent.cancelFunc != nil {
		t.Fatal("cancelFunc should be cleared after RunReview finishes")
	}
	if agent.requestCtx != nil {
		t.Fatal("requestCtx should be cleared after RunReview finishes")
	}
}

type reviewResponseIDProvider struct {
	scriptedChatProvider
	responseID string
}

func (p *reviewResponseIDProvider) HasCachedResponseID() bool {
	return p.responseID != ""
}

func (p *reviewResponseIDProvider) SetResponseID(id string) {
	p.responseID = strings.TrimSpace(id)
}

func (p *reviewResponseIDProvider) GetResponseID() string {
	return p.responseID
}

func newReviewAgentForTest(t *testing.T, provider api.Provider) *Agent {
	t.Helper()
	var out bytes.Buffer
	agent := newChatRequestTestAgent(t, provider, &out)
	agent.CurrentModel = "review-model"
	agent.Model = "review-model"
	return agent
}

func setupReviewGitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGitForReviewTest(t, repo, "init")
	if err := os.WriteFile(repo+"/main.go", []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write initial file: %v", err)
	}
	runGitForReviewTest(t, repo, "add", "main.go")
	runGitForReviewTest(t, repo, "-c", "user.name=Review Test", "-c", "user.email=review@example.test", "commit", "-m", "initial")
	if err := os.WriteFile(repo+"/main.go", []byte("package main\n\nfunc main() { println(\"review\") }\n"), 0o644); err != nil {
		t.Fatalf("write changed file: %v", err)
	}
	return repo
}

func runGitForReviewTest(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func mustMarshalReviewValueForAgentTest(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return string(data)
}
