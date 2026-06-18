package agent

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	promptplan "github.com/susugadx/xelyon-cli/internal/prompt/plan"
	"github.com/susugadx/xelyon-cli/internal/skills/router"
	"github.com/susugadx/xelyon-cli/internal/skills/usageledger"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func TestParseSkillRouterStatusPathsCapsAndParsesRenames(t *testing.T) {
	output := " M internal/agent/a.go\nR  old.go -> internal/agent/new.go\n?? docs/plan.md\n M extra.go\n"
	paths, capped := parseSkillRouterStatusPaths(output, 3)
	if !capped {
		t.Fatal("capped = false, want true")
	}
	want := []string{"internal/agent/a.go", "internal/agent/new.go", "docs/plan.md"}
	if len(paths) != len(want) {
		t.Fatalf("paths = %#v, want %#v", paths, want)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("paths[%d] = %q, want %q", i, paths[i], want[i])
		}
	}
}

func TestCollectSkillRouterTouchedPathsFailureSafeSignals(t *testing.T) {
	paths, diagnostics := collectSkillRouterTouchedPaths(context.Background(), "")
	if len(paths) != 0 || !hasSkillRouterDiagnostic(diagnostics, "cwd unavailable") {
		t.Fatalf("empty cwd paths=%#v diagnostics=%#v, want cwd diagnostic", paths, diagnostics)
	}

	paths, capped := parseSkillRouterStatusPaths("", 200)
	if capped || len(paths) != 0 {
		t.Fatalf("empty git status paths=%#v capped=%v, want no paths and no cap", paths, capped)
	}

	paths, diagnostics = collectSkillRouterTouchedPaths(context.Background(), t.TempDir())
	if len(paths) != 0 || !hasSkillRouterDiagnostic(diagnostics, "git status unavailable") {
		t.Fatalf("non-git cwd paths=%#v diagnostics=%#v, want git unavailable diagnostic", paths, diagnostics)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	paths, diagnostics = collectSkillRouterTouchedPaths(ctx, t.TempDir())
	if len(paths) != 0 || !hasSkillRouterDiagnostic(diagnostics, "git status timed out") {
		t.Fatalf("canceled context paths=%#v diagnostics=%#v, want timeout diagnostic", paths, diagnostics)
	}
}

func TestUsageSummariesFromRecommendationUsesRuntimeLimits(t *testing.T) {
	rec := router.Recommendation{
		Primary: []router.Candidate{
			{Name: "p1", Category: router.CategoryPrimary, Score: 90, Confidence: router.ConfidenceHigh, Activation: "hint"},
			{Name: "p2", Category: router.CategoryPrimary, Score: 85, Confidence: router.ConfidenceHigh, Activation: "hint"},
			{Name: "p3", Category: router.CategoryPrimary, Score: 84, Confidence: router.ConfidenceHigh, Activation: "hint"},
		},
		Maybe: []router.Candidate{
			{Name: "m1", Category: router.CategoryMaybe, Score: 49, Confidence: router.ConfidenceLow, Activation: "hint"},
		},
	}

	summaries := usageSummariesFromRecommendation(rec, router.DefaultRuntimeHintLimits())
	if len(summaries) != 2 {
		t.Fatalf("summaries = %#v, want two primary summaries", summaries)
	}
	if summaries[0].Name != "p1" || summaries[1].Name != "p2" {
		t.Fatalf("summaries = %#v, want p1/p2 only", summaries)
	}
}

func TestUsageSummariesFromRecommendationExcludesConflictsAndMaybe(t *testing.T) {
	rec := router.Recommendation{
		Primary: []router.Candidate{
			{Name: "p1", Category: router.CategoryPrimary, Score: 90, Confidence: router.ConfidenceHigh, Activation: "hint"},
		},
		Supporting: []router.Candidate{
			{Name: "s1", Category: router.CategorySupporting, Score: 70, Confidence: router.ConfidenceMedium, Activation: "hint"},
		},
		Conflicts: []router.Candidate{
			{Name: "c1", Category: router.CategoryConflict, Score: 80, Confidence: router.ConfidenceHigh, Activation: "hint"},
		},
		Maybe: []router.Candidate{
			{Name: "m1", Category: router.CategoryMaybe, Score: 80, Confidence: router.ConfidenceHigh, Activation: "hint"},
		},
	}

	summaries := usageSummariesFromRecommendation(rec, router.HintLimits{
		Primary:    2,
		Supporting: 5,
		Conflict:   5,
		Maybe:      5,
	})
	if len(summaries) != 2 {
		t.Fatalf("summaries = %#v, want primary/supporting only", summaries)
	}
	if summaries[0].Name != "p1" || summaries[1].Name != "s1" {
		t.Fatalf("summaries = %#v, want p1/s1 only", summaries)
	}
}

func TestSkillRouterProjectRootFallsBackToInvocationCWDGitRoot(t *testing.T) {
	repo := t.TempDir()
	initSkillRoutingGitRepo(t, repo)
	invocationCWD := filepath.Join(repo, "pkg", "nested")
	if err := os.MkdirAll(invocationCWD, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	cfg := config.DefaultConfig()
	runtime := &AgentRuntime{
		Config:        cfg,
		InvocationCWD: invocationCWD,
	}
	agent := NewAgentWithRuntime("model", &mockProvider{name: "mock"}, false, runtime)
	agent.projectMapRootPath = ""

	got := agent.skillRouterProjectRoot()
	want, err := filepath.Abs(repo)
	if err != nil {
		t.Fatalf("Abs(repo) error = %v", err)
	}
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("skillRouterProjectRoot() = %q, want git root %q", got, want)
	}
}

func TestRecordSkillActivationFromToolResultRecordsRepoScopedActivation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	initSkillRoutingGitRepo(t, repo)
	invocationCWD := filepath.Join(repo, "pkg", "nested")
	if err := os.MkdirAll(invocationCWD, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Skills.Router.UsageLedger = true
	agent := NewAgentWithRuntime("model", &mockProvider{name: "mock"}, false, &AgentRuntime{
		Config:        cfg,
		InvocationCWD: invocationCWD,
	})
	agent.projectMapRootPath = ""

	agent.recordSkillActivationFromToolResult(
		&tools.ToolCall{Tool: "activate_skill", Args: map[string]string{"name": "demo"}},
		`{"name": "demo"}`,
		false,
	)

	store := usageledger.NewStore(usageledger.Options{
		StateHome:   filepath.Join(home, ".xelyon"),
		ProjectRoot: repo,
		Enabled:     true,
	})
	summary, err := store.Summary()
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.Records != 1 || len(summary.Skills) != 1 || summary.Skills[0].Name != "demo" || summary.Skills[0].ActivatedCount != 1 {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestRecordSkillActivationFromToolResultSkipsWithoutProjectRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.DefaultConfig()
	cfg.Skills.Router.UsageLedger = true
	agent := NewAgentWithRuntime("model", &mockProvider{name: "mock"}, false, &AgentRuntime{
		Config:        cfg,
		InvocationCWD: t.TempDir(),
	})
	agent.projectMapRootPath = ""

	agent.recordSkillActivationFromToolResult(
		&tools.ToolCall{Tool: "activate_skill", Args: map[string]string{"name": "demo"}},
		`{"name": "demo"}`,
		false,
	)

	usageDir := filepath.Join(home, ".xelyon", "skills", "router", "usage")
	if _, err := os.Stat(usageDir); !os.IsNotExist(err) {
		t.Fatalf("usage dir state err=%v, want no ledger write", err)
	}
}

func TestRecordSkillRecommendationSkipsWithoutProjectRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.DefaultConfig()
	cfg.Skills.Router.UsageLedger = true
	agent := NewAgentWithRuntime("model", &mockProvider{name: "mock"}, false, &AgentRuntime{
		Config:        cfg,
		InvocationCWD: t.TempDir(),
	})
	agent.projectMapRootPath = ""

	agent.recordSkillRecommendation(router.Recommendation{
		Primary: []router.Candidate{{
			Name:       "demo",
			Category:   router.CategoryPrimary,
			Score:      90,
			Confidence: router.ConfidenceHigh,
			Activation: "hint",
		}},
	})

	usageDir := filepath.Join(home, ".xelyon", "skills", "router", "usage")
	if _, err := os.Stat(usageDir); !os.IsNotExist(err) {
		t.Fatalf("usage dir state err=%v, want no ledger write", err)
	}
}

func TestRecordSkillActivationFromToolResultSkipsErrors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	initSkillRoutingGitRepo(t, repo)

	cfg := config.DefaultConfig()
	cfg.Skills.Router.UsageLedger = true
	agent := NewAgentWithRuntime("model", &mockProvider{name: "mock"}, false, &AgentRuntime{
		Config:        cfg,
		InvocationCWD: repo,
	})

	agent.recordSkillActivationFromToolResult(
		&tools.ToolCall{Tool: "activate_skill", Args: map[string]string{"name": "demo"}},
		"Error: unknown skill: demo",
		false,
	)
	agent.recordSkillActivationFromToolResult(
		&tools.ToolCall{Tool: "activate_skill", Args: map[string]string{"name": "demo"}},
		`{"name": "demo"}`,
		true,
	)

	store := usageledger.NewStore(usageledger.Options{
		StateHome:   filepath.Join(home, ".xelyon"),
		ProjectRoot: repo,
		Enabled:     true,
	})
	summary, err := store.Summary()
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.Records != 0 || len(summary.Skills) != 0 {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestExecuteToolOnlyRecordsSkillActivation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	initSkillRoutingGitRepo(t, repo)
	writeSkillRoutingTestSkill(t, repo, "demo", "Demo skill description.")

	cfg := config.DefaultConfig()
	cfg.Skills.Router.UsageLedger = true
	runtime := &AgentRuntime{
		Config:        cfg,
		InvocationCWD: repo,
		UI:            uiruntime.NewRuntime(strings.NewReader(""), io.Discard, io.Discard),
	}
	agent := NewAgentWithRuntime("model", &mockProvider{name: "mock"}, false, runtime)

	result := agent.executeToolOnly(&tools.ToolCall{
		Tool: "activate_skill",
		Args: map[string]string{"name": "demo"},
	})
	if tools.IsErrorResult(result) {
		t.Fatalf("executeToolOnly(activate_skill) result = %q, want success", result)
	}

	store := usageledger.NewStore(usageledger.Options{
		StateHome:   filepath.Join(home, ".xelyon"),
		ProjectRoot: repo,
		Enabled:     true,
	})
	summary, err := store.Summary()
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.Records != 1 || len(summary.Skills) != 1 || summary.Skills[0].Name != "demo" || summary.Skills[0].ActivatedCount != 1 {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestInjectSkillRouterRuntimeHintHonorsConfigOff(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Skills.Router.Activation = config.SkillsRouterActivationOff
	agent := NewAgentWithRuntime("model", &mockProvider{name: "mock"}, false, NewAgentRuntimeWithConfig(cfg))

	prompt := agent.injectSkillRouterRuntimeHint(context.Background(), "base\n\n<!-- SKILL_ROUTER_HINT_START -->old<!-- SKILL_ROUTER_HINT_END -->", "review code", false)
	if strings.Contains(prompt, "SKILL_ROUTER_HINT_START") {
		t.Fatalf("prompt should strip hint block when activation is off:\n%s", prompt)
	}
	if strings.TrimSpace(prompt) != "base" {
		t.Fatalf("prompt = %q, want base", prompt)
	}
}

func TestInjectSkillRouterRuntimeHintKeepsHintOutOfPromptCacheStaticBlock(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = true
	first := router.Recommendation{Primary: []router.Candidate{{
		Name:       "first-skill",
		Category:   router.CategoryPrimary,
		Score:      90,
		Confidence: router.ConfidenceHigh,
		Activation: "hint",
		Reason:     "first reason",
	}}}
	second := router.Recommendation{Primary: []router.Candidate{{
		Name:       "second-skill",
		Category:   router.CategoryPrimary,
		Score:      90,
		Confidence: router.ConfidenceHigh,
		Activation: "hint",
		Reason:     "second reason",
	}}}

	prompt := injectSkillRouterRuntimeHintIntoSystemPrompt("static prompt", first, router.DefaultRuntimeHintLimits(), cfg)
	prompt = injectSkillRouterRuntimeHintIntoSystemPrompt(prompt, second, router.DefaultRuntimeHintLimits(), cfg)

	if strings.Count(prompt, "SKILL_ROUTER_HINT_START") != 1 {
		t.Fatalf("prompt should contain one runtime hint block:\n%s", prompt)
	}
	if strings.Contains(prompt, "first-skill") || !strings.Contains(prompt, "second-skill") {
		t.Fatalf("prompt should replace old runtime hint:\n%s", prompt)
	}
	field := api.BuildSystemFieldWithConfig(prompt, cfg)
	blocks, ok := field.([]api.SystemBlock)
	if !ok {
		t.Fatalf("BuildSystemFieldWithConfig() type = %T, want []api.SystemBlock", field)
	}
	if len(blocks) != 2 {
		t.Fatalf("system blocks = %d, want static + dynamic blocks", len(blocks))
	}
	if strings.Contains(blocks[0].Text, "SKILL_ROUTER_HINT_START") || strings.Contains(blocks[0].Text, "second-skill") {
		t.Fatalf("runtime hint should stay out of prompt-cache static block:\n%s", blocks[0].Text)
	}
	if !strings.Contains(blocks[1].Text, "SKILL_ROUTER_HINT_START") || !strings.Contains(blocks[1].Text, "second-skill") {
		t.Fatalf("runtime hint should be in dynamic block:\n%s", blocks[1].Text)
	}
}

func TestInjectSkillRouterRuntimeHintDoesNotCreateCacheBoundaryWhenPromptCacheDisabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = false
	rec := router.Recommendation{Primary: []router.Candidate{{
		Name:       "demo-skill",
		Category:   router.CategoryPrimary,
		Score:      90,
		Confidence: router.ConfidenceHigh,
		Activation: "hint",
		Reason:     "demo reason",
	}}}

	prompt := injectSkillRouterRuntimeHintIntoSystemPrompt("static prompt", rec, router.DefaultRuntimeHintLimits(), cfg)
	if strings.Contains(prompt, api.SystemPromptCacheBoundary) {
		t.Fatalf("prompt cache disabled should not create cache boundary:\n%s", prompt)
	}
	if !strings.Contains(prompt, "SKILL_ROUTER_HINT_START") || !strings.Contains(prompt, "demo-skill") {
		t.Fatalf("runtime hint missing from prompt:\n%s", prompt)
	}
}

func TestPlanInvestigationInjectsSkillRouterHintPreservesPlanningPromptAndRecordsInitialRecommendation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	initSkillRoutingGitRepo(t, repo)
	writeSkillRoutingTestSkillWithSidecar(t, repo, "plan-implementation", "Plan implementation changes.", strings.Join([]string{
		"version: 1",
		"intents:",
		"  - implementation",
		"  - planning",
		"role: primary",
		"modes:",
		"  - implementation",
		"  - planning",
		"triggers:",
		"  - implement",
		"  - plan",
		"activation: hint",
		"",
	}, "\n"))

	cfg := newProjectMapDisabledConfig()
	cfg.Skills.Router.UsageLedger = true
	var systemPrompts []string
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			systemPrompts = append(systemPrompts, systemPrompt)
			return "investigation done", nil
		},
	}
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.InvocationCWD = repo
	runtime.UI = uiruntime.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	runtime.Registry = tools.DefaultRegistry.Clone()
	agent := NewAgentWithRuntime("model", provider, false, runtime)
	agent.SystemPrompt += "\n\n" + promptplan.BuildPlanningPrompt()
	agent.History = []api.Message{{Role: "user", Content: "investigation prompt"}}

	runner := newPlanInvestigationRunnerWithOptions(agent, context.Background(), planInvestigationOptions{taskText: "implement feature"})
	if _, err := runner.requestResponseForIteration(0); err != nil {
		t.Fatalf("requestResponseForIteration(0) error = %v", err)
	}
	if _, err := runner.requestResponseForIteration(1); err != nil {
		t.Fatalf("requestResponseForIteration(1) error = %v", err)
	}
	if len(systemPrompts) != 2 {
		t.Fatalf("len(systemPrompts) = %d, want 2", len(systemPrompts))
	}
	for i, systemPrompt := range systemPrompts {
		for _, fragment := range []string{"SKILL_ROUTER_HINT_START", "plan-implementation", "You are in Plan Mode - producing a text plan"} {
			if !strings.Contains(systemPrompt, fragment) {
				t.Fatalf("systemPrompts[%d] missing %q:\n%s", i, fragment, systemPrompt)
			}
		}
		if strings.Count(systemPrompt, "SKILL_ROUTER_HINT_START") != 1 {
			t.Fatalf("systemPrompts[%d] should contain one runtime hint block:\n%s", i, systemPrompt)
		}
	}

	store := usageledger.NewStore(usageledger.Options{
		StateHome:   filepath.Join(home, ".xelyon"),
		ProjectRoot: repo,
		Enabled:     true,
	})
	summary, err := store.Summary()
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.Records != 1 || len(summary.Skills) != 1 || summary.Skills[0].Name != "plan-implementation" || summary.Skills[0].RecommendedCount != 1 {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestRunHeadlessWithConfigInjectsSkillRouterHintAndRecordsRecommendation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	initSkillRoutingGitRepo(t, repo)
	writeSkillRoutingTestSkillWithSidecar(t, repo, "strict-review", "Review diffs and report actionable findings.", strings.Join([]string{
		"version: 1",
		"intents:",
		"  - code-review",
		"role: primary",
		"read_only: true",
		"modes:",
		"  - review",
		"triggers:",
		"  - review",
		"activation: hint",
		"",
	}, "\n"))

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("Chdir(repo) error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	cfg := newProjectMapDisabledConfig()
	cfg.Skills.Router.UsageLedger = true
	provider := &headlessToolSetProbeProvider{}

	result := RunHeadlessWithConfig(context.Background(), "review this diff", "gpt-5.4", provider, cfg)
	if result.Status != "success" {
		t.Fatalf("RunHeadlessWithConfig() status = %q, want success: %#v", result.Status, result.Error)
	}
	for _, fragment := range []string{"SKILL_ROUTER_HINT_START", "strict-review"} {
		if !strings.Contains(provider.systemPrompt, fragment) {
			t.Fatalf("headless system prompt missing %q:\n%s", fragment, provider.systemPrompt)
		}
	}

	store := usageledger.NewStore(usageledger.Options{
		StateHome:   filepath.Join(home, ".xelyon"),
		ProjectRoot: repo,
		Enabled:     true,
	})
	summary, err := store.Summary()
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.Records != 1 || len(summary.Skills) != 1 || summary.Skills[0].Name != "strict-review" || summary.Skills[0].RecommendedCount != 1 {
		t.Fatalf("summary = %#v", summary)
	}
}

func initSkillRoutingGitRepo(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "init")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v (%s)", err, string(out))
	}
}

func writeSkillRoutingTestSkill(t *testing.T, repo, name, description string) {
	t.Helper()
	skillDir := filepath.Join(repo, ".agents", "skills", name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(skillDir) error = %v", err)
	}
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\n# " + name + "\n\nUse this skill.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(SKILL.md) error = %v", err)
	}
}

func writeSkillRoutingTestSkillWithSidecar(t *testing.T, repo, name, description, sidecar string) {
	t.Helper()
	writeSkillRoutingTestSkill(t, repo, name, description)
	skillDir := filepath.Join(repo, ".agents", "skills", name)
	agentsDir := filepath.Join(skillDir, "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(agentsDir) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "xelyon.yaml"), []byte(sidecar), 0o644); err != nil {
		t.Fatalf("WriteFile(xelyon.yaml) error = %v", err)
	}
}

func hasSkillRouterDiagnostic(diagnostics []string, want string) bool {
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic, want) {
			return true
		}
	}
	return false
}
