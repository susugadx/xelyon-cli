package prompt

import (
	"strings"
	"testing"
)

func TestCurrentSystemPrompt_LegacyEditToolMode(t *testing.T) {
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	prompt := CurrentSystemPrompt()
	if !strings.Contains(prompt, "### Legacy edit tools") {
		t.Error("legacy mode prompt should include legacy edit tool guidance")
	}
	if strings.Contains(prompt, "### apply_patch (edit tool)") {
		t.Error("legacy mode prompt should not include apply_patch guide")
	}
	if !strings.Contains(prompt, "actual gather_context, read_file, or search_code output") {
		t.Error("legacy mode prompt should allow gather_context as exact edit provenance")
	}
}
