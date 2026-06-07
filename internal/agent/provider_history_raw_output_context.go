package agent

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	"github.com/susugadx/xelyon-cli/internal/token"
)

const (
	providerHistoryRawOutputActiveContextName                       = "provider_history_raw_output_context"
	providerHistoryRawOutputContextHeader                           = "Provider History Raw Output Context"
	providerHistoryRawOutputRequiredRefsMissingReason               = "raw_output_active_context_required_refs_missing"
	providerHistoryRawOutputActiveContextCoverageInsufficientReason = "raw_output_active_context_coverage_insufficient"
)

type rawOutputArtifactResolver interface {
	Resolve(ctx context.Context, ref rawoutputs.RawOutputRef) (rawoutputs.ResolvedArtifact, error)
}

type providerHistoryRawOutputActiveContextBuild struct {
	Blocks                     []api.ActiveContextBlock
	RequiredRefCount           int
	InjectedRefCount           int
	MissingRefIDs              []string
	CoverageInsufficientRefIDs []string
}

func (b providerHistoryRawOutputActiveContextBuild) missingRequiredRefs() bool {
	return b.RequiredRefCount > b.InjectedRefCount || len(b.MissingRefIDs) > 0 || len(b.CoverageInsufficientRefIDs) > 0
}

func (b providerHistoryRawOutputActiveContextBuild) failClosedReason() string {
	if len(b.CoverageInsufficientRefIDs) > 0 {
		return providerHistoryRawOutputActiveContextCoverageInsufficientReason
	}
	return providerHistoryRawOutputRequiredRefsMissingReason
}

func (a *Agent) buildProviderHistoryRawOutputActiveContext(ctx context.Context, report ProviderHistoryProjectionReport, raw []api.Message) providerHistoryRawOutputActiveContextBuild {
	refs := providerHistoryAppliedRawOutputRefs(report)
	result := providerHistoryRawOutputActiveContextBuild{RequiredRefCount: len(refs)}
	if len(refs) == 0 {
		return result
	}
	if !a.shouldBuildProviderHistoryRawOutputActiveContext() {
		result.MissingRefIDs = providerHistoryRawOutputRefIDs(refs)
		return result
	}
	resolver := a.providerHistoryRawOutputArtifactResolver()
	if resolver == nil {
		result.MissingRefIDs = providerHistoryRawOutputRefIDs(refs)
		return result
	}

	budget := providerHistoryRawOutputActiveContextBudget(a.Runtime)
	var b strings.Builder
	b.WriteString(providerHistoryRawOutputContextHeader)
	usedTokens := token.EstimateTokenCount(providerHistoryRawOutputContextHeader)
	injected := 0
	hints := providerHistoryRawOutputRehydrateHintsFromRaw(raw)
	for _, ref := range refs {
		if strings.EqualFold(ref.SemanticRole, "sensitive") {
			result.MissingRefIDs = append(result.MissingRefIDs, ref.RefID)
			continue
		}
		resolved, err := resolver.Resolve(ctx, ref)
		if err != nil {
			result.MissingRefIDs = append(result.MissingRefIDs, ref.RefID)
			continue
		}
		body, readErr := io.ReadAll(resolved.Body)
		_ = resolved.Body.Close()
		if readErr != nil {
			result.MissingRefIDs = append(result.MissingRefIDs, ref.RefID)
			continue
		}
		entry, reason := renderProviderHistoryRawOutputContextEntry(ref, string(body), budget-usedTokens, hints)
		entryTokens := token.EstimateTokenCount(entry)
		if entry == "" || entryTokens <= 0 || usedTokens+entryTokens > budget {
			if reason == providerHistoryRawOutputActiveContextCoverageInsufficientReason {
				result.CoverageInsufficientRefIDs = append(result.CoverageInsufficientRefIDs, ref.RefID)
			} else {
				result.MissingRefIDs = append(result.MissingRefIDs, ref.RefID)
			}
			continue
		}
		b.WriteString("\n")
		b.WriteString(entry)
		usedTokens += entryTokens
		injected++
	}
	result.InjectedRefCount = injected
	if injected == 0 {
		return result
	}
	result.Blocks = []api.ActiveContextBlock{{
		Name:    providerHistoryRawOutputActiveContextName,
		Content: b.String(),
	}}
	return result
}

