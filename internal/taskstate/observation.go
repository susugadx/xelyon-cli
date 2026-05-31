package taskstate

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ToolObservation は tool result から機械的に分かる runtime fact の入力。
type ToolObservation struct {
	ToolName      string
	ToolCallID    string
	InvocationCWD string
	Args          map[string]string
	Result        string
	Change        *tools.FileChange
	Structured    *tools.RuntimeObservation
	Error         bool
}

// TestObservation はテスト・検証コマンドの runtime fact 入力。
type TestObservation struct {
	Command  string
	ExitCode int
	Status   string
	Output   string
}

type toolObservationFacts struct {
	changedFiles     []string
	touchedFiles     []string
	evidence         []evidenceFact
	recommendedReads []recommendedReadFact
	failedTests      []TestResult
	passedTests      []TestResult
}

type observationFactCoverage struct {
	evidence         bool
	recommendedReads bool
}

type renderedObservationFallback struct {
	readHeaders      bool
	searchHeaders    bool
	evidence         bool
	recommendedReads bool
}

func (coverage observationFactCoverage) readFileRenderedFallback() renderedObservationFallback {
	return renderedObservationFallback{
		readHeaders: true,
		evidence:    !coverage.evidence,
	}
}

func (coverage observationFactCoverage) searchLikeRenderedFallback(readHeaders bool) renderedObservationFallback {
	return renderedObservationFallback{
		readHeaders:      readHeaders,
		searchHeaders:    true,
		evidence:         !coverage.evidence,
		recommendedReads: !coverage.recommendedReads,
	}
}

func collectToolObservationFacts(repoRoot, invocationCWD string, observation ToolObservation) toolObservationFacts {
	var facts toolObservationFacts
	facts.recordChangedFiles(repoRoot, invocationCWD, observation.Change)
	structuredCoverage := facts.recordStructuredObservation(repoRoot, invocationCWD, observation.Structured, observation)

	toolName := strings.TrimSpace(observation.ToolName)
	switch toolName {
	case "read_file":
		facts.recordRenderedObservationFallback(repoRoot, invocationCWD, observation, structuredCoverage.readFileRenderedFallback())
	case "search_code":
		facts.recordRenderedObservationFallback(repoRoot, invocationCWD, observation, structuredCoverage.searchLikeRenderedFallback(false))
	case "gather_context":
		facts.recordRenderedObservationFallback(repoRoot, invocationCWD, observation, structuredCoverage.searchLikeRenderedFallback(true))
	case "bash":
		facts.recordBashObservation(repoRoot, invocationCWD, observation)
	}

	return facts
}

func (f *toolObservationFacts) recordChangedFiles(repoRoot, invocationCWD string, change *tools.FileChange) {
	if f == nil || change == nil {
		return
	}
	for _, path := range changedPathsFromFileChange(*change) {
		normalized, ok := normalizeLedgerPath(repoRoot, invocationCWD, path)
		if !ok {
			continue
		}
		f.changedFiles = appendUniqueString(f.changedFiles, normalized, maxRecordedFiles)
		f.touchedFiles = appendUniqueString(f.touchedFiles, normalized, maxRecordedFiles)
	}
}

func (f *toolObservationFacts) recordStructuredObservation(repoRoot, invocationCWD string, structured *tools.RuntimeObservation, observation ToolObservation) observationFactCoverage {
	var coverage observationFactCoverage
	if f == nil || structured == nil || structured.Empty() {
		return coverage
	}
	for _, file := range structured.TouchedFiles {
		if normalized, ok := normalizeStructuredObservationPath(repoRoot, invocationCWD, file.Path, file.ResolvedPath); ok {
			f.touchedFiles = appendUniqueString(f.touchedFiles, normalized, maxRecordedFiles)
		}
	}
	for _, evidence := range structured.Evidence {
		evidence = evidence.Normalize()
		normalized, ok := normalizeStructuredObservationPath(repoRoot, invocationCWD, evidence.Path, evidence.ResolvedPath)
		if !ok {
			if structuredObservationHasResolvedPath(evidence.ResolvedPath) {
				coverage.evidence = true
			}
			continue
		}
		f.touchedFiles = appendUniqueString(f.touchedFiles, normalized, maxRecordedFiles)
		if f.recordEvidence(normalized, evidence.StartLine, evidence.EndLine, observation, evidence.Excerpt) {
			coverage.evidence = true
		}
	}
	for _, read := range structured.RecommendedReads {
		normalized, ok := normalizeStructuredObservationPath(repoRoot, invocationCWD, read.Path, read.ResolvedPath)
		if !ok {
			if structuredObservationHasResolvedPath(read.ResolvedPath) {
				coverage.recommendedReads = true
			}
			continue
		}
		if f.recordRecommendedRead(normalized, read.Reason, observation) {
			coverage.recommendedReads = true
		}
	}
	return coverage
}

