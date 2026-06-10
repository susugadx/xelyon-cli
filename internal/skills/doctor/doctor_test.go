package doctor

import (
	"strings"
	"testing"

	skillcatalog "github.com/susugadx/xelyon-cli/internal/skills"
)

func TestBuildReportPlainDoesNotWarnMissingSidecar(t *testing.T) {
	catalog := skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{
		{Name: "legacy", Description: "Use this workflow for project-specific repeated operations.", Source: skillcatalog.SourceProject, SkillPath: "/repo/.agents/skills/legacy/SKILL.md"},
	}}

	report := BuildReport(catalog, Options{})
	if len(report.Diagnostics) != 0 {
		t.Fatalf("plain report diagnostics = %#v, want none", report.Diagnostics)
	}
}

func TestBuildReportRoutingReportsMissingSidecarAsInfo(t *testing.T) {
	catalog := skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{
		{Name: "legacy", Description: "Use this workflow for project-specific repeated operations.", Source: skillcatalog.SourceProject, SkillPath: "/repo/.agents/skills/legacy/SKILL.md"},
	}}

	report := BuildReport(catalog, Options{Routing: true})
	diag := findDiag(report.Diagnostics, "missing_xelyon_metadata")
	if diag == nil {
		t.Fatalf("diagnostics = %#v, want missing_xelyon_metadata", report.Diagnostics)
	}
	if diag.Severity != skillcatalog.SeverityInfo {
		t.Fatalf("severity = %s, want info", diag.Severity)
	}
}

func TestBuildReportRoutingReportsLegacySkillCreatorShadowing(t *testing.T) {
	catalog := skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{
		{Name: "skill-creator", Description: "Use this workflow to create skills.", Source: skillcatalog.SourceProject, SkillPath: "/repo/.agents/skills/skill-creator/SKILL.md"},
	}}

	report := BuildReport(catalog, Options{Routing: true})
	for _, code := range []string{"missing_xelyon_metadata", "source_shadowing"} {
		if findDiag(report.Diagnostics, code) == nil {
			t.Fatalf("diagnostics = %#v, want %s", report.Diagnostics, code)
		}
	}
}

func TestBuildReportRoutingReportsReadOnlyMissingConflictsAndOverlap(t *testing.T) {
	catalog := skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{
		doctorSkill("review-a", skillcatalog.RoutingRolePrimary, true, []string{"code-review"}, nil),
		doctorSkill("review-b", skillcatalog.RoutingRolePrimary, false, []string{"code-review"}, nil),
	}}

	report := BuildReport(catalog, Options{Routing: true})
	for _, code := range []string{"read_only_without_conflicts", "overlapping_primary_candidates"} {
		if findDiag(report.Diagnostics, code) == nil {
			t.Fatalf("diagnostics = %#v, want %s", report.Diagnostics, code)
		}
	}
}

func TestBuildReportRoutingReportsPromptBudgetPressure(t *testing.T) {
	catalog := skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{
		doctorSkill("one", skillcatalog.RoutingRolePrimary, false, []string{"implementation"}, nil),
		doctorSkill("two", skillcatalog.RoutingRoleSupporting, false, []string{"config"}, nil),
	}}

	report := BuildReport(catalog, Options{Routing: true, PromptCatalogMaxItems: 1})
	if findDiag(report.Diagnostics, "prompt_budget_pressure") == nil {
		t.Fatalf("diagnostics = %#v, want prompt_budget_pressure", report.Diagnostics)
	}
}

func TestBuildReportRoutingReportsDuplicateTriggersAndConflictCycle(t *testing.T) {
	catalog := skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{
		doctorSkillWithRouting("review-a", &skillcatalog.RoutingMetadata{
			Version:    skillcatalog.XelyonRoutingMetadataVersion,
			Intents:    []string{"code-review"},
			Role:       skillcatalog.RoutingRolePrimary,
			Modes:      []string{"review"},
			Triggers:   []string{"diff review"},
			Conflicts:  []string{"review"},
			Activation: skillcatalog.RoutingActivationHint,
		}),
		doctorSkillWithRouting("review-b", &skillcatalog.RoutingMetadata{
			Version:    skillcatalog.XelyonRoutingMetadataVersion,
			Intents:    []string{"risk-scan"},
			Role:       skillcatalog.RoutingRolePrimary,
			Modes:      []string{"investigation"},
			Triggers:   []string{"diff review"},
			Activation: skillcatalog.RoutingActivationHint,
		}),
	}}

	report := BuildReport(catalog, Options{Routing: true})
	for _, code := range []string{"description_duplicates_trigger", "conflict_cycle"} {
		if findDiag(report.Diagnostics, code) == nil {
			t.Fatalf("diagnostics = %#v, want %s", report.Diagnostics, code)
		}
	}
}

func TestFormatReportIncludesSeveritySummary(t *testing.T) {
	report := Report{
		SkillCount: 1,
		Diagnostics: []skillcatalog.Diagnostic{{
			Severity: skillcatalog.SeverityWarning,
			Code:     "read_only_without_conflicts",
			Message:  "warning",
		}},
	}

	got := FormatReport(report)
	if !strings.Contains(got, "Skills Doctor") || !strings.Contains(got, "- 1 warnings") || !strings.Contains(got, "[WARNING] read_only_without_conflicts") {
		t.Fatalf("FormatReport() missing expected content:\n%s", got)
	}
}

func doctorSkill(name string, role skillcatalog.RoutingRole, readOnly bool, intents, conflicts []string) skillcatalog.ParsedSkill {
	return doctorSkillWithRouting(name, &skillcatalog.RoutingMetadata{
		Version:    skillcatalog.XelyonRoutingMetadataVersion,
		Intents:    intents,
		Role:       role,
		ReadOnly:   readOnly,
		Conflicts:  conflicts,
		Activation: skillcatalog.RoutingActivationHint,
	})
}

func doctorSkillWithRouting(name string, routing *skillcatalog.RoutingMetadata) skillcatalog.ParsedSkill {
	return skillcatalog.ParsedSkill{
		Name:        name,
		Description: "Use this skill for focused routing diagnostics in tests.",
		Source:      skillcatalog.SourceProject,
		SkillPath:   "/repo/.agents/skills/" + name + "/SKILL.md",
		Routing:     routing,
	}
}

func findDiag(diagnostics []skillcatalog.Diagnostic, code string) *skillcatalog.Diagnostic {
	for i := range diagnostics {
		if diagnostics[i].Code == code {
			return &diagnostics[i]
		}
	}
	return nil
}
