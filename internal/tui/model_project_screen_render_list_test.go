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

	m.width = 80
	m.height = 10
	m.projectScreen.NormalizeSize(m.width, m.height)
	m = moveProjectToSection(t, m, "rules")
	m = moveProjectToItemPane(t, m)
	m = moveProjectItemSelection(t, m, 10)
	plain := stripANSI(m.projectScreen.View(m.width, m.height))

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

	m.width = 80
	m.height = 10
	m.projectScreen.NormalizeSize(m.width, m.height)
	m = moveProjectToSection(t, m, "rules")
	m = moveProjectToItemPane(t, m)
	plain := stripANSI(m.projectScreen.View(m.width, m.height))
	if !strings.Contains(plain, "first line second line third line") {
		t.Fatalf("rendered list item was not sanitized:\n%s", plain)
	}
}
