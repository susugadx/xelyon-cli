package doctor

import (
	"fmt"
	"sort"
	"strings"

	skillcatalog "github.com/susugadx/xelyon-cli/internal/skills"
)

const defaultPromptCatalogMaxEntries = skillcatalog.DefaultPromptCatalogMaxEntries

// Options は Doctor report の診断範囲を指定する。
type Options struct {
	Routing               bool
	PromptCatalogMaxItems int
	AdditionalDiagnostics []skillcatalog.Diagnostic
}

// Report は Doctor が表示する診断結果。
type Report struct {
	SkillCount  int
	Diagnostics []skillcatalog.Diagnostic
	Routing     bool
}

// BuildReport は catalog と routing metadata の deterministic diagnostics を構築する。
func BuildReport(catalog skillcatalog.SkillCatalog, opts Options) Report {
	diagnostics := append([]skillcatalog.Diagnostic(nil), catalog.Diagnostics...)
	if opts.Routing {
		diagnostics = append(diagnostics, buildRoutingDiagnostics(catalog, opts)...)
	}
	diagnostics = append(diagnostics, opts.AdditionalDiagnostics...)
	return Report{
		SkillCount:  len(catalog.Skills),
		Diagnostics: diagnostics,
		Routing:     opts.Routing,
	}
}

// FormatReport は classic REPL/TUI 向けの human-readable Doctor report を返す。
func FormatReport(report Report) string {
	var b strings.Builder
	b.WriteString("Skills Doctor\n\n")
	fmt.Fprintf(&b, "Catalog:\n- %d skills\n", report.SkillCount)
	writeSeveritySummary(&b, report.Diagnostics)
	if len(report.Diagnostics) == 0 {
		b.WriteString("\nNo diagnostics. Skills catalog is healthy.\n")
		return b.String()
	}
	b.WriteString("\nDiagnostics:\n")
	for _, diag := range report.Diagnostics {
		fmt.Fprintf(&b, "- [%s] %s", strings.ToUpper(string(diag.Severity)), diag.Code)
		if path := sanitizeDoctorLine(diag.Path, ""); path != "" {
			fmt.Fprintf(&b, " (%s)", path)
		}
		fmt.Fprintf(&b, ": %s\n", sanitizeDoctorLine(diag.Message, "diagnostic unavailable"))
	}
	return b.String()
}

func buildRoutingDiagnostics(catalog skillcatalog.SkillCatalog, opts Options) []skillcatalog.Diagnostic {
	var diagnostics []skillcatalog.Diagnostic
	promptCap := opts.PromptCatalogMaxItems
	if promptCap <= 0 {
		promptCap = defaultPromptCatalogMaxEntries
	}
	if len(catalog.Skills) > promptCap {
		diagnostics = append(diagnostics, diag(skillcatalog.SeverityWarning, "prompt_budget_pressure", "", fmt.Sprintf("catalog has %d skills; prompt metadata cap is %d", len(catalog.Skills), promptCap)))
	}

	primaryByIntent := map[string][]skillcatalog.ParsedSkill{}
	supportingByIntent := map[string][]skillcatalog.ParsedSkill{}
	triggerOwners := map[string][]skillcatalog.ParsedSkill{}
	for _, skill := range catalog.Skills {
		if skill.Name == "skill-creator" && skill.Source != skillcatalog.SourceXelyon {
			diagnostics = append(diagnostics, diag(skillcatalog.SeverityInfo, "source_shadowing", skill.SkillPath, "project/home skill shadows XELYON built-in skill-creator by name"))
		}
		metadata := skill.Routing
		if metadata == nil {
			if skill.Source != skillcatalog.SourceXelyon {
				diagnostics = append(diagnostics, diag(skillcatalog.SeverityInfo, "missing_xelyon_metadata", skill.SkillPath, "skill has no agents/xelyon.yaml; description-only routing will be used"))
			}
			diagnostics = append(diagnostics, descriptionDiagnostics(skill)...)
			continue
		}
		if metadata.ReadOnly && len(metadata.Conflicts) == 0 {
			diagnostics = append(diagnostics, diag(skillcatalog.SeverityWarning, "read_only_without_conflicts", skill.SkillPath, "read-only skill should declare conflicts such as implementation or file-edit"))
		}
		if metadata.Activation == skillcatalog.RoutingActivationAuto && len(skill.Body) > 6000 {
			diagnostics = append(diagnostics, diag(skillcatalog.SeverityWarning, "auto_activation_without_budget_guard", skill.SkillPath, "auto activation skill body is large; keep v1 runtime hint-only or reduce body size"))
		}
		for _, trigger := range metadata.Triggers {
			if isBroadTrigger(trigger) {
				diagnostics = append(diagnostics, diag(skillcatalog.SeverityWarning, "trigger_too_broad", skill.SkillPath, fmt.Sprintf("trigger %q is too broad for deterministic routing", trigger)))
			}
			if normalized := normalizedTriggerKey(trigger); normalized != "" {
				triggerOwners[normalized] = append(triggerOwners[normalized], skill)
			}
		}
		if conflicts := selfConflicts(metadata); len(conflicts) > 0 {
			diagnostics = append(diagnostics, diag(skillcatalog.SeverityWarning, "conflict_cycle", skill.SkillPath, fmt.Sprintf("skill conflicts with its own routing metadata: %s", strings.Join(conflicts, ", "))))
		}
		diagnostics = append(diagnostics, descriptionDiagnostics(skill)...)
		for _, intent := range metadata.Intents {
			switch metadata.Role {
			case skillcatalog.RoutingRolePrimary, "":
				primaryByIntent[intent] = append(primaryByIntent[intent], skill)
			case skillcatalog.RoutingRoleSupporting, skillcatalog.RoutingRoleGuardrail:
				supportingByIntent[intent] = append(supportingByIntent[intent], skill)
			}
		}
	}

	for _, intent := range sortedDiagnosticKeys(primaryByIntent) {
		skills := primaryByIntent[intent]
		if len(skills) <= 1 {
			continue
		}
		diagnostics = append(diagnostics, diag(
			skillcatalog.SeverityWarning,
			"overlapping_primary_candidates",
			"",
			fmt.Sprintf("intent %q has multiple primary candidates: %s", intent, skillNames(skills)),
		))
	}
	for _, trigger := range sortedDiagnosticKeys(triggerOwners) {
		skills := triggerOwners[trigger]
		if len(skills) <= 1 {
			continue
		}
		diagnostics = append(diagnostics, diag(
			skillcatalog.SeverityWarning,
			"description_duplicates_trigger",
			"",
			fmt.Sprintf("trigger %q is shared by multiple skills: %s", trigger, skillNames(skills)),
		))
	}
	for _, intent := range sortedDiagnosticKeys(supportingByIntent) {
		skills := supportingByIntent[intent]
		if len(primaryByIntent[intent]) == 0 {
			diagnostics = append(diagnostics, diag(
				skillcatalog.SeverityWarning,
				"missing_primary_for_intent",
				"",
				fmt.Sprintf("intent %q has supporting skills but no primary candidate: %s", intent, skillNames(skills)),
			))
		}
	}
	return diagnostics
}

