package navigation

import "context"

// LSPLocation describes a location returned by the LSP layer.
type LSPLocation struct {
	File      string
	Line      int
	Character int
	EndLine   int
	EndChar   int
}

// LSPClient is the minimal interface navigation needs for Go symbol resolution.
// A nil client means navigation should fall back to the existing ripgrep path.
type LSPClient interface {
	FindReferences(ctx context.Context, filePath string, line, character int, includeDeclaration bool) ([]LSPLocation, error)
	GotoDefinition(ctx context.Context, filePath string, line, character int) ([]LSPLocation, error)
	GotoImplementation(ctx context.Context, filePath string, line, character int) ([]LSPLocation, error)
}
