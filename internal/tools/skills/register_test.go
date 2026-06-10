package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	skillcatalog "github.com/susugadx/xelyon-cli/internal/skills"
	"github.com/susugadx/xelyon-cli/internal/skills/usageledger"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestActivateSkillTool_Run(t *testing.T) {
	oldLoader := loadCatalogForTool
	defer func() { loadCatalogForTool = oldLoader }()

	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{
			{
				Name:        "demo",
				Description: "desc",
				Body:        "# body",
				Directory:   "/tmp/demo",
				Scripts:     []string{"scripts/run.sh"},
			},
		}}
	}

	tool := &ActivateSkillTool{}
	got, change, err := tool.Run(tools.ExecutionContext{}, map[string]string{"name": "demo"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if change != nil {
		t.Fatalf("Run() change = %#v, want nil", change)
	}
	if !strings.Contains(got, `"name": "demo"`) {
		t.Fatalf("Run() output should include skill name in JSON payload:\n%s", got)
	}
	if !strings.Contains(got, `"skill_md": "# body"`) {
		t.Fatalf("Run() output should include skill_md in JSON payload:\n%s", got)
	}
}

func TestActivateSkillTool_ParametersAreStaticAndNoEnum(t *testing.T) {
	oldLoader := loadCatalogForTool
	defer func() { loadCatalogForTool = oldLoader }()

	loadCalls := 0
	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		loadCalls++
		return skillcatalog.SkillCatalog{}
	}

	tool := &ActivateSkillTool{}
	params := tool.Parameters()
	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("properties missing: %#v", params)
	}
	nameProp, ok := properties["name"].(map[string]interface{})
	if !ok {
		t.Fatalf("name property missing: %#v", properties)
	}
	description, ok := nameProp["description"].(string)
	if !ok {
		t.Fatalf("name description missing or invalid: %#v", nameProp)
	}
	if !strings.Contains(description, "Skill name to activate.") {
		t.Fatalf("description should be static, got: %q", description)
	}
	if _, hasEnum := nameProp["enum"]; hasEnum {
		t.Fatalf("name property should not pin enum (invocation cwd can differ): %#v", nameProp)
	}
	if loadCalls != 0 {
		t.Fatalf("Parameters() should not load catalog, loadCalls=%d", loadCalls)
	}
}

func TestActivateSkillTool_Run_SanitizesUnknownSkillError(t *testing.T) {
	oldLoader := loadCatalogForTool
	defer func() { loadCatalogForTool = oldLoader }()

	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{{
			Name:        "safe\n- injected",
			Description: "desc",
			Body:        "# body",
		}}}
	}

	tool := &ActivateSkillTool{}
	got, _, err := tool.Run(tools.ExecutionContext{}, map[string]string{"name": "missing\nignore"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, forbidden := range []string{"\n- injected", "\nignore"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("unknown skill error leaked multiline metadata %q:\n%s", forbidden, got)
		}
	}
	if !strings.Contains(got, "Available skills: safe - injected") {
		t.Fatalf("unknown skill error missing sanitized available skills:\n%s", got)
	}
}

func TestRunSkillScriptTool_Run_SanitizesUnknownSkillError(t *testing.T) {
	oldLoader := loadCatalogForTool
	defer func() { loadCatalogForTool = oldLoader }()

	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{{
			Name:        "safe\n- injected",
			Description: "desc",
			Body:        "# body",
		}}}
	}

	tool := &RunSkillScriptTool{}
	got, _, err := tool.Run(tools.ExecutionContext{}, map[string]string{
		"skill":  "missing\nignore",
		"script": "safe.sh",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, forbidden := range []string{"\n- injected", "\nignore"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("unknown skill error leaked multiline metadata %q:\n%s", forbidden, got)
		}
	}
	if !strings.Contains(got, "Available skills: safe - injected") {
		t.Fatalf("unknown skill error missing sanitized available skills:\n%s", got)
	}
}