func normalizedTriggerKey(trigger string) string {
	normalized := strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(trigger))), " ")
	if len([]rune(normalized)) < 4 {
		return ""
	}
	return normalized
}

func selfConflicts(metadata *skillcatalog.RoutingMetadata) []string {
	if metadata == nil {
		return nil
	}
	conflicts := map[string]struct{}{}
	for _, conflict := range metadata.Conflicts {
		conflict = strings.ToLower(strings.TrimSpace(conflict))
		if conflict != "" {
			conflicts[conflict] = struct{}{}
		}
	}
	var self []string
	for _, value := range metadata.Modes {
		if _, ok := conflicts[strings.ToLower(strings.TrimSpace(value))]; ok {
			self = append(self, value)
		}
	}
	for _, value := range metadata.Intents {
		if _, ok := conflicts[strings.ToLower(strings.TrimSpace(value))]; ok {
			self = append(self, value)
		}
	}
	if metadata.Role != "" {
		value := string(metadata.Role)
		if _, ok := conflicts[value]; ok {
			self = append(self, value)
		}
	}
	return self
}

func sortedDiagnosticKeys(values map[string][]skillcatalog.ParsedSkill) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func descriptionDiagnostics(skill skillcatalog.ParsedSkill) []skillcatalog.Diagnostic {
	desc := strings.TrimSpace(skill.Description)
	if desc == "" {
		return nil
	}
	words := strings.Fields(desc)
	var diagnostics []skillcatalog.Diagnostic
	if len(words) < 6 {
		diagnostics = append(diagnostics, diag(skillcatalog.SeverityInfo, "description_too_short", skill.SkillPath, "description should include clearer trigger conditions for routing"))
	}
	lower := strings.ToLower(desc)
	if lower == "use this skill" || lower == "use when needed" || lower == "general workflow" {
		diagnostics = append(diagnostics, diag(skillcatalog.SeverityInfo, "description_too_broad", skill.SkillPath, "description is too generic for stable skill selection"))
	}
	return diagnostics
}

func writeSeveritySummary(b *strings.Builder, diagnostics []skillcatalog.Diagnostic) {
	counts := map[skillcatalog.DiagnosticSeverity]int{}
	for _, diag := range diagnostics {
		counts[diag.Severity]++
	}
	if counts[skillcatalog.SeverityError] > 0 {
		fmt.Fprintf(b, "- %d errors\n", counts[skillcatalog.SeverityError])
	}
	if counts[skillcatalog.SeverityWarning] > 0 {
		fmt.Fprintf(b, "- %d warnings\n", counts[skillcatalog.SeverityWarning])
	}
	if counts[skillcatalog.SeverityInfo] > 0 {
		fmt.Fprintf(b, "- %d info\n", counts[skillcatalog.SeverityInfo])
	}
}

func isBroadTrigger(trigger string) bool {
	switch strings.ToLower(strings.TrimSpace(trigger)) {
	case "code", "fix", "work", "task", "change", "file", "test":
		return true
	default:
		return false
	}
}

func skillNames(skills []skillcatalog.ParsedSkill) string {
	names := make([]string, 0, len(skills))
	for _, skill := range skills {
		name := sanitizeDoctorLine(skill.Name, "")
		if name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, ", ")
}

func sanitizeDoctorLine(value, fallback string) string {
	value = skillcatalog.SanitizePromptLineValue(value)
	if value == "" {
		return fallback
	}
	return value
}

func diag(severity skillcatalog.DiagnosticSeverity, code, path, message string) skillcatalog.Diagnostic {
	return skillcatalog.Diagnostic{
		Severity: severity,
		Code:     code,
		Path:     path,
		Message:  message,
	}
}
