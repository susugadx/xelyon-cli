package ledger

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

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

func validEvidencePointerRange(startLine, endLine int) bool {
	return startLine > 0 && endLine > 0 && endLine >= startLine
}

func evidenceRangeContent(data []byte, startLine, endLine int) (string, bool) {
	lines := splitEvidenceRehydrateLines(data)
	if len(lines) == 0 || startLine > len(lines) || endLine > len(lines) {
		return "", false
	}
	return strings.Join(lines[startLine-1:endLine], "\n"), true
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
