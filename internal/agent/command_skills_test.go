package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/commandruntime"
	"github.com/susugadx/xelyon-cli/internal/skills/usageledger"
)

func TestHandleSkillsCommand_ListAndShow(t *testing.T) {
	disableColors(t)
	agent, out := newSurfaceDispatchTestAgent(t)

	skillDir := filepath.Join(".agents", "skills", "demo-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	content := "---\nname: demo\ndescription: demo description\n---\n# Demo\nUse demo.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	for path, content := range map[string]string{
		filepath.Join(skillDir, "scripts", "run.sh"):        "#!/usr/bin/env bash\necho run\n",
		filepath.Join(skillDir, "references", "guide.md"):   "guide",
		filepath.Join(skillDir, "assets", "screenshot.txt"): "asset",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", path, err)
		}
	}

	if !handleSpecialCommandForSurface("/skills overview", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills overview should be handled")
	}
	for _, fragment := range []string{"Agent Skills Overview", "Project skills", "demo", "demo description"} {
		if !strings.Contains(out.String(), fragment) {
			t.Fatalf("/skills overview output missing %q:\n%s", fragment, out.String())
		}
	}

	out.Reset()
	if !handleSpecialCommandForSurface("/skills list", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills list should be handled as overview alias")
	}
	if !strings.Contains(out.String(), "demo") {
		t.Fatalf("/skills list alias output missing catalog entry:\n%s", out.String())
	}

	out.Reset()
	if !handleSpecialCommandForSurface("/skills show demo", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills show should be handled")
	}
	got := out.String()
	for _, fragment := range []string{"Agent Skill: demo", "Name: demo", "Source: project", "Resources: 1 scripts, 1 references, 1 assets", "Description", "demo description", "SKILL.md", "# Demo", "Use demo.", "Resources", "scripts", "scripts/run.sh", "references", "references/guide.md", "assets", "assets/screenshot.txt"} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("/skills show output missing %q:\n%s", fragment, got)
		}
	}
	for _, fragment := range []string{`"name": "demo"`, `"skill_md"`, `\nUse demo.`} {
		if strings.Contains(got, fragment) {
			t.Fatalf("/skills show output should be human-readable, found raw JSON fragment %q:\n%s", fragment, got)
		}
	}
}