func (a *Agent) shouldBuildProviderHistoryRawOutputActiveContext() bool {
	if a == nil || a.Runtime == nil {
		return false
	}
	return a.Runtime.Options.EnableProviderHistoryRehydrateContext && a.providerCanConsumeActiveContext()
}

func (a *Agent) providerHistoryRawOutputArtifactResolver() rawOutputArtifactResolver {
	if a == nil || a.Runtime == nil {
		return nil
	}
	if resolver, ok := a.Runtime.RawOutputArtifactStore.(rawOutputArtifactResolver); ok {
		return resolver
	}
	if resolver, ok := a.providerHistoryRawOutputArtifactStore().(rawOutputArtifactResolver); ok {
		return resolver
	}
	return nil
}

func providerHistoryAppliedRawOutputRefs(report ProviderHistoryProjectionReport) []rawoutputs.RawOutputRef {
	if len(report.RawOutputRefs) == 0 || (len(report.Candidates) == 0 && len(report.CommandEditDryRun.Candidates) == 0) {
		return nil
	}
	refsByID := make(map[string]rawoutputs.RawOutputRef, len(report.RawOutputRefs))
	for _, ref := range report.RawOutputRefs {
		if ref.RefID == "" {
			continue
		}
		refsByID[ref.RefID] = ref
	}
	seen := make(map[string]struct{})
	refs := make([]rawoutputs.RawOutputRef, 0)
	for _, candidate := range report.Candidates {
		if !candidate.ArtifactBackedCandidate || !candidate.ReplacementApplied || candidate.RawOutputRefID == "" {
			continue
		}
		if _, exists := seen[candidate.RawOutputRefID]; exists {
			continue
		}
		ref, ok := refsByID[candidate.RawOutputRefID]
		if !ok {
			continue
		}
		seen[candidate.RawOutputRefID] = struct{}{}
		refs = append(refs, ref)
	}
	for _, candidate := range report.CommandEditDryRun.Candidates {
		if !candidate.ArtifactBackedCandidate || !candidate.ReplacementApplied || candidate.RawOutputRefID == "" {
			continue
		}
		if _, exists := seen[candidate.RawOutputRefID]; exists {
			continue
		}
		ref, ok := refsByID[candidate.RawOutputRefID]
		if !ok {
			continue
		}
		seen[candidate.RawOutputRefID] = struct{}{}
		refs = append(refs, ref)
	}
	return refs
}

func providerHistoryRawOutputRefIDs(refs []rawoutputs.RawOutputRef) []string {
	if len(refs) == 0 {
		return nil
	}
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.RefID == "" {
			continue
		}
		ids = append(ids, ref.RefID)
	}
	return ids
}

func renderProviderHistoryRawOutputContextEntry(ref rawoutputs.RawOutputRef, body string, availableTokens int, hints []string) (string, string) {
	if availableTokens <= 0 {
		return "", providerHistoryRawOutputRequiredRefsMissingReason
	}
	body = providerHistoryRawOutputContextDisplayBody(ref, body)
	metadata := fmt.Sprintf(
		"- ref: %s\n  surface: %s\n  tool_name: %s\n  command_preview: %s\n  family: %s\n  classifier: %s\n  byte_size: %d\n  content_hash: %s\n  body:\n",
		ref.RefID,
		ref.Surface,
		ref.ToolName,
		ref.CommandPreview,
		ref.Family,
		ref.Classifier,
		ref.ByteSize,
		ref.ContentHash,
	)
	metadataTokens := token.EstimateTokenCount(metadata)
	bodyBudget := availableTokens - metadataTokens
	if bodyBudget <= 0 {
		return "", providerHistoryRawOutputRequiredRefsMissingReason
	}
	for bodyBudget > 0 {
		excerpt, reason := providerHistoryRawOutputBodyCoverageExcerpt(body, bodyBudget, hints)
		if strings.TrimSpace(excerpt) == "" {
			return "", reason
		}
		entry := metadata + indentRawOutputBody(excerpt)
		if token.EstimateTokenCount(entry) <= availableTokens {
			return entry, ""
		}
		if reason == providerHistoryRawOutputActiveContextCoverageInsufficientReason {
			return "", reason
		}
		bodyBudget = bodyBudget * 3 / 4
	}
	return "", providerHistoryRawOutputRequiredRefsMissingReason
}