func (f *toolObservationFacts) recordRenderedObservationFallback(repoRoot, invocationCWD string, observation ToolObservation, fallback renderedObservationFallback) {
	pathPolicy := newRenderedObservationPathPolicy(repoRoot, invocationCWD, observation)
	if fallback.evidence {
		f.recordFormattedEvidence(observation.Result, observation, fallback, pathPolicy)
	}
	if fallback.recommendedReads {
		f.recordRecommendedReads(observation.Result, observation, pathPolicy)
	}
}

func normalizeStructuredObservationPath(repoRoot, invocationCWD, path, resolvedPath string) (string, bool) {
	if strings.TrimSpace(resolvedPath) != "" {
		return normalizeLedgerPath(repoRoot, invocationCWD, resolvedPath)
	}
	return normalizeLedgerPath(repoRoot, invocationCWD, path)
}

func structuredObservationHasResolvedPath(resolvedPath string) bool {
	return strings.TrimSpace(resolvedPath) != ""
}

func (f *toolObservationFacts) recordFormattedEvidence(output string, observation ToolObservation, fallback renderedObservationFallback, pathPolicy renderedObservationPathPolicy) {
	if f == nil || strings.TrimSpace(output) == "" {
		return
	}

	currentPath := ""
	inRecommendedReads := false
	inAmbiguousSymbolList := false
	for _, rawLine := range strings.Split(output, "\n") {
		line := strings.TrimRight(rawLine, "\r")
		trimmed := strings.TrimSpace(line)
		if fallback.searchHeaders && isAmbiguousSymbolListHeader(trimmed) {
			inAmbiguousSymbolList = true
			currentPath = ""
			continue
		}
		if inAmbiguousSymbolList && trimmed == "" {
			inAmbiguousSymbolList = false
			continue
		}
		if strings.EqualFold(trimmed, "Recommended reads:") {
			inRecommendedReads = true
			inAmbiguousSymbolList = false
			continue
		}
		if inRecommendedReads && trimmed == "" {
			inRecommendedReads = false
		}
		if fallback.readHeaders {
			if path, _, _, ok := parseReadHeaderLine(line); ok {
				if normalized, pathOK := pathPolicy.normalizeReadHeader(path); pathOK {
					currentPath = normalized
					f.touchedFiles = appendUniqueString(f.touchedFiles, normalized, maxRecordedFiles)
				} else {
					currentPath = ""
				}
				continue
			}
		}
		if fallback.searchHeaders {
			if path, ok := parseSearchHeaderLine(line); ok {
				if normalized, pathOK := pathPolicy.normalizeSearchHeader(path); pathOK {
					currentPath = normalized
					f.touchedFiles = appendUniqueString(f.touchedFiles, normalized, maxRecordedFiles)
				} else {
					currentPath = ""
				}
				continue
			}
		}
		if path, startLine, endLine, ok := parseSymbolHeaderLine(line); ok {
			if normalized, pathOK := pathPolicy.normalizeSymbolPath(path); pathOK {
				currentPath = normalized
				f.touchedFiles = appendUniqueString(f.touchedFiles, normalized, maxRecordedFiles)
				f.recordEvidence(normalized, startLine, endLine, observation, line)
			} else {
				currentPath = ""
			}
			continue
		}
		if fallback.searchHeaders && !inRecommendedReads {
			if location, ok := parseRenderedPathLocationBullet(line); ok {
				if normalized, pathOK := pathPolicy.normalizeSymbolPath(location.path); pathOK {
					f.touchedFiles = appendUniqueString(f.touchedFiles, normalized, maxRecordedFiles)
					f.recordEvidence(normalized, location.startLine, location.endLine, observation, locationEvidenceExcerpt(location, line))
				}
				continue
			}
		}
		if fallback.searchHeaders && inAmbiguousSymbolList {
			if candidate, ok := parseNumberedSymbolCandidateLine(line); ok {
				if normalized, pathOK := pathPolicy.normalizeNumberedSymbolCandidate(candidate); pathOK {
					f.touchedFiles = appendUniqueString(f.touchedFiles, normalized, maxRecordedFiles)
					f.recordEvidence(normalized, candidate.startLine, candidate.endLine, observation, line)
				}
				continue
			}
		}
		if currentPath == "" {
			continue
		}
		if lineNo, excerpt, ok := parseFormattedEvidenceLine(line); ok {
			f.recordEvidence(currentPath, lineNo, lineNo, observation, excerpt)
		}
	}
}

