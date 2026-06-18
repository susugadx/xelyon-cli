package tuiagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
	agentpkg "github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/review"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

const reviewRunArtifactsEnv = "XELYON_REVIEW_RUN_ARTIFACTS"

func disableColors(t *testing.T) {
	t.Helper()
	prev := color.NoColor
	color.NoColor = true
	t.Cleanup(func() {
		color.NoColor = prev
	})
}

func newProjectMapDisabledConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.ProjectMap.Enabled = false
	cfg.MCP.Enabled = false
	cfg.LSP.Enabled = false
	return cfg
}

func newChatRequestTestConfig() *config.Config {
	cfg := newProjectMapDisabledConfig()
	cfg.MCP.Enabled = false
	cfg.LSP.Enabled = false
	cfg.Compression.Enabled = false
	cfg.Compression.KeepRecent = 10
	return cfg
}

func newChatRequestTestAgent(t *testing.T, provider api.Provider, out *bytes.Buffer) *agentpkg.Agent {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	runtime := agentpkg.NewAgentRuntimeWithConfig(newChatRequestTestConfig())
	runtime.UI = uiruntime.NewRuntime(strings.NewReader(""), out, out)
	runtime.Registry = tools.DefaultRegistry.Clone()
	runtime.AutoApprove = true

	agent := agentpkg.NewAgentWithRuntime("gpt-5.4", provider, false, runtime)
	agent.AutoApprove = true
	return agent
}

type scriptedChatProvider struct {
	name            string
	functionCalling bool
	callCount       int
	usageCallback   api.UsageCallback
	chatWithToolsFn func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error)
}

func (p *scriptedChatProvider) Name() string {
	if p.name != "" {
		return p.name
	}
	return "openai"
}

func (p *scriptedChatProvider) SupportsImages() bool { return false }

func (p *scriptedChatProvider) IsFunctionCallingEnabled() bool { return p.functionCalling }

func (p *scriptedChatProvider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}

func (p *scriptedChatProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	call := p.callCount
	p.callCount++
	if p.chatWithToolsFn != nil {
		return p.chatWithToolsFn(call, ctx, systemPrompt, history, model)
	}
	return "done", nil
}

func (p *scriptedChatProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

type imageOnceProvider struct {
	imageCalls int
}

func (p *imageOnceProvider) Name() string { return "openai" }

func (p *imageOnceProvider) SupportsImages() bool { return true }

func (p *imageOnceProvider) IsFunctionCallingEnabled() bool { return true }

func (p *imageOnceProvider) ChatWithTools(context.Context, string, []api.Message, string) (string, error) {
	return "", fmt.Errorf("ChatWithTools should not be called for image one-shot")
}

func (p *imageOnceProvider) ChatWithImage(_ context.Context, _ string, _ []api.Message, _ string, image *api.ImageData, _ string) (string, error) {
	if image == nil {
		return "", fmt.Errorf("image is required")
	}
	p.imageCalls++
	return "mock image response", nil
}

type mockErrorProvider struct{}

func (m *mockErrorProvider) Name() string                   { return "test-error" }
func (m *mockErrorProvider) SupportsImages() bool           { return false }
func (m *mockErrorProvider) IsFunctionCallingEnabled() bool { return false }
func (m *mockErrorProvider) ChatWithTools(context.Context, string, []api.Message, string) (string, error) {
	return "", fmt.Errorf("mock error")
}
func (m *mockErrorProvider) ChatWithImage(context.Context, string, []api.Message, string, *api.ImageData, string) (string, error) {
	return "", fmt.Errorf("mock error")
}

type mockProvider struct {
	name      string
	configKey string
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) ProviderConfigKey() string {
	if m.configKey != "" {
		return m.configKey
	}
	return m.name
}
func (m *mockProvider) SetProviderConfigKey(key string) { m.configKey = key }
func (m *mockProvider) ChatWithTools(context.Context, string, []api.Message, string) (string, error) {
	return "", nil
}
func (m *mockProvider) SupportsImages() bool { return false }
func (m *mockProvider) ChatWithImage(context.Context, string, []api.Message, string, *api.ImageData, string) (string, error) {
	return "", nil
}
func (m *mockProvider) IsFunctionCallingEnabled() bool { return false }

func newReviewAgentForTest(t *testing.T, provider api.Provider) *agentpkg.Agent {
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

func newAgentNoProbeReviewPlanForTest(surfaceSummary, riskSummary string) review.ReviewProbePlan {
	return review.ReviewProbePlan{
		SchemaVersion: review.ReviewProbePlanSchemaVersionV2,
		TargetKind:    review.TargetCurrentChanges,
		ImpactSurfaces: []review.ReviewProbeImpactSurface{
			{
				ID:              "surface-1",
				Summary:         surfaceSummary,
				Category:        review.ReviewProbeImpactSurfaceChangedFile,
				EvidenceSummary: "Git evidence covers main.go.",
				Status:          review.ReviewProbeImpactSurfaceChecked,
				Reason:          "Existing evidence covers surface-1.",
			},
		},
		CandidateRisks: []review.ReviewProbeCandidateRisk{
			{
				ID:                   "risk-1",
				Summary:              riskSummary,
				Severity:             review.ReviewGroupSeverityMedium,
				SurfaceIDs:           []string{"surface-1"},
				EvidenceSummary:      "Existing evidence covers the path.",
				VerificationStrategy: "No additional probe is needed.",
				Status:               review.ReviewProbeCandidateRiskCheckedByEvidence,
			},
		},
		Probes:        []review.ReviewPlannedProbe{},
		NoProbeReason: "surface-1 and risk-1 are checked by existing evidence.",
	}
}

func newAgentCleanReviewReportForTest() review.ReviewReport {
	return review.ReviewReport{
		SchemaVersion:             review.ReviewReportSchemaVersionV2,
		TargetKind:                review.TargetCurrentChanges,
		GeneratedAt:               time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		OverallVerificationStatus: review.ReviewVerificationVerified,
		Verdict:                   review.ReviewVerdictClean,
		Summary:                   "No findings.",
		ScopeCoverage: &review.ReviewReportScopeCoverage{
			ReviewedImpactSurfaces: []review.ReviewReportImpactSurfaceCoverage{
				{
					SurfaceID: "surface-1",
					Status:    review.ReviewReportImpactSurfaceChecked,
					Summary:   "surface-1 was checked.",
				},
			},
			ReviewedCandidateRisks: []review.ReviewReportCandidateRiskCoverage{
				{
					RiskID:  "risk-1",
					Status:  review.ReviewReportCandidateRiskDismissed,
					Summary: "risk-1 was dismissed.",
				},
			},
		},
	}
}

func newAgentSaturatedReviewCheckForTest() review.ReviewSaturationCheck {
	return review.ReviewSaturationCheck{
		SchemaVersion:  review.ReviewSaturationCheckSchemaVersionV1,
		Status:         review.ReviewSaturationStatusSaturated,
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
