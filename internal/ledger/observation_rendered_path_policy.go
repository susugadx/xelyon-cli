package ledger

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/filefilter"
)

type renderedObservationPathPolicy struct {
	repoRoot         string
	invocationCWD    string
	toolName         string
	searchPathArg    string
	searchOutputRoot string
}

func newRenderedObservationPathPolicy(repoRoot, invocationCWD string, observation ToolObservation) renderedObservationPathPolicy {
	return renderedObservationPathPolicy{
		repoRoot:         repoRoot,
		invocationCWD:    invocationCWD,
		toolName:         strings.TrimSpace(observation.ToolName),
		searchPathArg:    strings.TrimSpace(observation.Args["path"]),
		searchOutputRoot: renderedSearchOutputRoot(repoRoot, observation),
	}
}

func renderedSearchOutputRoot(repoRoot string, observation ToolObservation) string {
	if !toolEmitsRenderedSearchOutput(observation.ToolName) {
		return ""
	}
	pathArg := strings.TrimSpace(observation.Args["path"])
	if pathArg == "" || !isAbsoluteLedgerPath(pathArg) {
		return ""
	}
	workspaceRoot := normalizeRepoRoot(repoRoot)
	basis := filefilter.ResolveSearchPathBasisWithWorkspace(pathArg, workspaceRoot)
	return normalizeRepoRoot(basis.MatchRoot)
}

func (p renderedObservationPathPolicy) normalizeReadHeader(path string) (string, bool) {
	return normalizeLedgerPath(p.repoRoot, p.invocationCWD, path)
}

func (p renderedObservationPathPolicy) normalizeSearchHeader(path string) (string, bool) {
	if p.searchOutputRoot != "" {
		return p.normalizeSearchOutputPath(path)
	}
	if p.searchObservationUsesRepoRelativeOutput() {
		return normalizeRepoRelativeLedgerPath(p.repoRoot, p.invocationCWD, path)
	}
	return normalizeLedgerPath(p.repoRoot, p.invocationCWD, path)
}

func (p renderedObservationPathPolicy) normalizeSymbolPath(path string) (string, bool) {
	if p.searchOutputRoot != "" {
		return p.normalizeSearchOutputPath(path)
	}
	if isBareRelativeLedgerPath(path) {
		return normalizeLedgerPath(p.repoRoot, p.invocationCWD, path)
	}
	return normalizeRepoRelativeLedgerPath(p.repoRoot, p.invocationCWD, path)
}

func (p renderedObservationPathPolicy) normalizeNumberedSymbolCandidate(candidate numberedSymbolCandidate) (string, bool) {
	if p.searchOutputRoot != "" {
		return p.normalizeSearchOutputPath(candidate.path)
	}
	if candidate.repoPreferred {
		return normalizeRepoRelativeLedgerPath(p.repoRoot, p.invocationCWD, candidate.path)
	}
	return normalizeLedgerPath(p.repoRoot, p.invocationCWD, candidate.path)
}

func (p renderedObservationPathPolicy) normalizeRecommendedRead(path string) (string, bool) {
	if p.searchOutputRoot != "" {
		return p.normalizeSearchOutputPath(path)
	}
	switch p.toolName {
	case "search_code", "gather_context":
		return normalizeRepoRelativeLedgerPath(p.repoRoot, p.invocationCWD, path)
	default:
		return normalizeLedgerPath(p.repoRoot, p.invocationCWD, path)
	}
}

func (p renderedObservationPathPolicy) normalizeSearchOutputPath(path string) (string, bool) {
	path = cleanPathCandidate(path)
	if !isLedgerPathCandidateSafe(path) {
		return "", false
	}
	root := normalizeRepoRoot(p.repoRoot)
	if root == "" {
		return "", false
	}
	if isAbsoluteLedgerPath(path) {
		return normalizeAbsoluteLedgerPath(root, path)
	}
	absolute := filepath.Clean(filepath.Join(p.searchOutputRoot, filepath.FromSlash(path)))
	return normalizeAbsoluteLedgerPath(root, absolute)
}

func (p renderedObservationPathPolicy) searchObservationUsesRepoRelativeOutput() bool {
	if !toolEmitsRenderedSearchOutput(p.toolName) {
		return false
	}
	root := normalizeRepoRoot(p.repoRoot)
	if root == "" {
		return false
	}
	if p.searchPathArg == "" {
		return false
	}
	if isAbsoluteLedgerPath(p.searchPathArg) {
		return pathIsWithinRepoRoot(root, normalizeRepoRoot(p.searchPathArg))
	}
	cwd := normalizeRepoRoot(p.invocationCWD)
	if cwd == "" {
		return false
	}
	return normalizeRepoRoot(filepath.Join(cwd, filepath.FromSlash(p.searchPathArg))) == root
}

func toolEmitsRenderedSearchOutput(toolName string) bool {
	switch strings.TrimSpace(toolName) {
	case "search_code", "gather_context":
		return true
	default:
		return false
	}
}

func isBareRelativeLedgerPath(path string) bool {
	path = cleanPathCandidate(path)
	if path == "" || isAbsoluteLedgerPath(path) {
		return false
	}
	return !strings.ContainsAny(path, `/\`)
}

func isAbsoluteLedgerPath(path string) bool {
	return filepath.IsAbs(path) || isWindowsAbsPath(path)
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
