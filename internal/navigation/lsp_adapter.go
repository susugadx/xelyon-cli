package navigation

import (
	"context"
	"path/filepath"

	"github.com/susugadx/xelyon-cli/internal/lsp"
)

// LSPAdapter adapts the concrete lsp.Client to the navigation interface.
type LSPAdapter struct {
	Client  *lsp.Client
	RootDir string
}

// NewLSPAdapter returns an adapter for the provided client.
func NewLSPAdapter(client *lsp.Client, rootDir string) *LSPAdapter {
	if client == nil {
		return nil
	}
	return &LSPAdapter{Client: client, RootDir: rootDir}
}

// FindReferences delegates to the concrete LSP client and converts locations.
func (a *LSPAdapter) FindReferences(ctx context.Context, filePath string, line, character int, includeDeclaration bool) ([]LSPLocation, error) {
	locations, err := a.Client.FindReferences(ctx, a.toAbsPath(filePath), line, character, includeDeclaration)
	if err != nil {
		return nil, err
	}
	return a.convertLocations(locations), nil
}

// GotoDefinition delegates to the concrete LSP client and converts locations.
func (a *LSPAdapter) GotoDefinition(ctx context.Context, filePath string, line, character int) ([]LSPLocation, error) {
	locations, err := a.Client.GotoDefinition(ctx, a.toAbsPath(filePath), line, character)
	if err != nil {
		return nil, err
	}
	return a.convertLocations(locations), nil
}

// GotoImplementation delegates to the concrete LSP client and converts locations.
func (a *LSPAdapter) GotoImplementation(ctx context.Context, filePath string, line, character int) ([]LSPLocation, error) {
	locations, err := a.Client.GotoImplementation(ctx, a.toAbsPath(filePath), line, character)
	if err != nil {
		return nil, err
	}
	return a.convertLocations(locations), nil
}

func (a *LSPAdapter) toAbsPath(path string) string {
	if filepath.IsAbs(path) || a.RootDir == "" {
		return path
	}
	return filepath.Join(a.RootDir, path)
}

func (a *LSPAdapter) convertLocations(locations []lsp.Location) []LSPLocation {
	return ProtocolLocationsToLSPLocations(locations, a.RootDir)
}

// ProtocolLocationsToLSPLocations は raw LSP protocol location を navigation の LSPLocation に変換する。
func ProtocolLocationsToLSPLocations(locations []lsp.Location, rootDir string) []LSPLocation {
	result := make([]LSPLocation, 0, len(locations))
	for _, loc := range locations {
		file := lsp.URIToFile(loc.URI)
		if rootDir != "" {
			if rel, err := filepath.Rel(rootDir, file); err == nil {
				file = rel
			}
		}
		result = append(result, LSPLocation{
			File:      file,
			Line:      loc.Range.Start.Line + 1,
			Character: loc.Range.Start.Character + 1,
			EndLine:   loc.Range.End.Line + 1,
			EndChar:   loc.Range.End.Character + 1,
		})
	}
	return result
}
