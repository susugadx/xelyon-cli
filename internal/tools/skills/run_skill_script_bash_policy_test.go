package skills

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	skillcatalog "github.com/susugadx/xelyon-cli/internal/skills"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestRunSkillScriptTool_Run_DoesNotBypassBashPolicyOrConfirmation(t *testing.T) {
	restore := stubRunSkillScriptDependencies(t)
	defer restore()

	skill := makeScriptSkill(t, "demo")
	scriptPath := filepath.Join(skill.Directory, "scripts", "safe.sh")
	mustWriteRunSkillFile(t, scriptPath, "echo safe\n")
	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{skill}}
	}

	t.Run("confirmation rejection is respected", func(t *testing.T) {
		origSimple := common.SimpleConfirm
		origSimpleWithIO := common.SimpleConfirmWithIO
		common.SimpleConfirm = func(_ string) bool { return false }
		common.SimpleConfirmWithIO = func(_ ui.PromptIO, _ string) bool { return false }
		t.Cleanup(func() {
			common.SimpleConfirm = origSimple
			common.SimpleConfirmWithIO = origSimpleWithIO
		})

		_ = os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
		t.Cleanup(func() {
			_ = os.Unsetenv("XELYON_INTERACTIVE_CONFIRM")
		})

		var out bytes.Buffer
		tool := &RunSkillScriptTool{}
		got, _, _ := tool.Run(tools.ExecutionContext{
			InvocationCWD: skillWorkspaceRoot(skill),
			Stdout:        &out,
			Stderr:        &out,
		}, map[string]string{
			"skill":  "demo",
			"script": "safe.sh",
		})
		if got != "Cancelled by user" {
			t.Fatalf("Run() output = %q, want %q", got, "Cancelled by user")
		}
	})
}
