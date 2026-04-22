package navigation

import (
	"path/filepath"

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
	relPath := toRelativePath(fileAbsPath)
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
