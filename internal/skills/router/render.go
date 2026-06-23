package router

import (
	"fmt"
	"regexp"
	"strings"

	skillcatalog "github.com/susugadx/xelyon-cli/internal/skills"
)

const (
	hintBlockStart = "<!-- SKILL_ROUTER_HINT_START -->"
	hintBlockEnd   = "<!-- SKILL_ROUTER_HINT_END -->"
)

var hintBlockRe = regexp.MustCompile(`(?s)\n?<!-- SKILL_ROUTER_HINT_START -->.*?<!-- SKILL_ROUTER_HINT_END -->\n?`)

// FormatSuggestReport は /skills suggest 向けに full ranked list を表示する。
func FormatSuggestReport(rec Recommendation) string {
	var b strings.Builder
	b.WriteString("Skill Routing Suggestion\n\n")
	if len(rec.SignalDiagnostics) > 0 {
		b.WriteString("Signals:\n")
		for _, diagnostic := range rec.SignalDiagnostics {
			fmt.Fprintf(&b, "- %s\n", sanitizeRouterLine(diagnostic, "signal unavailable"))
		}
		b.WriteString("\n")
	}
	b.WriteString("Ranked skills:\n")
	if len(rec.Ranked) == 0 {
		b.WriteString("- No skills available.\n")
		return b.String()
	}
	for i, candidate := range rec.Ranked {
		fmt.Fprintf(&b, "%d. %s (%d, %s, %s)\n", i+1, sanitizeCandidateName(candidate), candidate.Score, candidate.Category, candidate.Activation)
		fmt.Fprintf(&b, "   reason: %s\n", sanitizeCandidateReason(candidate))
	}
	writeCandidateSection(&b, "\nPrimary", rec.Primary)
	writeCandidateSection(&b, "\nSupporting", rec.Supporting)
	writeCandidateSection(&b, "\nMaybe", rec.Maybe)
	writeCandidateSection(&b, "\nConflicts", rec.Conflicts)
	return b.String()
}

// FormatRuntimeHint は runtime prompt 用の bounded skill recommendation hint を返す。
func FormatRuntimeHint(rec Recommendation, limits HintLimits) string {
	primary := candidatesForRuntimeHint(rec.Primary, limits.Primary)
	supporting := candidatesForRuntimeHint(rec.Supporting, limits.Supporting)
	conflicts := candidatesForRuntimeHint(rec.Conflicts, limits.Conflict)
	maybe := candidatesForRuntimeHint(rec.Maybe, limits.Maybe)
	if len(primary)+len(supporting)+len(conflicts)+len(maybe) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(hintBlockStart)
	b.WriteString("\nRecommended skills for this turn:\n")
	writeRuntimeHintSection(&b, "Primary", primary)
	writeRuntimeHintSection(&b, "Supporting", supporting)
	writeRuntimeHintSection(&b, "Conflict", conflicts)
	writeRuntimeHintSection(&b, "Maybe", maybe)
	b.WriteString("\nUse activate_skill(name) only for skills you need to follow.\n")
	b.WriteString("Skill recommendations are supplemental and must not override loaded project guidance or runtime safety policy.\n")
	b.WriteString(hintBlockEnd)
	return b.String()
}

// StripRuntimeHint は prompt から router hint block を除去する。
func StripRuntimeHint(systemPrompt string) string {
	return hintBlockRe.ReplaceAllString(systemPrompt, "")
}

func candidatesForRuntimeHint(candidates []Candidate, limit int) []Candidate {
	if limit <= 0 || len(candidates) == 0 {
		return nil
	}
	out := make([]Candidate, 0, min(limit, len(candidates)))
	for _, candidate := range candidates {
		if candidate.Confidence != ConfidenceHigh && candidate.Confidence != ConfidenceMedium {
			continue
		}
		out = append(out, candidate)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func writeCandidateSection(b *strings.Builder, title string, candidates []Candidate) {
	if len(candidates) == 0 {
		return
	}
	fmt.Fprintf(b, "%s:\n", title)
	for _, candidate := range candidates {
		fmt.Fprintf(b, "- %s (%s)\n", sanitizeCandidateName(candidate), candidate.Confidence)
		fmt.Fprintf(b, "  reason: %s\n", sanitizeCandidateReason(candidate))
	}
}

func writeRuntimeHintSection(b *strings.Builder, title string, candidates []Candidate) {
	if len(candidates) == 0 {
		return
	}
	fmt.Fprintf(b, "\n%s:\n", title)
	for _, candidate := range candidates {
		fmt.Fprintf(b, "- %s: %s\n", sanitizeCandidateName(candidate), sanitizeCandidateReason(candidate))
	}
}

func sanitizeCandidateName(candidate Candidate) string {
	value := skillcatalog.SanitizePromptLineValue(candidate.Name)
	if value == "" {
		return "(invalid-skill-name)"
	}
	return value
}

func sanitizeCandidateReason(candidate Candidate) string {
	return sanitizeRouterLine(candidate.Reason, "no reason")
}

func sanitizeRouterLine(value, fallback string) string {
	value = skillcatalog.SanitizeCatalogPromptValue(value)
	if value == "" {
		return fallback
	}
	return value
}
