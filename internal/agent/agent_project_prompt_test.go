package agent

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestExtractProjectMapPathsFromInput_QuotedFilename(t *testing.T) {
	t.Parallel()

	input := "please inspect 'design spec.md' and then compare with \"notes.txt\""

	got := extractProjectMapPathsFromInput(input)
	want := []string{"design spec.md", "notes.txt"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("extractProjectMapPathsFromInput() = %#v, want %#v", got, want)
	}
}

func TestBuildProjectInstructionBlock_IgnoresLegacyXelyonProse(t *testing.T) {
	bundle := &config.ProjectInstructionBundle{
		ProjectConfig: &config.ProjectConfig{
			Context: "LEGACY_BASE_CONTEXT",
			Rules:   []string{"LEGACY_BASE_RULE"},
			Conditional: []config.ProjectConditionalBlock{
				{
					Name:    "Agent",
					Paths:   []string{"internal/agent/**"},
					Context: "LEGACY_CONDITIONAL_CONTEXT",
					Rules:   []string{"LEGACY_CONDITIONAL_RULE"},
				},
			},
		},
		ProjectGuidance: []config.InstructionFile{
			{
				Label:           "AGENTS.md",
				RepositoryScope: ".",
				Strength:        config.InstructionStrengthProjectGuidance,
				Content:         "ROOT_AGENTS_GUIDANCE",
			},
			{
				Label:           "internal/agent/AGENTS.md",
				RepositoryScope: "internal/agent",
				Strength:        config.InstructionStrengthProjectGuidance,
				Content:         "SCOPED_AGENTS_GUIDANCE",
			},
		},
	}

	block := buildProjectInstructionBlock(bundle)

	for _, forbidden := range []string{
		"LEGACY_BASE_CONTEXT",
		"LEGACY_BASE_RULE",
		"LEGACY_CONDITIONAL_CONTEXT",
		"LEGACY_CONDITIONAL_RULE",
		"PROJECT-SPECIFIC RULES (MANDATORY)",
		"critical failure",
		"Legacy xelyon.yaml Context",
	} {
		if strings.Contains(block, forbidden) {
			t.Fatalf("legacy xelyon.yaml prose leaked into project instruction block via %q:\n%s", forbidden, block)
		}
	}
	rootIdx := strings.Index(block, "ROOT_AGENTS_GUIDANCE")
	scopedIdx := strings.Index(block, "SCOPED_AGENTS_GUIDANCE")
	if rootIdx < 0 || scopedIdx < 0 {
		t.Fatalf("AGENTS.md guidance missing from project instruction block:\n%s", block)
	}
	if scopedIdx < rootIdx {
		t.Fatalf("AGENTS.md guidance should preserve root-to-nearest order:\n%s", block)
	}
}

func TestBuildProjectInstructionBlock_LegacyXelyonOnlyReturnsEmpty(t *testing.T) {
	block := buildProjectInstructionBlock(&config.ProjectInstructionBundle{
		ProjectConfig: &config.ProjectConfig{
			Context: "LEGACY_BASE_CONTEXT",
			Rules:   []string{"LEGACY_BASE_RULE"},
		},
	})
	if block != "" {
		t.Fatalf("legacy-only xelyon.yaml prose should not create a prompt block, got:\n%s", block)
	}
}

func TestBuildProjectInstructionBlock_FallbackModeLoadsAgentsWithXelyon(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "xelyon.yaml"), "context: LEGACY_BASE_CONTEXT\nrules:\n  - LEGACY_BASE_RULE\n")
	writeTestFile(t, filepath.Join(root, "AGENTS.md"), "ROOT_AGENTS_GUIDANCE\n")

	cfg := config.DefaultConfig()
	cfg.AgentInstructions.Project.Mode = config.AgentInstructionProjectModeFallback
	cfg.AgentInstructions.Project.IncludeGitignored = true

	bundle := loadProjectInstructionBundleForCWD(cfg, root)
	if bundle == nil {
		t.Fatal("expected project instruction bundle")
	}
	block := buildProjectInstructionBlock(bundle)

	if !strings.Contains(block, "ROOT_AGENTS_GUIDANCE") {
		t.Fatalf("fallback mode with xelyon.yaml should keep AGENTS.md guidance:\n%s", block)
	}
	if !strings.Contains(block, "project.mode=fallback is deprecated") {
		t.Fatalf("fallback mode should be explicit in prompt load notes:\n%s", block)
	}
	for _, forbidden := range []string{"LEGACY_BASE_CONTEXT", "LEGACY_BASE_RULE"} {
		if strings.Contains(block, forbidden) {
			t.Fatalf("fallback mode should not inject legacy xelyon prose %q:\n%s", forbidden, block)
		}
	}
}