func providerHistoryRawOutputContextDisplayBody(ref rawoutputs.RawOutputRef, body string) string {
	if ref.Surface == string(rawoutputs.SurfaceXelyonWebSearchToolResult) {
		return rawoutputs.RedactDisplaySecrets(body)
	}
	return body
}

func providerHistoryRawOutputBodyExcerpt(body string, budgetTokens int) string {
	body = strings.TrimSpace(body)
	if body == "" || budgetTokens <= 0 {
		return ""
	}
	if token.EstimateTokenCount(body) <= budgetTokens {
		return body
	}
	maxRunes := budgetTokens * 2
	if maxRunes < 256 {
		maxRunes = 256
	}
	runes := []rune(body)
	if len(runes) <= maxRunes {
		return body
	}
	headRunes := maxRunes / 2
	tailRunes := maxRunes - headRunes
	head := strings.TrimSpace(string(runes[:headRunes]))
	tail := strings.TrimSpace(string(runes[len(runes)-tailRunes:]))
	return head + "\n...\n" + tail
}

func providerHistoryRawOutputBodyCoverageExcerpt(body string, budgetTokens int, hints []string) (string, string) {
	body = strings.TrimSpace(body)
	if body == "" || budgetTokens <= 0 {
		return "", providerHistoryRawOutputRequiredRefsMissingReason
	}
	if token.EstimateTokenCount(body) <= budgetTokens {
		return body, ""
	}
	excerpt, ok := providerHistoryRawOutputMatchedBodyExcerpt(body, hints, budgetTokens)
	if ok {
		return excerpt, ""
	}
	return "", providerHistoryRawOutputActiveContextCoverageInsufficientReason
}

func providerHistoryRawOutputMatchedBodyExcerpt(body string, hints []string, budgetTokens int) (string, bool) {
	if budgetTokens <= 0 || len(hints) == 0 {
		return "", false
	}
	lines := strings.Split(strings.TrimSpace(body), "\n")
	lineIndex, matchedTerm := providerHistoryRawOutputMatchedLine(lines, hints)
	if lineIndex < 0 {
		return "", false
	}
	maxRadius := providerHistoryRawOutputMinInt(4, providerHistoryRawOutputMaxInt(lineIndex, len(lines)-lineIndex-1))
	for radius := maxRadius; radius >= 0; radius-- {
		start := providerHistoryRawOutputMaxInt(0, lineIndex-radius)
		end := providerHistoryRawOutputMinInt(len(lines), lineIndex+radius+1)
		selected := append([]string(nil), lines[start:end]...)
		if radius == 0 {
			selected[0] = providerHistoryRawOutputTrimMatchedLine(selected[0], matchedTerm, budgetTokens)
		}
		excerpt := providerHistoryRawOutputRenderMatchedExcerpt(selected, matchedTerm, lineIndex, len(lines), start, end)
		if token.EstimateTokenCount(excerpt) <= budgetTokens {
			return excerpt, true
		}
	}
	return "", false
}

func providerHistoryRawOutputMatchedLine(lines []string, hints []string) (int, string) {
	for i, line := range lines {
		lowerLine := strings.ToLower(line)
		for _, hint := range hints {
			if hint != "" && strings.Contains(lowerLine, hint) {
				return i, hint
			}
		}
	}
	return -1, ""
}

func providerHistoryRawOutputRenderMatchedExcerpt(lines []string, matchedTerm string, lineIndex, totalLines, start, end int) string {
	parts := []string{fmt.Sprintf(
		"[matched raw output excerpt; matched_term=%q; line=%d/%d]",
		rawoutputs.RedactDisplaySecrets(matchedTerm),
		lineIndex+1,
		totalLines,
	)}
	if start > 0 {
		parts = append(parts, fmt.Sprintf("[omitted %d lines before match]", start))
	}
	parts = append(parts, lines...)
	if end < totalLines {
		parts = append(parts, fmt.Sprintf("[omitted %d lines after match]", totalLines-end))
	}
	return strings.Join(parts, "\n")
}