func TestHandleSkillsCommand_ShowQuotedSkillName(t *testing.T) {
	disableColors(t)
	agent, out := newSurfaceDispatchTestAgent(t)

	skillName := `my "quoted"  skill\`
	skillDir := filepath.Join(".agents", "skills", "quoted-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	content := "---\nname: 'my \"quoted\"  skill\\'\ndescription: quoted skill description\n---\n# Quoted\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	command := "/skills show " + commandruntime.QuoteArg(skillName)
	if !handleSpecialCommandForSurface(command, agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatalf("%s should be handled", command)
	}

	got := out.String()
	for _, fragment := range []string{"Agent Skill: " + skillName, "quoted skill description", "# Quoted"} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("/skills show output missing %q:\n%s", fragment, got)
		}
	}
}

func TestHandleSkillsCommand_OverviewWrapsDescriptionsWithoutTruncating(t *testing.T) {
	disableColors(t)
	agent, out := newSurfaceDispatchTestAgent(t)

	skillDir := filepath.Join(".agents", "skills", "long-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	description := strings.Repeat("長い説明文を折り返して表示する。", 10) + "最後まで表示される"
	content := "---\nname: long-skill\ndescription: " + description + "\n---\n# Long\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if !handleSpecialCommandForSurface("/skills overview", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills overview should be handled")
	}

	got := out.String()
	if !strings.Contains(got, "最後まで表示される") {
		t.Fatalf("/skills overview should not truncate long descriptions:\n%s", got)
	}
	if strings.Contains(got, "…") {
		t.Fatalf("/skills overview should wrap instead of ellipsizing:\n%s", got)
	}
	if !strings.Contains(got, "\n    ") {
		t.Fatalf("/skills overview should render wrapped description lines with indentation:\n%s", got)
	}
}

func TestHandleSkillsCommand_OverviewShowsDiagnosticsWhenNoSkillsParse(t *testing.T) {
	disableColors(t)
	agent, out := newSurfaceDispatchTestAgent(t)

	brokenDir := filepath.Join(".agents", "skills", "broken")
	if err := os.MkdirAll(brokenDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	broken := "---\nname: broken\n---\n# missing description\n"
	if err := os.WriteFile(filepath.Join(brokenDir, "SKILL.md"), []byte(broken), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if !handleSpecialCommandForSurface("/skills overview", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills overview should be handled")
	}
	got := out.String()
	for _, fragment := range []string{"Agent Skills Overview", "Skills: 1", "XELYON skills", "skill-creator", "Diagnostics: 1 issue(s). Run /skills doctor."} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("/skills overview output missing %q:\n%s", fragment, got)
		}
	}
	if strings.Contains(got, "No skills found.") {
		t.Fatalf("/skills overview should render built-in skills when project skills fail to parse:\n%s", got)
	}
}

func TestHandleSkillsCommand_DoctorShowsDiagnostics(t *testing.T) {
	disableColors(t)
	agent, out := newSurfaceDispatchTestAgent(t)

	brokenDir := filepath.Join(".agents", "skills", "broken")
	if err := os.MkdirAll(brokenDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	broken := "---\nname: broken\n---\n# missing description\n"
	if err := os.WriteFile(filepath.Join(brokenDir, "SKILL.md"), []byte(broken), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if !handleSpecialCommandForSurface("/skills doctor", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills doctor should be handled")
	}
	if !strings.Contains(out.String(), "parse_skill_failed") {
		t.Fatalf("/skills doctor output should include parse diagnostics:\n%s", out.String())
	}
}

func TestHandleSkillsCommand_DoctorRoutingShowsMissingSidecar(t *testing.T) {
	disableColors(t)
	agent, out := newSurfaceDispatchTestAgent(t)

	skillDir := filepath.Join(".agents", "skills", "legacy")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	content := "---\nname: legacy\ndescription: Use this workflow for a project repeated operation.\n---\n# Legacy\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if !handleSpecialCommandForSurface("/skills doctor --routing", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills doctor --routing should be handled")
	}
	if !strings.Contains(out.String(), "missing_xelyon_metadata") {
		t.Fatalf("/skills doctor --routing output should include missing sidecar info:\n%s", out.String())
	}
}

func TestHandleSkillsCommand_SuggestShowsFullRankedList(t *testing.T) {
	disableColors(t)
	agent, out := newSurfaceDispatchTestAgent(t)

	skillDir := filepath.Join(".agents", "skills", "review")
	if err := os.MkdirAll(filepath.Join(skillDir, "agents"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	content := "---\nname: strict-review\ndescription: Review diffs and report actionable findings.\n---\n# Review\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	sidecar := "version: 1\nintents:\n  - code-review\nrole: primary\nread_only: true\nmodes:\n  - review\ntriggers:\n  - review\nconflicts:\n  - implementation\nactivation: hint\n"
	if err := os.WriteFile(filepath.Join(skillDir, "agents", "xelyon.yaml"), []byte(sidecar), 0o644); err != nil {
		t.Fatalf("WriteFile(sidecar) error = %v", err)
	}

	if !handleSpecialCommandForSurface("/skills suggest review this diff", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills suggest should be handled")
	}
	got := out.String()
	for _, fragment := range []string{"Skill Routing Suggestion", "Ranked skills:", "strict-review", "Primary:"} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("/skills suggest output missing %q:\n%s", fragment, got)
		}
	}
}

func TestHandleSkillsCommand_SuggestDoesNotScoreCommandName(t *testing.T) {
	disableColors(t)
	agent, out := newSurfaceDispatchTestAgent(t)

	if !handleSpecialCommandForSurface("/skills suggest review provider runtime config", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills suggest should be handled")
	}
	got := out.String()
	if !strings.Contains(got, "Skill Routing Suggestion") {
		t.Fatalf("/skills suggest output missing report header:\n%s", got)
	}
	if strings.Contains(got, "Primary:\n- skill-creator") {
		t.Fatalf("/skills suggest should not promote skill-creator from the command name itself:\n%s", got)
	}
	if !strings.Contains(got, "skill-creator (0, none, hint)") {
		t.Fatalf("skill-creator should remain unranked for non-skill task text:\n%s", got)
	}
}

func TestHandleSkillsCommand_UsageAndClear(t *testing.T) {
	disableColors(t)
	agent, out := newSurfaceDispatchTestAgent(t)
	repo, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	initSkillRoutingGitRepo(t, repo)

	store, ok := agent.skillUsageLedgerStoreForProject(true)
	if !ok {
		t.Fatal("skillUsageLedgerStoreForProject() ok = false, want true")
	}
	if err := store.Append(usageledger.Record{
		Type:        "recommendation",
		Recommended: []usageledger.SkillSummary{{Name: "demo", Category: "primary", Score: 90, Confidence: "high", Activation: "hint"}},
	}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	if !handleSpecialCommandForSurface("/skills usage", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills usage should be handled")
	}
	if !strings.Contains(out.String(), "Skill Routing Usage") || !strings.Contains(out.String(), "demo") {
		t.Fatalf("/skills usage output missing summary:\n%s", out.String())
	}

	out.Reset()
	if !handleSpecialCommandForSurface("/skills usage clear", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills usage clear should be handled")
	}
	if !strings.Contains(out.String(), "Cleared current repo") {
		t.Fatalf("/skills usage clear output missing confirmation:\n%s", out.String())
	}
	summary, err := store.Summary()
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.Records != 0 {
		t.Fatalf("summary.Records = %d, want 0 after clear", summary.Records)
	}
}

func TestHandleSkillsCommand_UsageClearRejectsUnknownArgs(t *testing.T) {
	disableColors(t)
	agent, out := newSurfaceDispatchTestAgent(t)
	repo, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	initSkillRoutingGitRepo(t, repo)

	store, ok := agent.skillUsageLedgerStoreForProject(true)
	if !ok {
		t.Fatal("skillUsageLedgerStoreForProject() ok = false, want true")
	}
	if err := store.Append(usageledger.Record{
		Type:        "recommendation",
		Recommended: []usageledger.SkillSummary{{Name: "demo", Category: "primary", Score: 90, Confidence: "high", Activation: "hint"}},
	}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	for _, command := range []string{
		"/skills usage clear --dry-run",
		"/skills usage clear --all typo",
		"/skills usage clear --all --all",
	} {
		out.Reset()
		if !handleSpecialCommandForSurface(command, agent, commandcatalog.CommandSurfaceClassic) {
			t.Fatalf("%s should be handled", command)
		}
		if !strings.Contains(out.String(), skillsUsageCommandUsage) {
			t.Fatalf("%s output should reject unknown args:\n%s", command, out.String())
		}
		summary, err := store.Summary()
		if err != nil {
			t.Fatalf("Summary() after %s error = %v", command, err)
		}
		if summary.Records != 1 {
			t.Fatalf("summary.Records after %s = %d, want 1", command, summary.Records)
		}
	}
}

func TestHandleSkillsCommand_UsageSkipsNoRepoLedgerWithoutProjectRoot(t *testing.T) {
	disableColors(t)
	agent, out := newSurfaceDispatchTestAgent(t)

	noRepoStore := usageledger.NewStore(usageledger.Options{
		StateHome: filepath.Join(os.Getenv("HOME"), ".xelyon"),
		Enabled:   true,
	})
	if err := noRepoStore.Append(usageledger.Record{
		Type:        "recommendation",
		Recommended: []usageledger.SkillSummary{{Name: "stale-no-repo", Category: "primary", Score: 90, Confidence: "high", Activation: "hint"}},
	}); err != nil {
		t.Fatalf("Append(noRepoStore) error = %v", err)
	}

	if !handleSpecialCommandForSurface("/skills usage", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills usage should be handled")
	}
	got := out.String()
	if !strings.Contains(got, skillUsageLedgerRootUnavailableMessage) {
		t.Fatalf("/skills usage output should explain unavailable project root:\n%s", got)
	}
	if strings.Contains(got, "stale-no-repo") {
		t.Fatalf("/skills usage should not read no-repo fallback as current repo:\n%s", got)
	}

	out.Reset()
	if !handleSpecialCommandForSurface("/skills usage clear", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills usage clear should be handled")
	}
	if !strings.Contains(out.String(), skillUsageLedgerRootUnavailableMessage) {
		t.Fatalf("/skills usage clear output should explain unavailable project root:\n%s", out.String())
	}
	summary, err := noRepoStore.Summary()
	if err != nil {
		t.Fatalf("Summary(noRepoStore) error = %v", err)
	}
	if summary.Records != 1 {
		t.Fatalf("no-repo summary.Records = %d, want 1 because current repo clear must not delete it", summary.Records)
	}
}
