package agent

import (
	"strings"
	"testing"
)

func TestGeneratedHelpText_InvestigationSurfaceContract(t *testing.T) {
	if !strings.Contains(GeneratedHelpCommandsText, "Commands:\n") {
		t.Fatal("generated commands help should contain the commands section")
	}
	if !strings.Contains(GeneratedHelpCommandsText, "/help                     - Show this help") {
		t.Fatal("generated commands help should contain /help")
	}
	if strings.Contains(GeneratedHelpCommandsText, "Built-in tools") {
		t.Fatal("generated commands help should not embed tool visibility text")
	}
	if !strings.Contains(GeneratedHelpTipsText, "Tips:\n") {
		t.Fatal("generated tips help should contain the tips section")
	}
	if !strings.Contains(GeneratedHelpTipsText, "Use Ctrl+C to cancel current operation") {
		t.Fatal("generated tips help should contain the expected tip")
	}
}