func providerHistoryRawOutputTrimMatchedLine(line, matchedTerm string, budgetTokens int) string {
	line = strings.TrimSpace(line)
	if line == "" || budgetTokens <= 0 || token.EstimateTokenCount(line) <= budgetTokens {
		return line
	}
	maxRunes := budgetTokens * 2
	if maxRunes < 256 {
		maxRunes = 256
	}
	runes := []rune(line)
	if len(runes) <= maxRunes {
		return line
	}
	index := providerHistoryRawOutputMatchedRuneIndex(line, matchedTerm)
	if index < 0 {
		return providerHistoryRawOutputBodyExcerpt(line, budgetTokens)
	}
	start := index - maxRunes/2
	if start < 0 {
		start = 0
	}
	end := start + maxRunes
	if end > len(runes) {
		end = len(runes)
		start = providerHistoryRawOutputMaxInt(0, end-maxRunes)
	}
	trimmed := strings.TrimSpace(string(runes[start:end]))
	if start > 0 {
		trimmed = "..." + trimmed
	}
	if end < len(runes) {
		trimmed += "..."
	}
	return trimmed
}

func providerHistoryRawOutputMatchedRuneIndex(line, matchedTerm string) int {
	byteIndex := strings.Index(strings.ToLower(line), strings.ToLower(matchedTerm))
	if byteIndex < 0 {
		return -1
	}
	return len([]rune(line[:byteIndex]))
}

func providerHistoryRawOutputRehydrateHintsFromRaw(raw []api.Message) []string {
	for i := len(raw) - 1; i >= 0; i-- {
		if raw[i].Role == "user" {
			return providerHistoryRawOutputRehydrateHints(raw[i].Content)
		}
	}
	return nil
}

func providerHistoryRawOutputRehydrateHints(value string) []string {
	words := providerHistoryRawOutputHintWords(value)
	seen := make(map[string]struct{}, len(words))
	hints := make([]string, 0, len(words))
	for _, word := range words {
		word = strings.ToLower(strings.Trim(word, ".,;:!?()[]{}<>\"'"))
		if word == "" || providerHistoryRawOutputHintStopWords[word] {
			continue
		}
		if len([]rune(word)) < 4 && !providerHistoryRawOutputContainsDigit(word) {
			continue
		}
		if _, ok := seen[word]; ok {
			continue
		}
		seen[word] = struct{}{}
		hints = append(hints, word)
	}
	sort.SliceStable(hints, func(i, j int) bool {
		leftDigit := providerHistoryRawOutputContainsDigit(hints[i])
		rightDigit := providerHistoryRawOutputContainsDigit(hints[j])
		if leftDigit != rightDigit {
			return leftDigit
		}
		return len(hints[i]) > len(hints[j])
	})
	if len(hints) > 32 {
		return hints[:32]
	}
	return hints
}

func providerHistoryRawOutputHintWords(value string) []string {
	words := make([]string, 0)
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		words = append(words, b.String())
		b.Reset()
	}
	for _, r := range value {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_' || r == '-' || r == '.' || r == '/' || r == ':':
			b.WriteRune(r)
		default:
			flush()
		}
	}
	flush()
	return words
}

var providerHistoryRawOutputHintStopWords = map[string]bool{
	"about":   true,
	"again":   true,
	"check":   true,
	"current": true,
	"history": true,
	"inspect": true,
	"latest":  true,
	"next":    true,
	"please":  true,
	"request": true,
	"result":  true,
	"show":    true,
	"that":    true,
	"this":    true,
	"with":    true,
}

func providerHistoryRawOutputContainsDigit(value string) bool {
	for _, r := range value {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func providerHistoryRawOutputMinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func providerHistoryRawOutputMaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func indentRawOutputBody(body string) string {
	lines := strings.Split(strings.TrimSpace(body), "\n")
	for i, line := range lines {
		lines[i] = "    " + line
	}
	return strings.Join(lines, "\n")
}

func providerHistoryRawOutputActiveContextBudget(runtime *AgentRuntime) int {
	defaults := config.DefaultProviderHistoryRawOutputArtifactsConfig()
	if runtime == nil {
		return defaults.ActiveContextBudgetTokens
	}
	cfg := runtime.Options.ProviderHistoryRawOutputArtifacts
	budget := cfg.ActiveContextBudgetTokens
	if budget <= 0 {
		budget = defaults.ActiveContextBudgetTokens
	}
	maxBudget := cfg.ActiveContextBudgetMaxTokens
	if maxBudget <= 0 {
		maxBudget = defaults.ActiveContextBudgetMaxTokens
	}
	if budget > maxBudget {
		return maxBudget
	}
	return budget
}
