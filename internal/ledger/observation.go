package ledger

import (
	"path/filepath"
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

type structuredObservationRecordResult struct {
	touchedFiles     bool
	evidence         bool
	recommendedReads bool
}

func collectToolObservationFacts(repoRoot, invocationCWD string, observation ToolObservation) toolObservationFacts {
	var facts toolObservationFacts
	facts.recordChangedFiles(repoRoot, invocationCWD, observation.Change)
	structuredResult := facts.recordStructuredObservation(repoRoot, invocationCWD, observation.Structured, observation)

	toolName := strings.TrimSpace(observation.ToolName)
	switch toolName {
	case "read_file":
		if !structuredResult.evidence {
			facts.recordFormattedEvidence(repoRoot, invocationCWD, observation.Result, observation, true, false)
		}
	case "search_code":
		facts.recordSearchLikeRenderedObservation(repoRoot, invocationCWD, observation, structuredResult, false)
	case "gather_context":
		facts.recordSearchLikeRenderedObservation(repoRoot, invocationCWD, observation, structuredResult, true)
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

func (f *toolObservationFacts) recordStructuredObservation(repoRoot, invocationCWD string, structured *tools.RuntimeObservation, observation ToolObservation) structuredObservationRecordResult {
	var result structuredObservationRecordResult
	if f == nil || structured == nil || structured.Empty() {
		return result
	}
	for _, file := range structured.TouchedFiles {
		if normalized, ok := normalizeStructuredObservationPath(repoRoot, invocationCWD, file.Path, file.ResolvedPath); ok {
			f.touchedFiles = appendUniqueString(f.touchedFiles, normalized, maxRecordedFiles)
			result.touchedFiles = true
		}
	}
	for _, evidence := range structured.Evidence {
		evidence = evidence.Normalize()
		normalized, ok := normalizeStructuredObservationPath(repoRoot, invocationCWD, evidence.Path, evidence.ResolvedPath)
		if !ok {
			continue
		}
		f.touchedFiles = appendUniqueString(f.touchedFiles, normalized, maxRecordedFiles)
		result.touchedFiles = true
		if f.recordEvidence(normalized, evidence.StartLine, evidence.EndLine, observation, evidence.Excerpt) {
			result.evidence = true
		}
	}
	for _, read := range structured.RecommendedReads {
		normalized, ok := normalizeStructuredObservationPath(repoRoot, invocationCWD, read.Path, read.ResolvedPath)
		if !ok {
			continue
		}
		if f.recordRecommendedRead(normalized, read.Reason, observation) {
			result.recommendedReads = true
		}
	}
	return result
}

func (f *toolObservationFacts) recordSearchLikeRenderedObservation(repoRoot, invocationCWD string, observation ToolObservation, structuredResult structuredObservationRecordResult, readHeaders bool) {
	if !structuredResult.evidence {
		f.recordFormattedEvidence(repoRoot, invocationCWD, observation.Result, observation, readHeaders, true)
	}
	f.recordRecommendedReads(repoRoot, invocationCWD, observation.Result, observation)
}

func normalizeStructuredObservationPath(repoRoot, invocationCWD, path, resolvedPath string) (string, bool) {
	if strings.TrimSpace(resolvedPath) != "" {
		return normalizeLedgerPath(repoRoot, invocationCWD, resolvedPath)
	}
	return normalizeLedgerPath(repoRoot, invocationCWD, path)
}

func (f *toolObservationFacts) recordFormattedEvidence(repoRoot, invocationCWD, output string, observation ToolObservation, readHeaders bool, searchHeaders bool) {
	if f == nil || strings.TrimSpace(output) == "" {
		return
	}

	currentPath := ""
	inRecommendedReads := false
	inAmbiguousSymbolList := false
	for _, rawLine := range strings.Split(output, "\n") {
		line := strings.TrimRight(rawLine, "\r")
		trimmed := strings.TrimSpace(line)
		if searchHeaders && isAmbiguousSymbolListHeader(trimmed) {
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
		if readHeaders {
			if path, _, _, ok := parseReadHeaderLine(line); ok {
				if normalized, pathOK := normalizeLedgerPath(repoRoot, invocationCWD, path); pathOK {
					currentPath = normalized
					f.touchedFiles = appendUniqueString(f.touchedFiles, normalized, maxRecordedFiles)
				} else {
					currentPath = ""
				}
				continue
			}
		}
		if searchHeaders {
			if path, ok := parseSearchHeaderLine(line); ok {
				if normalized, pathOK := normalizeSearchHeaderLedgerPath(repoRoot, invocationCWD, path, observation); pathOK {
					currentPath = normalized
					f.touchedFiles = appendUniqueString(f.touchedFiles, normalized, maxRecordedFiles)
				} else {
					currentPath = ""
				}
				continue
			}
		}
		if path, startLine, endLine, ok := parseSymbolHeaderLine(line); ok {
			if normalized, pathOK := normalizeSymbolObservationLedgerPath(repoRoot, invocationCWD, path); pathOK {
				currentPath = normalized
				f.touchedFiles = appendUniqueString(f.touchedFiles, normalized, maxRecordedFiles)
				f.recordEvidence(normalized, startLine, endLine, observation, line)
			} else {
				currentPath = ""
			}
			continue
		}
		if searchHeaders && !inRecommendedReads {
			if location, ok := parseRenderedPathLocationBullet(line); ok {
				if normalized, pathOK := normalizeSymbolObservationLedgerPath(repoRoot, invocationCWD, location.path); pathOK {
					f.touchedFiles = appendUniqueString(f.touchedFiles, normalized, maxRecordedFiles)
					f.recordEvidence(normalized, location.startLine, location.endLine, observation, locationEvidenceExcerpt(location, line))
				}
				continue
			}
		}
		if searchHeaders && inAmbiguousSymbolList {
			if candidate, ok := parseNumberedSymbolCandidateLine(line); ok {
				if normalized, pathOK := normalizeNumberedSymbolCandidateLedgerPath(repoRoot, invocationCWD, candidate); pathOK {
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

func normalizeSymbolObservationLedgerPath(repoRoot, invocationCWD, path string) (string, bool) {
	if isBareRelativeLedgerPath(path) {
		return normalizeLedgerPath(repoRoot, invocationCWD, path)
	}
	return normalizeRepoRelativeLedgerPath(repoRoot, invocationCWD, path)
}

func isBareRelativeLedgerPath(path string) bool {
	path = cleanPathCandidate(path)
	if path == "" || filepath.IsAbs(path) || isWindowsAbsPath(path) {
		return false
	}
	return !strings.ContainsAny(path, `/\`)
}

func normalizeNumberedSymbolCandidateLedgerPath(repoRoot, invocationCWD string, candidate numberedSymbolCandidate) (string, bool) {
	if candidate.repoPreferred {
		return normalizeRepoRelativeLedgerPath(repoRoot, invocationCWD, candidate.path)
	}
	return normalizeLedgerPath(repoRoot, invocationCWD, candidate.path)
}

func normalizeSearchHeaderLedgerPath(repoRoot, invocationCWD, path string, observation ToolObservation) (string, bool) {
	if searchObservationUsesRepoRelativeOutput(repoRoot, invocationCWD, observation) {
		return normalizeRepoRelativeLedgerPath(repoRoot, invocationCWD, path)
	}
	return normalizeLedgerPath(repoRoot, invocationCWD, path)
}

func searchObservationUsesRepoRelativeOutput(repoRoot, invocationCWD string, observation ToolObservation) bool {
	if observation.ToolName != "search_code" {
		return false
	}
	root := normalizeRepoRoot(repoRoot)
	if root == "" {
		return false
	}
	pathArg := strings.TrimSpace(observation.Args["path"])
	if pathArg == "" {
		return false
	}
	if filepath.IsAbs(pathArg) || isWindowsAbsPath(pathArg) {
		return pathIsWithinRepoRoot(root, normalizeRepoRoot(pathArg))
	}
	cwd := normalizeRepoRoot(invocationCWD)
	if cwd == "" {
		return false
	}
	return normalizeRepoRoot(filepath.Join(cwd, filepath.FromSlash(pathArg))) == root
}

func pathIsWithinRepoRoot(repoRoot, path string) bool {
	if repoRoot == "" || path == "" {
		return false
	}
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	_, ok := cleanLedgerRelativePath(rel)
	return ok
}

func (f *toolObservationFacts) recordRecommendedReads(repoRoot, invocationCWD, output string, observation ToolObservation) {
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
		normalized, pathOK := normalizeRecommendedReadLedgerPath(repoRoot, invocationCWD, path, observation)
		if !pathOK {
			continue
		}
		f.recordRecommendedRead(normalized, reason, observation)
	}
}

func normalizeRecommendedReadLedgerPath(repoRoot, invocationCWD, path string, observation ToolObservation) (string, bool) {
	switch observation.ToolName {
	case "search_code", "gather_context":
		return normalizeRepoRelativeLedgerPath(repoRoot, invocationCWD, path)
	default:
		return normalizeLedgerPath(repoRoot, invocationCWD, path)
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
