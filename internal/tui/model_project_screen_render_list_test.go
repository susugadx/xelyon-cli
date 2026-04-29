package tui

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestProjectScreen_LongListRenderingKeepsSelectionVisible(t *testing.T) {
	rules := make([]string, 12)
	for i := range rules {
		rules[i] = "rule-" + string(rune('A'+i))
	}
	agent := &stubAgent{
		statusLine: "ready",
		projectConfig: &config.ProjectConfig{
			Context: "ctx",
			Rules:   rules,
		},
	}
	m := newProjectTestModel(agent)

	m.projectScreen.sectionIndex = int(projectSectionRules)
	m.projectScreen.activePane = projectPaneItem
	m.projectScreen.itemIndex[projectSectionRules] = 10
	lines := m.renderProjectListLines(40, 5)
	plain := stripANSI(strings.Join(lines, "\n"))

	if !strings.Contains(plain, "rule-K") {
		t.Fatalf("rendered list should include selected item rule-K:\n%s", plain)
	}
	if strings.Contains(plain, "rule-A") {
		t.Fatalf("rendered list should scroll past first item when selection is low:\n%s", plain)
	}
}

func TestProjectScreen_ListRenderingSanitizesMultilineItems(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		projectConfig: &config.ProjectConfig{
			Context: "ctx",
			Rules: []string{
				"first line\nsecond line\tthird line",
			},
		},
	}
	m := newProjectTestModel(agent)

	m.projectScreen.sectionIndex = int(projectSectionRules)
	m.projectScreen.activePane = projectPaneItem
	lines := m.renderProjectListLines(48, 4)
	if len(lines) == 0 {
		t.Fatal("renderProjectListLines returned no lines")
	}
	for i, line := range lines {
		if strings.Contains(line, "\n") {
			t.Fatalf("rendered list line %d contains embedded newline: %q", i, line)
		}
	}

	plain := stripANSI(lines[0])
	if !strings.Contains(plain, "first line second line third line") {
		t.Fatalf("rendered list item was not sanitized:\n%s", plain)
	}
}
