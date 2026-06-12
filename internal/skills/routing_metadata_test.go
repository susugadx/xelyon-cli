package skills

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSKILLWithDiagnostics_LoadsValidXelyonMetadata(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".agents", "skills", "review")
	mustWriteSkill(t, skillDir, validSkill("strict-review", "review diffs", "# Review"))
	mustWriteFile(t, filepath.Join(skillDir, "agents", "xelyon.yaml"), `version: 1
intents:
  - code-review
role: primary
read_only: true
modes:
  - review
triggers:
  - review
  - レビュー
conflicts:
  - implementation
  - file-edit
activation: hint
`)

	parsed, diagnostics, err := ParseSKILLWithDiagnostics(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("ParseSKILLWithDiagnostics() error = %v", err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
	if parsed.Routing == nil {
		t.Fatal("Routing = nil, want metadata")
	}
	if parsed.Routing.Role != RoutingRolePrimary {
		t.Fatalf("Role = %q, want primary", parsed.Routing.Role)
	}
	if !parsed.Routing.ReadOnly {
		t.Fatal("ReadOnly = false, want true")
	}
	assertStringSliceEqual(t, parsed.Routing.Intents, []string{"code-review"})
	assertStringSliceEqual(t, parsed.Routing.Modes, []string{"review"})
	assertStringSliceEqual(t, parsed.Routing.Conflicts, []string{"implementation", "file-edit"})
}

func TestParseSKILLWithDiagnostics_MissingSidecarKeepsLegacySkillValid(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".agents", "skills", "legacy")
	mustWriteSkill(t, skillDir, validSkill("legacy", "legacy desc", "# Legacy"))

	parsed, diagnostics, err := ParseSKILLWithDiagnostics(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("ParseSKILLWithDiagnostics() error = %v", err)
	}
	if parsed.Routing != nil {
		t.Fatalf("Routing = %#v, want nil", parsed.Routing)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
}

func TestCatalog_InvalidXelyonMetadataWarnsAndFallsBackToDescription(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".agents", "skills", "invalid")
	mustWriteSkill(t, skillDir, validSkill("invalid-sidecar", "desc fallback", "# Body"))
	mustWriteFile(t, filepath.Join(skillDir, "agents", "xelyon.yaml"), `version: 2
intents:
  - code-review
`)

	catalog := Catalog(DiscoverResult{Skills: []DiscoveredSkill{{
		Directory: skillDir,
		SkillPath: filepath.Join(skillDir, "SKILL.md"),
		Source:    SourceProject,
		RootOrder: 0,
		PathOrder: skillDir,
	}}})
	skill, ok := findParsedSkill(catalog.Skills, "invalid-sidecar")
	if !ok {
		t.Fatalf("Catalog() missing skill: %#v", catalog.Skills)
	}
	if skill.Routing != nil {
		t.Fatalf("Routing = %#v, want nil fallback", skill.Routing)
	}
	if !hasDiagnosticCode(catalog.Diagnostics, "invalid_xelyon_metadata") {
		t.Fatalf("diagnostics = %#v, want invalid_xelyon_metadata", catalog.Diagnostics)
	}
}

func TestCatalog_XelyonMetadataUnknownFieldsAndValuesWarnWithoutInvalidating(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".agents", "skills", "unknowns")
	mustWriteSkill(t, skillDir, validSkill("unknowns", "desc", "# Body"))
	mustWriteFile(t, filepath.Join(skillDir, "agents", "xelyon.yaml"), `version: 1
future_field: true
intents:
  - code-review
  - future-intent
role: future-role
modes:
  - review
  - future-mode
conflicts:
  - implementation
  - future-conflict
activation: future
`)

	catalog := Catalog(DiscoverResult{Skills: []DiscoveredSkill{{
		Directory: skillDir,
		SkillPath: filepath.Join(skillDir, "SKILL.md"),
		Source:    SourceProject,
		RootOrder: 0,
		PathOrder: skillDir,
	}}})
	skill, ok := findParsedSkill(catalog.Skills, "unknowns")
	if !ok || skill.Routing == nil {
		t.Fatalf("Catalog() skill = %#v, want valid metadata with known fields", skill)
	}
	assertStringSliceEqual(t, skill.Routing.Intents, []string{"code-review"})
	assertStringSliceEqual(t, skill.Routing.Modes, []string{"review"})
	assertStringSliceEqual(t, skill.Routing.Conflicts, []string{"implementation"})
	if skill.Routing.Role != "" {
		t.Fatalf("Role = %q, want unknown role ignored", skill.Routing.Role)
	}
	for _, code := range []string{
		"unknown_xelyon_metadata_field",
		"unknown_intent",
		"unknown_mode",
		"unknown_role",
		"unknown_conflict",
		"unknown_activation",
	} {
		if !hasDiagnosticCode(catalog.Diagnostics, code) {
			t.Fatalf("diagnostics = %#v, want %s", catalog.Diagnostics, code)
		}
	}
}

func TestCatalog_IgnoresOtherAgentSidecars(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".agents", "skills", "portable")
	mustWriteSkill(t, skillDir, validSkill("portable", "desc", "# Body"))
	mustWriteFile(t, filepath.Join(skillDir, "agents", "openai.yaml"), `version: 999
role: primary
`)

	catalog := Catalog(DiscoverResult{Skills: []DiscoveredSkill{{
		Directory: skillDir,
		SkillPath: filepath.Join(skillDir, "SKILL.md"),
		Source:    SourceProject,
		RootOrder: 0,
		PathOrder: skillDir,
	}}})
	skill, ok := findParsedSkill(catalog.Skills, "portable")
	if !ok {
		t.Fatalf("Catalog() missing skill: %#v", catalog.Skills)
	}
	if skill.Routing != nil {
		t.Fatalf("Routing = %#v, want nil", skill.Routing)
	}
	if len(catalog.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", catalog.Diagnostics)
	}
}

func TestBuiltinSkillCreatorHasRoutingMetadataAndSidecarGuidance(t *testing.T) {
	catalog := Catalog(DiscoverResult{})
	skill, ok := findParsedSkill(catalog.Skills, xelyonBuiltinSkillCreatorName)
	if !ok {
		t.Fatalf("missing built-in %s", xelyonBuiltinSkillCreatorName)
	}
	if skill.Routing == nil || skill.Routing.Role != RoutingRoleAuthoring {
		t.Fatalf("built-in routing = %#v, want authoring metadata", skill.Routing)
	}
	if !containsText(skill.Body, "agents/xelyon.yaml") {
		t.Fatalf("built-in guidance should mention agents/xelyon.yaml:\n%s", skill.Body)
	}
	if !containsText(skill.Body, "Do not add XELYON-specific fields to SKILL.md frontmatter") {
		t.Fatalf("built-in guidance should forbid XELYON-specific frontmatter:\n%s", skill.Body)
	}
}

func hasDiagnosticCode(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func containsText(value, fragment string) bool {
	return strings.Contains(value, fragment)
}
