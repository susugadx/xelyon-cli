package ledger

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EvidencePointer は現在ファイルから再読込できる evidence の位置情報。
type EvidencePointer struct {
	Path       string
	StartLine  int
	EndLine    int
	Source     string
	ToolCallID string
	FileHash   string
	Stale      bool
	PathBase   EvidencePointerPathBase
}

// EvidenceRehydrateOptions は evidence pointer の再読込に使う workspace 情報。
type EvidenceRehydrateOptions struct {
	RepoRoot      string
	InvocationCWD string
}

// EvidenceRehydrateResult は evidence pointer の現在ファイル再読込結果。
type EvidenceRehydrateResult struct {
	Path            string
	StartLine       int
	EndLine         int
	Content         string
	CurrentFileHash string
	Stale           bool
	Reason          EvidenceRehydrateErrorReason
}

// EvidenceRehydrateErrorReason は evidence rehydrate が失敗した理由。
type EvidenceRehydrateErrorReason string

const (
	EvidenceRehydrateReasonWorkspaceUnavailable EvidenceRehydrateErrorReason = "workspace_unavailable"
	EvidenceRehydrateReasonInvalidPath          EvidenceRehydrateErrorReason = "invalid_path"
	EvidenceRehydrateReasonPathEscape           EvidenceRehydrateErrorReason = "path_escape"
	EvidenceRehydrateReasonMissingFile          EvidenceRehydrateErrorReason = "missing_file"
	EvidenceRehydrateReasonNotRegularFile       EvidenceRehydrateErrorReason = "not_regular_file"
	EvidenceRehydrateReasonUnreadableFile       EvidenceRehydrateErrorReason = "unreadable_file"
	EvidenceRehydrateReasonBinaryFile           EvidenceRehydrateErrorReason = "binary_file"
	EvidenceRehydrateReasonInvalidRange         EvidenceRehydrateErrorReason = "invalid_range"
	EvidenceRehydrateReasonRangeOutOfBounds     EvidenceRehydrateErrorReason = "range_out_of_bounds"
	EvidenceRehydrateReasonContextCancelled     EvidenceRehydrateErrorReason = "context_cancelled"
)

// EvidenceRehydrateError は evidence rehydrate の structured error。
type EvidenceRehydrateError struct {
	Reason EvidenceRehydrateErrorReason
	Path   string
	Err    error
}

// EvidencePointerPathBase は relative Path の解決基準を表す。
type EvidencePointerPathBase string

const (
	// EvidencePointerPathBaseAuto は repo root を先に試し、必要なら invocation cwd を fallback に使う。
	EvidencePointerPathBaseAuto EvidencePointerPathBase = ""
	// EvidencePointerPathBaseRepoRoot は Path を repo-relative としてだけ解決する。
	EvidencePointerPathBaseRepoRoot EvidencePointerPathBase = "repo_root"
)

func (e *EvidenceRehydrateError) Error() string {
	if e == nil {
		return "rehydrate evidence pointer: unknown error"
	}
	message := fmt.Sprintf("rehydrate evidence pointer: %s", e.Reason)
	if e.Path != "" {
		message += ": " + e.Path
	}
	if e.Err != nil {
		message += ": " + e.Err.Error()
	}
	return message
}

func (e *EvidenceRehydrateError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// RehydrateEvidencePointer は evidence pointer の対象 range を現在ファイルから読み直す。
func RehydrateEvidencePointer(ctx context.Context, pointer EvidencePointer, opts EvidenceRehydrateOptions) (EvidenceRehydrateResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return evidenceRehydrateFailure(EvidenceRehydrateResult{Path: pointer.Path}, EvidenceRehydrateReasonContextCancelled, err)
	}

	workspace, result, err := newEvidenceRehydrateWorkspace(opts)
	if err != nil {
		return result, err
	}
	resolved, result, err := resolveEvidencePointerPath(pointer, workspace)
	if err != nil {
		return result, err
	}

	return rehydrateEvidenceResolvedPointer(ctx, pointer, workspace, resolved)
}

func rehydrateEvidenceResolvedPointer(ctx context.Context, pointer EvidencePointer, workspace evidenceRehydrateWorkspace, resolved resolvedEvidencePointerPath) (EvidenceRehydrateResult, error) {
	result := EvidenceRehydrateResult{
		Path:      resolved.relativePath,
		StartLine: pointer.StartLine,
		EndLine:   pointer.EndLine,
	}
	if !validEvidencePointerRange(pointer.StartLine, pointer.EndLine) {
		return evidenceRehydrateFailure(result, EvidenceRehydrateReasonInvalidRange, nil)
	}

	data, result, err := readEvidenceResolvedFile(ctx, pointer, workspace, resolved, result)
	if err != nil {
		return result, err
	}
	if bytes.Contains(data, []byte{0}) {
		return evidenceRehydrateFailure(result, EvidenceRehydrateReasonBinaryFile, nil)
	}
	content, ok := evidenceRangeContent(data, pointer.StartLine, pointer.EndLine)
	if !ok {
		return evidenceRehydrateFailure(result, EvidenceRehydrateReasonRangeOutOfBounds, nil)
	}
	result.Content = content
	if err := ctx.Err(); err != nil {
		return evidenceRehydrateFailure(result, EvidenceRehydrateReasonContextCancelled, err)
	}
	return result, nil
}

