package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/review"
	reviewdomain "github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func TestAgentRunReviewUsesRunnerAndDoesNotMutateConversation(t *testing.T) {
	repo := setupReviewGitRepo(t)
	t.Chdir(repo)
	t.Setenv(reviewRunArtifactsEnv, "")

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
			return mustMarshalReviewValueForAgentTest(t, newAgentNoProbeReviewPlanForTest(
				"Agent review runner path.",
				"Agent runner could mutate conversation state.",
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
	agent.History = []api.Message{{Role: "user", Content: "existing chat"}}
	agent.session = history.NewSession("review-model")
	agent.session.AddMessage("user", "existing chat", "review-model")

	report, err := agent.RunReview(context.Background(), review.NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("RunReview() error = %v", err)
	}
	if report.Verdict != reviewreport.ReviewVerdictClean {
		t.Fatalf("report verdict = %q, want clean", report.Verdict)
	}
	if provider.callCount != 3 {
		t.Fatalf("provider calls = %d, want 3", provider.callCount)
	}
	if len(agent.History) != 1 || agent.History[0].Content != "existing chat" {
		t.Fatalf("agent history mutated: %#v", agent.History)
	}
	if got := len(agent.session.Messages); got != 1 {
		t.Fatalf("session messages = %d, want 1", got)
	}
	if _, err := os.Stat(filepath.Join(repo, ".xelyon", "review-runs")); !os.IsNotExist(err) {
		t.Fatalf("review artifacts directory exists without %s: err=%v", reviewRunArtifactsEnv, err)
	}
}

func TestAgentRunReviewArtifactEnvDoesNotCreateRepoFilesBeforeProbesComplete(t *testing.T) {
	repo := setupReviewGitRepo(t)
	t.Chdir(repo)
	t.Setenv(reviewRunArtifactsEnv, "1")

	provider := &scriptedChatProvider{name: "openai"}
	provider.chatWithToolsFn = func(call int, _ context.Context, _ string, history []api.Message, _ string) (string, error) {
		if len(history) != 1 || history[0].Role != "user" {
			t.Fatalf("history = %#v, want single review prompt", history)
		}
		switch call {
		case 0:
			return mustMarshalReviewValueForAgentTest(t, newAgentArtifactIsolationProbePlanForTest()), nil
		case 1:
			if _, err := os.Stat(filepath.Join(repo, ".xelyon")); !os.IsNotExist(err) {
				t.Fatalf(".xelyon exists before report phase flush: err=%v", err)
			}
			if strings.Contains(history[0].Content, "review-runs") {
				t.Fatalf("report prompt contains repo-local review artifacts before probes completed:\n%s", history[0].Content)
			}
			return mustMarshalReviewValueForAgentTest(t, newAgentCleanReviewReportWithPassedProbeEvidenceForTest("probe-artifact-isolation")), nil
		case 2:
			return mustMarshalReviewValueForAgentTest(t, newAgentSaturatedReviewCheckForTest()), nil
		default:
			t.Fatalf("unexpected provider call %d", call)
			return "", nil
		}
	}
	agent := newReviewAgentForTest(t, provider)

	if _, err := agent.RunReview(context.Background(), review.NewCurrentChangesRequest("")); err != nil {
		t.Fatalf("RunReview() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".xelyon", "review-runs")); err != nil {
		t.Fatalf("review artifacts were not flushed after review completed: %v", err)
	}
}

func TestAgentRunReviewArtifactFlushFailureWarnsWithoutFailingReview(t *testing.T) {
	repo := setupReviewGitRepo(t)
	t.Chdir(repo)
	t.Setenv(reviewRunArtifactsEnv, "1")

	provider := &scriptedChatProvider{name: "openai"}
	provider.chatWithToolsFn = func(call int, _ context.Context, _ string, _ []api.Message, _ string) (string, error) {
		switch call {
		case 0:
			return mustMarshalReviewValueForAgentTest(t, newAgentNoProbeReviewPlanForTest(
				"Agent artifact flush failure path.",
				"Agent artifact flush failure could fail the review.",
			)), nil
		case 1:
			return mustMarshalReviewValueForAgentTest(t, newAgentCleanReviewReportForTest()), nil
		case 2:
			if err := os.WriteFile(filepath.Join(repo, ".xelyon"), []byte("not a directory\n"), 0o600); err != nil {
				t.Fatalf("write blocking .xelyon file: %v", err)
			}
			return mustMarshalReviewValueForAgentTest(t, newAgentSaturatedReviewCheckForTest()), nil
		default:
			t.Fatalf("unexpected provider call %d", call)
			return "", nil
		}
	}
	agent, out := newReviewAgentWithOutputForTest(t, provider)

	report, err := agent.RunReview(context.Background(), review.NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("RunReview() error = %v, want nil despite artifact flush failure", err)
	}
	if report.Verdict != reviewreport.ReviewVerdictClean {
		t.Fatalf("report verdict = %q, want clean", report.Verdict)
	}
	if !strings.Contains(out.String(), "Warning: failed to save review artifact") {
		t.Fatalf("warning output = %q, want artifact warning", out.String())
	}
}

func TestAgentRunReviewArtifactFlushRejectsSymlinkedRepoArtifactDir(t *testing.T) {
	repo := setupReviewGitRepo(t)
	t.Chdir(repo)
	t.Setenv(reviewRunArtifactsEnv, "1")
	outside := t.TempDir()

	provider := &scriptedChatProvider{name: "openai"}
	provider.chatWithToolsFn = func(call int, _ context.Context, _ string, _ []api.Message, _ string) (string, error) {
		switch call {
		case 0:
			return mustMarshalReviewValueForAgentTest(t, newAgentNoProbeReviewPlanForTest(
				"Agent artifact symlink rejection path.",
				"Agent artifact flush could follow a repo-controlled symlink.",
			)), nil
		case 1:
			return mustMarshalReviewValueForAgentTest(t, newAgentCleanReviewReportForTest()), nil
		case 2:
			createReviewArtifactSymlinkForAgentTest(t, outside, filepath.Join(repo, ".xelyon"))
			return mustMarshalReviewValueForAgentTest(t, newAgentSaturatedReviewCheckForTest()), nil
		default:
			t.Fatalf("unexpected provider call %d", call)
			return "", nil
		}
	}
	agent, out := newReviewAgentWithOutputForTest(t, provider)

	report, err := agent.RunReview(context.Background(), review.NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("RunReview() error = %v, want nil despite artifact flush failure", err)
	}
	if report.Verdict != reviewreport.ReviewVerdictClean {
		t.Fatalf("report verdict = %q, want clean", report.Verdict)
	}
	if !strings.Contains(out.String(), "Warning: failed to save review artifact") || !strings.Contains(out.String(), "symlink") {
		t.Fatalf("warning output = %q, want artifact symlink warning", out.String())
	}
	if _, err := os.Stat(filepath.Join(outside, "review-runs")); !os.IsNotExist(err) {
		t.Fatalf("artifact escaped through .xelyon symlink: err=%v", err)
	}
}

func TestAgentRunReviewSavesArtifactsWhenEnvEnabled(t *testing.T) {
	repo := setupReviewGitRepo(t)
	t.Chdir(repo)
	t.Setenv(reviewRunArtifactsEnv, "1")

	provider := &scriptedChatProvider{name: "openai"}
	provider.chatWithToolsFn = func(call int, _ context.Context, _ string, _ []api.Message, _ string) (string, error) {
		switch call {
		case 0:
			return mustMarshalReviewValueForAgentTest(t, newAgentNoProbeReviewPlanForTest(
				"Agent artifact runner path.",
				"Agent artifact runner could skip debug output.",
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

	if _, err := agent.RunReview(context.Background(), review.NewCurrentChangesRequest("")); err != nil {
		t.Fatalf("RunReview() error = %v", err)
	}

	runsRoot := filepath.Join(repo, ".xelyon", "review-runs")
	entries, err := os.ReadDir(runsRoot)
	if err != nil {
		t.Fatalf("ReadDir(%s) error = %v", runsRoot, err)
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		t.Fatalf("review-runs entries = %#v, want one run directory", entries)
	}
	runDir := filepath.Join(runsRoot, entries[0].Name())
	for _, name := range []string{
		"evidence.md",
		"probe_plan_prompt.md",
		"probe_plan_raw.json",
		"probe_plan_final.json",
		"probe_requests.json",
		"probe_results.json",
		"report_prompt.md",
		"report_raw.json",
		"report_final.json",
		"saturation_prompt.md",
		"saturation_raw.json",
	} {
		if _, err := os.Stat(filepath.Join(runDir, name)); err != nil {
			t.Fatalf("expected artifact %s: %v", name, err)
		}
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
			return mustMarshalReviewValueForAgentTest(t, newAgentNoProbeReviewPlanForTest(
				"Agent repair path.",
				"Repair path could lose context.",
			)), nil
		case 2:
			return `{"schema_version":"review_report.v2"`, nil
		case 3:
			return mustMarshalReviewValueForAgentTest(t, newAgentCleanReviewReportForTest()), nil
		case 4:
			return mustMarshalReviewValueForAgentTest(t, newAgentSaturatedReviewCheckForTest()), nil
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
	if report.Verdict != reviewreport.ReviewVerdictClean {
		t.Fatalf("report verdict = %q, want clean", report.Verdict)
	}
	if provider.callCount != 5 {
		t.Fatalf("provider calls = %d, want 5", provider.callCount)
	}
	if len(prompts) != 5 {
		t.Fatalf("captured prompts = %d, want 5", len(prompts))
	}
	for _, want := range []string{"Probe Plan JSON Repair", "{not-json"} {
		if !strings.Contains(prompts[1], want) {
			t.Fatalf("probe plan repair prompt missing %q:\n%s", want, prompts[1])
		}
	}
	for _, want := range []string{"Report JSON Repair", `{"schema_version":"review_report.v2"`} {
		if !strings.Contains(prompts[3], want) {
			t.Fatalf("report repair prompt missing %q:\n%s", want, prompts[3])
		}
	}
	for _, want := range []string{"Review Final Report Saturation Check", "review_saturation_check.v1"} {
		if !strings.Contains(prompts[4], want) {
			t.Fatalf("saturation prompt missing %q:\n%s", want, prompts[4])
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

func newReviewAgentForTest(t *testing.T, provider api.Provider) *Agent {
	t.Helper()
	agent, _ := newReviewAgentWithOutputForTest(t, provider)
	return agent
}

func newReviewAgentWithOutputForTest(t *testing.T, provider api.Provider) (*Agent, *bytes.Buffer) {
	t.Helper()
	var out bytes.Buffer
	agent := newChatRequestTestAgent(t, provider, &out)
	agent.CurrentModel = "review-model"
	agent.Model = "review-model"
	return agent, &out
}

func setupReviewGitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGitForReviewTest(t, repo, "init")
	if err := os.WriteFile(repo+"/main.go", []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write initial file: %v", err)
	}
	if err := os.WriteFile(repo+"/helper.go", []byte("package main\n\nfunc helper() { main() }\n"), 0o644); err != nil {
		t.Fatalf("write helper file: %v", err)
	}
	runGitForReviewTest(t, repo, "add", "main.go", "helper.go")
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

func createReviewArtifactSymlinkForAgentTest(t *testing.T, oldname, newname string) {
	t.Helper()

	if err := os.Symlink(oldname, newname); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}
}

func newAgentNoProbeReviewPlanForTest(surfaceSummary, riskSummary string) reviewprobeplan.ReviewProbePlan {
	return reviewprobeplan.ReviewProbePlan{
		SchemaVersion: reviewprobeplan.ReviewProbePlanSchemaVersionV2,
		TargetKind:    reviewdomain.TargetCurrentChanges,
		ImpactSurfaces: []reviewprobeplan.ReviewProbeImpactSurface{
			{
				ID:              "surface-1",
				Summary:         surfaceSummary,
				Category:        reviewprobeplan.ReviewProbeImpactSurfaceChangedFile,
				EvidenceSummary: "Git evidence covers main.go.",
				Status:          reviewprobeplan.ReviewProbeImpactSurfaceChecked,
				Reason:          "Existing evidence covers surface-1.",
			},
		},
		CandidateRisks: []reviewprobeplan.ReviewProbeCandidateRisk{
			{
				ID:                   "risk-1",
				Summary:              riskSummary,
				Severity:             reviewreport.ReviewGroupSeverityMedium,
				SurfaceIDs:           []string{"surface-1"},
				EvidenceSummary:      "Existing evidence covers the path.",
				VerificationStrategy: "No additional probe is needed.",
				Status:               reviewprobeplan.ReviewProbeCandidateRiskCheckedByEvidence,
			},
		},
		Probes:        []reviewprobeplan.ReviewPlannedProbe{},
		NoProbeReason: "surface-1 and risk-1 are checked by existing evidence.",
	}
}

func newAgentArtifactIsolationProbePlanForTest() reviewprobeplan.ReviewProbePlan {
	plan := newAgentNoProbeReviewPlanForTest(
		"Agent artifact runner path covers main.go production changes and untracked debug artifact pressure.",
		"Agent artifact runner could expose debug files to review probes.",
	)
	plan.ImpactSurfaces[0].Status = reviewprobeplan.ReviewProbeImpactSurfaceNeedsProbe
	plan.ImpactSurfaces[0].Reason = "A host-readonly probe should verify repo-local debug artifacts are absent before artifact flush."
	plan.CandidateRisks[0].Status = reviewprobeplan.ReviewProbeCandidateRiskNeedsProbe
	plan.CandidateRisks[0].VerificationStrategy = "Run a read-only find scoped to .xelyon and confirm no review-run paths are visible."
	plan.NoProbeReason = ""
	plan.Probes = []reviewprobeplan.ReviewPlannedProbe{
		{
			ID:         "probe-artifact-isolation",
			SurfaceIDs: []string{"surface-1"},
			RiskIDs:    []string{"risk-1"},
			Purpose:    "Verify debug review artifacts do not appear in the repository before probes complete.",
			Mode:       reviewdomain.ReviewProbeHostReadOnly,
			Commands: []reviewprobeplan.ReviewPlannedProbeCommand{
				{
					Command: "find",
					Args:    []string{".", "-path", "./.xelyon/*", "-print"},
					WorkDir: ".",
				},
			},
			TimeoutSeconds: 2,
			MaxOutputBytes: 1024,
		},
	}
	return plan
}

func newAgentCleanReviewReportForTest() reviewreport.ReviewReport {
	return reviewreport.ReviewReport{
		SchemaVersion:             reviewreport.ReviewReportSchemaVersionV2,
		TargetKind:                reviewdomain.TargetCurrentChanges,
		GeneratedAt:               time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		OverallVerificationStatus: reviewreport.ReviewVerificationVerified,
		Verdict:                   reviewreport.ReviewVerdictClean,
		Summary:                   "No findings.",
		ScopeCoverage: &reviewreport.ReviewReportScopeCoverage{
			ReviewedImpactSurfaces: []reviewreport.ReviewReportImpactSurfaceCoverage{
				{
					SurfaceID: "surface-1",
					Status:    reviewreport.ReviewReportImpactSurfaceChecked,
					Summary:   "surface-1 was checked.",
				},
			},
			ReviewedCandidateRisks: []reviewreport.ReviewReportCandidateRiskCoverage{
				{
					RiskID:  "risk-1",
					Status:  reviewreport.ReviewReportCandidateRiskDismissed,
					Summary: "risk-1 was dismissed.",
				},
			},
		},
	}
}

func newAgentCleanReviewReportWithPassedProbeEvidenceForTest(probeID string) reviewreport.ReviewReport {
	report := newAgentCleanReviewReportForTest()
	ref := reviewreport.ReviewEvidenceRef{
		Kind:    reviewreport.ReviewEvidenceKindProbe,
		ProbeID: probeID,
		Summary: "The linked probe passed.",
	}
	report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{ref}
	report.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []reviewreport.ReviewEvidenceRef{ref}
	return report
}

func newAgentSaturatedReviewCheckForTest() reviewreport.ReviewSaturationCheck {
	return reviewreport.ReviewSaturationCheck{
		SchemaVersion:  reviewreport.ReviewSaturationCheckSchemaVersionV1,
		Status:         reviewreport.ReviewSaturationStatusSaturated,
		CheckedSummary: "Final report covers Pass1 scope.",
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