func (f *toolObservationFacts) recordRecommendedReads(output string, observation ToolObservation, pathPolicy renderedObservationPathPolicy) {
	if f == nil || strings.TrimSpace(output) == "" {
		return
	}

	inBlock := false
	for _, rawLine := range strings.Split(output, "\n") {
		line := strings.TrimSpace(rawLine)
		if strings.EqualFold(line, "Recommended reads:") {
			inBlock = true
			continue
		}
		if !inBlock {
			continue
		}
		if line == "" {
			inBlock = false
			continue
		}
		match := recommendedReadItemRe.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		path, reason, ok := parseRecommendedReadItem(match[1])
		if !ok {
			continue
		}
		normalized, pathOK := pathPolicy.normalizeRecommendedRead(path)
		if !pathOK {
			continue
		}
		f.recordRecommendedRead(normalized, reason, observation)
	}
}

func (f *toolObservationFacts) recordBashObservation(repoRoot, invocationCWD string, observation ToolObservation) {
	if f == nil {
		return
	}
	for _, path := range parsePathLineFacts(repoRoot, invocationCWD, observation.Result) {
		f.touchedFiles = appendUniqueString(f.touchedFiles, path, maxRecordedFiles)
	}
	if !isTestLikeCommand(observation.Args["command"]) {
		return
	}
	exitCode := bashExitCode(observation.Result, observation.Error)
	result, ok := testResultFromObservation(TestObservation{
		Command:  observation.Args["command"],
		ExitCode: exitCode,
		Output:   observation.Result,
	})
	if !ok {
		return
	}
	if result.Status() == "passed" {
		f.passedTests = append(f.passedTests, result)
		return
	}
	f.failedTests = append(f.failedTests, result)
}

func newEvidenceFact(path string, startLine, endLine int, observation ToolObservation, excerpt string) evidenceFact {
	return evidenceFact{
		path:       path,
		startLine:  startLine,
		endLine:    endLine,
		source:     observation.ToolName,
		toolCallID: observation.ToolCallID,
		excerpt:    truncateBytes(strings.TrimSpace(excerpt), maxFactExcerptBytes),
	}
}

func (f *toolObservationFacts) recordEvidence(path string, startLine, endLine int, observation ToolObservation, excerpt string) bool {
	if f == nil {
		return false
	}
	fact, ok := prepareEvidenceFact(newEvidenceFact(path, startLine, endLine, observation, excerpt))
	if !ok {
		return false
	}
	f.evidence = append(f.evidence, fact)
	return true
}

func (f *toolObservationFacts) recordRecommendedRead(path, reason string, observation ToolObservation) bool {
	if f == nil || path == "" {
		return false
	}
	f.recommendedReads = append(f.recommendedReads, recommendedReadFact{
		path:       path,
		reason:     truncateBytes(strings.TrimSpace(reason), maxFactExcerptBytes),
		source:     observation.ToolName,
		toolCallID: observation.ToolCallID,
	})
	return true
}

func truncateBytes(s string, limit int) string {
	if limit <= 0 || len(s) <= limit {
		return s
	}
	const suffix = "\n... (truncated)"
	if limit <= len(suffix) {
		return s[:limit]
	}
	return s[:limit-len(suffix)] + suffix
}

func appendUniqueString(items []string, item string, limit int) []string {
	if item == "" {
		return items
	}
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	if limit > 0 && len(items) >= limit {
		return items
	}
	return append(items, item)
}