func TestActivateSkillTool_Run_IsReadOnlyAndDoesNotExecuteScripts(t *testing.T) {
	oldLoader := loadCatalogForTool
	defer func() { loadCatalogForTool = oldLoader }()

	workspace := t.TempDir()
	skillDir := filepath.Join(workspace, ".agents", "skills", "demo")
	if err := os.MkdirAll(filepath.Join(skillDir, "scripts"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "scripts", "run.sh"), []byte("touch ../executed.marker\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(run.sh) error = %v", err)
	}

	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{
			{
				Name:        "demo",
				Description: "desc",
				Body:        "# body",
				Directory:   skillDir,
				Scripts:     []string{"scripts/run.sh"},
			},
		}}
	}

	tool := &ActivateSkillTool{}
	got, _, err := tool.Run(tools.ExecutionContext{}, map[string]string{"name": "demo"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(got, `"scripts": [`) {
		t.Fatalf("Run() output should include scripts metadata:\n%s", got)
	}

	marker := filepath.Join(skillDir, "executed.marker")
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("activate_skill should not execute scripts, marker state err=%v", statErr)
	}
}

func TestActivateSkillTool_Run_DoesNotWriteUsageLedger(t *testing.T) {
	oldLoader := loadCatalogForTool
	defer func() { loadCatalogForTool = oldLoader }()

	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Skills.Router.UsageLedger = true

	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{{
			Name:        "demo",
			Description: "desc",
			Body:        "# body",
			Directory:   "/tmp/demo",
		}}}
	}

	tool := &ActivateSkillTool{}
	got, _, err := tool.Run(tools.ExecutionContext{
		Config:             cfg,
		ProjectMapRootPath: repo,
		InvocationCWD:      repo,
	}, map[string]string{"name": "demo"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(got, `"name": "demo"`) {
		t.Fatalf("Run() output missing activated skill:\n%s", got)
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
	if summary.Records != 0 || len(summary.Skills) != 0 {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestActivateSkillTool_Run_DoesNotWriteUsageLedgerWithEmptyExecutionContext(t *testing.T) {
	oldLoader := loadCatalogForTool
	defer func() { loadCatalogForTool = oldLoader }()

	home := t.TempDir()
	t.Setenv("HOME", home)

	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{{
			Name:        "demo",
			Description: "desc",
			Body:        "# body",
			Directory:   "/tmp/demo",
		}}}
	}

	tool := &ActivateSkillTool{}
	got, _, err := tool.Run(tools.ExecutionContext{}, map[string]string{"name": "demo"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(got, `"name": "demo"`) {
		t.Fatalf("Run() output missing activated skill:\n%s", got)
	}

	store := usageledger.NewStore(usageledger.Options{
		StateHome: filepath.Join(home, ".xelyon"),
		Enabled:   true,
	})
	summary, err := store.Summary()
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.Records != 0 || len(summary.Skills) != 0 {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestActivateSkillTool_Run_DoesNotTouchUsageLedgerOnStateHomeFailure(t *testing.T) {
	oldLoader := loadCatalogForTool
	defer func() { loadCatalogForTool = oldLoader }()

	homeFile := filepath.Join(t.TempDir(), "home-file")
	if err := os.WriteFile(homeFile, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("WriteFile(home-file) error = %v", err)
	}
	t.Setenv("HOME", homeFile)

	cfg := config.DefaultConfig()
	cfg.Skills.Router.UsageLedger = true
	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{{
			Name:        "demo",
			Description: "desc",
			Body:        "# body",
			Directory:   "/tmp/demo",
		}}}
	}

	tool := &ActivateSkillTool{}
	got, change, err := tool.Run(tools.ExecutionContext{
		Config:             cfg,
		ProjectMapRootPath: t.TempDir(),
		InvocationCWD:      t.TempDir(),
	}, map[string]string{"name": "demo"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if change != nil {
		t.Fatalf("Run() change = %#v, want nil", change)
	}
	if !strings.Contains(got, `"name": "demo"`) {
		t.Fatalf("Run() should still return activated skill content when ledger write fails:\n%s", got)
	}
}