func readEvidenceResolvedFile(ctx context.Context, pointer EvidencePointer, workspace evidenceRehydrateWorkspace, resolved resolvedEvidencePointerPath, result EvidenceRehydrateResult) ([]byte, EvidenceRehydrateResult, error) {
	if err := ctx.Err(); err != nil {
		return evidenceRehydrateReadFailure(result, EvidenceRehydrateReasonContextCancelled, err)
	}
	info, statErr := os.Stat(resolved.absolutePath)
	if statErr != nil {
		reason := EvidenceRehydrateReasonUnreadableFile
		if os.IsNotExist(statErr) {
			reason = EvidenceRehydrateReasonMissingFile
		}
		return evidenceRehydrateReadFailure(result, reason, statErr)
	}
	evaluatedPath, evalErr := filepath.EvalSymlinks(resolved.absolutePath)
	if evalErr != nil {
		reason := EvidenceRehydrateReasonUnreadableFile
		if os.IsNotExist(evalErr) {
			reason = EvidenceRehydrateReasonMissingFile
		}
		return evidenceRehydrateReadFailure(result, reason, evalErr)
	}
	if !pathIsWithinRepoRoot(workspace.repoRootReal, filepath.Clean(evaluatedPath)) {
		return evidenceRehydrateReadFailure(result, EvidenceRehydrateReasonPathEscape, nil)
	}
	if !info.Mode().IsRegular() {
		return evidenceRehydrateReadFailure(result, EvidenceRehydrateReasonNotRegularFile, nil)
	}

	if err := ctx.Err(); err != nil {
		return evidenceRehydrateReadFailure(result, EvidenceRehydrateReasonContextCancelled, err)
	}
	data, readErr := os.ReadFile(resolved.absolutePath)
	if readErr != nil {
		return evidenceRehydrateReadFailure(result, EvidenceRehydrateReasonUnreadableFile, readErr)
	}
	result.CurrentFileHash = evidenceFileHash(data)
	result.Stale = pointer.FileHash != "" && pointer.FileHash != result.CurrentFileHash
	return data, result, nil
}

func evidenceRehydrateReadFailure(result EvidenceRehydrateResult, reason EvidenceRehydrateErrorReason, err error) ([]byte, EvidenceRehydrateResult, error) {
	result.Reason = reason
	return nil, result, newEvidenceRehydrateError(reason, result.Path, err)
}

func evidenceRangeContent(data []byte, startLine, endLine int) (string, bool) {
	lines := splitEvidenceRehydrateLines(data)
	if len(lines) == 0 || startLine > len(lines) || endLine > len(lines) {
		return "", false
	}
	return strings.Join(lines[startLine-1:endLine], "\n"), true
}

// RehydrateEvidencePointer は Store の workspace 情報で evidence pointer を再読込する。
func (s *Store) RehydrateEvidencePointer(ctx context.Context, pointer EvidencePointer) (EvidenceRehydrateResult, error) {
	if s == nil {
		return evidenceRehydrateFailure(EvidenceRehydrateResult{Path: pointer.Path}, EvidenceRehydrateReasonWorkspaceUnavailable, nil)
	}
	return RehydrateEvidencePointer(ctx, pointer, EvidenceRehydrateOptions{
		RepoRoot:      s.repoRoot,
		InvocationCWD: s.invocationCWD,
	})
}

func evidencePointersFromFacts(facts []evidenceFact) []EvidencePointer {
	if len(facts) == 0 {
		return nil
	}
	pointers := make([]EvidencePointer, 0, len(facts))
	for _, fact := range facts {
		pointers = append(pointers, evidencePointerFromFact(fact))
	}
	return pointers
}

func evidencePointerFromFact(fact evidenceFact) EvidencePointer {
	return EvidencePointer{
		Path:       fact.path,
		StartLine:  fact.startLine,
		EndLine:    fact.endLine,
		Source:     fact.source,
		ToolCallID: fact.toolCallID,
		FileHash:   fact.fileHash,
		Stale:      fact.stale,
		PathBase:   EvidencePointerPathBaseRepoRoot,
	}
}

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

func validEvidencePointerRange(startLine, endLine int) bool {
	return startLine > 0 && endLine > 0 && endLine >= startLine
}

func splitEvidenceRehydrateLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	lines := make([]string, 0, bytes.Count(data, []byte{'\n'})+1)
	start := 0
	for i, b := range data {
		if b != '\n' {
			continue
		}
		lines = append(lines, evidenceRehydrateLineText(data[start:i]))
		start = i + 1
	}
	if start < len(data) {
		lines = append(lines, evidenceRehydrateLineText(data[start:]))
	}
	return lines
}

func evidenceRehydrateLineText(line []byte) string {
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	return string(line)
}

func evidenceFileHash(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func evidenceRehydrateFailure(result EvidenceRehydrateResult, reason EvidenceRehydrateErrorReason, err error) (EvidenceRehydrateResult, error) {
	result.Reason = reason
	return result, newEvidenceRehydrateError(reason, result.Path, err)
}

func newEvidenceRehydrateError(reason EvidenceRehydrateErrorReason, path string, err error) *EvidenceRehydrateError {
	return &EvidenceRehydrateError{
		Reason: reason,
		Path:   path,
		Err:    err,
	}
}
