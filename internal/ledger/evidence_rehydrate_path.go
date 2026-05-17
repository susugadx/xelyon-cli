package ledger

import (
	"os"
	"path/filepath"
	"strings"
)

type evidenceRehydrateWorkspace struct {
	repoRoot               string
	repoRootReal           string
	invocationCWD          string
	invocationCWDAvailable bool
}

type resolvedEvidencePointerPath struct {
	relativePath string
	absolutePath string
}

type evidenceRelativePathCandidate struct {
	resolved resolvedEvidencePointerPath
	reason   EvidenceRehydrateErrorReason
	ok       bool
}

func newEvidenceRehydrateWorkspace(opts EvidenceRehydrateOptions) (evidenceRehydrateWorkspace, EvidenceRehydrateResult, error) {
	root := normalizeRepoRoot(opts.RepoRoot)
	if root == "" {
		return evidenceRehydrateWorkspace{}, EvidenceRehydrateResult{Reason: EvidenceRehydrateReasonWorkspaceUnavailable}, newEvidenceRehydrateError(EvidenceRehydrateReasonWorkspaceUnavailable, "", nil)
	}
	rootInfo, err := os.Stat(root)
	if err != nil || !rootInfo.IsDir() {
		return evidenceRehydrateWorkspace{}, EvidenceRehydrateResult{Path: root, Reason: EvidenceRehydrateReasonWorkspaceUnavailable}, newEvidenceRehydrateError(EvidenceRehydrateReasonWorkspaceUnavailable, root, err)
	}
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return evidenceRehydrateWorkspace{}, EvidenceRehydrateResult{Path: root, Reason: EvidenceRehydrateReasonWorkspaceUnavailable}, newEvidenceRehydrateError(EvidenceRehydrateReasonWorkspaceUnavailable, root, err)
	}

	cwd := normalizeRepoRoot(opts.InvocationCWD)
	if cwd == "" {
		cwd = root
	}
	cwdAvailable := false
	if cwdInfo, err := os.Stat(cwd); err == nil && cwdInfo.IsDir() {
		cwdAvailable = true
	}

	return evidenceRehydrateWorkspace{
		repoRoot:               root,
		repoRootReal:           filepath.Clean(rootReal),
		invocationCWD:          cwd,
		invocationCWDAvailable: cwdAvailable,
	}, EvidenceRehydrateResult{}, nil
}

func resolveEvidencePointerPath(pointer EvidencePointer, workspace evidenceRehydrateWorkspace) (resolvedEvidencePointerPath, EvidenceRehydrateResult, error) {
	if reason, ok := invalidRawEvidencePointerPathReason(pointer.Path); ok {
		return evidencePointerPathFailure(pointer.Path, reason, nil)
	}
	candidate := cleanPathCandidate(pointer.Path)
	if reason, ok := invalidEvidencePointerPathReason(candidate); ok {
		return evidencePointerPathFailure(candidate, reason, nil)
	}
	if isWindowsAbsPath(candidate) && !filepath.IsAbs(candidate) {
		return evidencePointerPathFailure(candidate, EvidenceRehydrateReasonPathEscape, nil)
	}
	if filepath.IsAbs(candidate) {
		return resolveAbsoluteEvidencePointerPath(candidate, workspace)
	}
	return resolveRelativeEvidencePointerPath(candidate, pointer.PathBase, workspace)
}

func invalidRawEvidencePointerPathReason(candidate string) (EvidenceRehydrateErrorReason, bool) {
	if strings.Contains(candidate, "\x00") {
		return EvidenceRehydrateReasonInvalidPath, true
	}
	if strings.Contains(candidate, "\n") || strings.Contains(candidate, "\r") {
		return EvidenceRehydrateReasonInvalidPath, true
	}
	return "", false
}

func resolveAbsoluteEvidencePointerPath(candidate string, workspace evidenceRehydrateWorkspace) (resolvedEvidencePointerPath, EvidenceRehydrateResult, error) {
	absolutePath := filepath.Clean(candidate)
	relativePath, ok := relativeEvidencePathWithinWorkspace(workspace, absolutePath)
	if !ok {
		return evidencePointerPathFailure(absolutePath, EvidenceRehydrateReasonPathEscape, nil)
	}
	return resolvedEvidencePointerPath{relativePath: relativePath, absolutePath: absolutePath}, EvidenceRehydrateResult{}, nil
}

func resolveRelativeEvidencePointerPath(candidate string, pathBase EvidencePointerPathBase, workspace evidenceRehydrateWorkspace) (resolvedEvidencePointerPath, EvidenceRehydrateResult, error) {
	switch pathBase {
	case EvidencePointerPathBaseRepoRoot:
		return resolveRepoRelativeEvidencePointerPath(candidate, workspace)
	case EvidencePointerPathBaseAuto:
		return resolveAutoRelativeEvidencePointerPath(candidate, workspace)
	default:
		return evidencePointerPathFailure(candidate, EvidenceRehydrateReasonInvalidPath, nil)
	}
}

func resolveRepoRelativeEvidencePointerPath(candidate string, workspace evidenceRehydrateWorkspace) (resolvedEvidencePointerPath, EvidenceRehydrateResult, error) {
	repoRelativePath, reason, ok := cleanEvidencePointerRelativePath(candidate)
	if !ok {
		return evidencePointerPathFailure(candidate, reason, nil)
	}
	return newRepoRelativeEvidencePointerPath(repoRelativePath, workspace), EvidenceRehydrateResult{}, nil
}

