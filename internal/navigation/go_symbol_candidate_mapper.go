package navigation

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

func newSymbolCandidateFromSnapshotEntry(entry goSymbolSnapshotEntry, rootPath string) SymbolCandidate {
	return SymbolCandidate{
		Name:               entry.Name,
		Kind:               entry.Kind,
		File:               entry.File,
		Line:               entry.Line,
		EndLine:            entry.EndLine,
		Receiver:           entry.Receiver,
		ReceiverNorm:       entry.ReceiverNorm,
		Signature:          entry.Signature,
		Exported:           entry.Exported,
		PackageDir:         entry.PackageDir,
		StableKey:          entry.StableKey,
		StableKeyCollision: entry.Collision,
		RootPath:           rootPath,
	}
}

func newSymbolCandidateFromASTSymbol(symbol ast.Symbol, fileAbsPath, rootPath string) SymbolCandidate {
	receiver := extractMethodReceiver(symbol.Signature)
	relPath := symbolCandidateFilePath(fileAbsPath, rootPath)
	receiverNorm := canonicalReceiver(receiver)
	packageDir := filepath.Dir(relPath)
	return SymbolCandidate{
		Name:         symbol.Name,
		Kind:         string(symbol.Kind),
		File:         relPath,
		Line:         symbol.Line,
		EndLine:      symbol.EndLine,
		Receiver:     receiver,
		ReceiverNorm: receiverNorm,
		Signature:    symbol.Signature,
		Exported:     symbol.Exported,
		PackageDir:   packageDir,
		StableKey:    stableGoSymbolKey(packageDir, receiverNorm, symbol.Name, string(symbol.Kind), symbol.Signature),
		RootPath:     rootPath,
	}
}

func symbolCandidateFilePath(filePath, rootPath string) string {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return ""
	}
	absPath := filePath
	if !filepath.IsAbs(absPath) {
		if resolved, err := filepath.Abs(absPath); err == nil {
			absPath = resolved
		}
	}

	rootPath = normalizeNavigationRootPath(rootPath)
	if rootPath != "" {
		if relPath, err := filepath.Rel(rootPath, absPath); err == nil {
			return filepath.ToSlash(filepath.Clean(relPath))
		}
	}
	return filepath.ToSlash(filepath.Clean(toRelativePath(absPath)))
}
