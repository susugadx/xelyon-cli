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
	if !strings.Contains(prompt, "Write new_str as the intended replacement based on that verified context") {
		t.Error("legacy mode prompt should allow generated replacement text based on verified context")
	}
	if !strings.Contains(prompt, "edits=[{old_str,new_str},...]") {
		t.Error("legacy mode prompt should recommend same-file str_replace batch edits")
	}
	if !strings.Contains(prompt, "advanced fallback only") {
		t.Error("legacy mode prompt should demote line-range str_replace to fallback guidance")
	}
	for _, forbidden := range []string{
		"no read_file needed",
		"old_str mode requires read_file first",
		"Prefer str_replace for partial edits after targeted reads or searches.",
		"old_str/new_str copied from actual",
	} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("legacy mode prompt should not contain old str_replace guidance %q", forbidden)
		}
	}
}

func TestResolveEditToolMode_BedrockClaudeOnly(t *testing.T) {
	t.Setenv("XELYON_EDIT_TOOL", "")

	if got := ResolveEditToolMode("bedrock", "amazon.nova-pro-v1:0"); got != EditToolModeLegacy {
		t.Fatalf("ResolveEditToolMode(bedrock, non-Claude) = %q, want legacy because Bedrock provider is Claude-only in this phase", got)
	}
}