func resolveAutoRelativeEvidencePointerPath(candidate string, workspace evidenceRehydrateWorkspace) (resolvedEvidencePointerPath, EvidenceRehydrateResult, error) {
	repoCandidate := repoRelativeEvidencePointerCandidate(candidate, workspace)
	if !repoCandidate.ok && !repoCandidate.allowsInvocationCWDFallback() {
		return evidencePointerPathFailure(candidate, repoCandidate.reason, nil)
	}
	if repoCandidate.ok && !evidencePathIsMissing(repoCandidate.resolved.absolutePath) {
		return repoCandidate.resolved, EvidenceRehydrateResult{}, nil
	}
	if !workspace.invocationCWDAvailable {
		if repoCandidate.ok {
			return repoCandidate.resolved, EvidenceRehydrateResult{}, nil
		}
		return evidencePointerPathFailure(candidate, repoCandidate.reason, nil)
	}

	cwdCandidate := invocationCWDRelativeEvidencePointerCandidate(candidate, workspace)
	if !cwdCandidate.ok {
		return evidencePointerPathFailure(candidate, EvidenceRehydrateReasonPathEscape, nil)
	}
	if !evidencePathIsMissing(cwdCandidate.resolved.absolutePath) {
		return cwdCandidate.resolved, EvidenceRehydrateResult{}, nil
	}
	if repoCandidate.ok {
		return repoCandidate.resolved, EvidenceRehydrateResult{}, nil
	}
	return cwdCandidate.resolved, EvidenceRehydrateResult{}, nil
}

func repoRelativeEvidencePointerCandidate(candidate string, workspace evidenceRehydrateWorkspace) evidenceRelativePathCandidate {
	repoRelativePath, reason, ok := cleanEvidencePointerRelativePath(candidate)
	if !ok {
		return evidenceRelativePathCandidate{reason: reason}
	}
	return evidenceRelativePathCandidate{resolved: newRepoRelativeEvidencePointerPath(repoRelativePath, workspace), ok: true}
}

func (c evidenceRelativePathCandidate) allowsInvocationCWDFallback() bool {
	return !c.ok && c.reason == EvidenceRehydrateReasonPathEscape
}

func newRepoRelativeEvidencePointerPath(repoRelativePath string, workspace evidenceRehydrateWorkspace) resolvedEvidencePointerPath {
	return resolvedEvidencePointerPath{
		relativePath: repoRelativePath,
		absolutePath: filepath.Clean(filepath.Join(workspace.repoRoot, filepath.FromSlash(repoRelativePath))),
	}
}

func invocationCWDRelativeEvidencePointerCandidate(candidate string, workspace evidenceRehydrateWorkspace) evidenceRelativePathCandidate {
	cwdAbsolutePath := filepath.Clean(filepath.Join(workspace.invocationCWD, filepath.FromSlash(candidate)))
	cwdRelativePath, ok := relativeEvidencePathWithinWorkspace(workspace, cwdAbsolutePath)
	if !ok {
		return evidenceRelativePathCandidate{reason: EvidenceRehydrateReasonPathEscape}
	}
	return evidenceRelativePathCandidate{
		resolved: resolvedEvidencePointerPath{relativePath: cwdRelativePath, absolutePath: cwdAbsolutePath},
		ok:       true,
	}
}

func invalidEvidencePointerPathReason(candidate string) (EvidenceRehydrateErrorReason, bool) {
	if !isLedgerPathCandidateSafe(candidate) {
		return EvidenceRehydrateReasonInvalidPath, true
	}
	return "", false
}

func cleanEvidencePointerRelativePath(candidate string) (string, EvidenceRehydrateErrorReason, bool) {
	cleaned := filepath.Clean(filepath.FromSlash(candidate))
	if cleaned == "." || cleaned == "" {
		return "", EvidenceRehydrateReasonInvalidPath, false
	}
	if filepath.IsAbs(cleaned) || isWindowsAbsPath(cleaned) {
		return "", EvidenceRehydrateReasonPathEscape, false
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", EvidenceRehydrateReasonPathEscape, false
	}
	return filepath.ToSlash(cleaned), "", true
}

func relativeEvidencePathWithinWorkspace(workspace evidenceRehydrateWorkspace, absolutePath string) (string, bool) {
	absolutePath = filepath.Clean(absolutePath)
	if relativePath, ok := relativeEvidencePathWithinRoot(workspace.repoRoot, absolutePath); ok {
		return relativePath, true
	}
	if relativePath, ok := relativeEvidencePathWithinRoot(workspace.repoRootReal, absolutePath); ok {
		return relativePath, true
	}
	if evaluatedPath, ok := evalLedgerPathBestEffort(absolutePath); ok {
		return relativeEvidencePathWithinRoot(workspace.repoRootReal, evaluatedPath)
	}
	return "", false
}

func relativeEvidencePathWithinRoot(root, absolutePath string) (string, bool) {
	if root == "" || absolutePath == "" {
		return "", false
	}
	relativePath, err := filepath.Rel(root, absolutePath)
	if err != nil {
		return "", false
	}
	if relativePath == "." {
		return ".", true
	}
	return cleanLedgerRelativePath(relativePath)
}

func evidencePathIsMissing(path string) bool {
	_, err := os.Lstat(path)
	return os.IsNotExist(err)
}

func evidencePointerPathFailure(path string, reason EvidenceRehydrateErrorReason, err error) (resolvedEvidencePointerPath, EvidenceRehydrateResult, error) {
	return resolvedEvidencePointerPath{}, EvidenceRehydrateResult{Path: path, Reason: reason}, newEvidenceRehydrateError(reason, path, err)
}
