package search

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

type mockJSFamilyLSPClient struct {
	refs          []navigation.LSPLocation
	err           error
	requestedFile string
	requestedLine int
	requestedChar int
}

func (m *mockJSFamilyLSPClient) FindReferences(_ context.Context, filePath string, line, character int, _ bool) ([]navigation.LSPLocation, error) {
	m.requestedFile = filePath
	m.requestedLine = line
	m.requestedChar = character
	if m.err != nil {
		return nil, m.err
	}
	return m.refs, nil
}

func (m *mockJSFamilyLSPClient) GotoDefinition(context.Context, string, int, int) ([]navigation.LSPLocation, error) {
	return nil, nil
}

func (m *mockJSFamilyLSPClient) GotoImplementation(context.Context, string, int, int) ([]navigation.LSPLocation, error) {
	return nil, nil
}

type mockJSFamilyRawLSPClient struct {
	rootDir string
	refs    []lsp.Location
	err     error
}

func (m *mockJSFamilyRawLSPClient) FindReferences(context.Context, string, int, int, bool) ([]navigation.LSPLocation, error) {
	if m.err != nil {
		return nil, m.err
	}
	return navigation.ProtocolLocationsToLSPLocations(m.refs, m.rootDir), nil
}

func (m *mockJSFamilyRawLSPClient) GotoDefinition(context.Context, string, int, int) ([]navigation.LSPLocation, error) {
	return nil, nil
}

func (m *mockJSFamilyRawLSPClient) GotoImplementation(context.Context, string, int, int) ([]navigation.LSPLocation, error) {
	return nil, nil
}

func rawJSFamilyLSPLocationForToken(file string, line int, lineText string, token string) lsp.Location {
	start, end := testLSPRangeForSearchToken(lineText, token)
	return rawJSFamilyLSPLocation(file, line, start, end)
}

func rawJSFamilyLSPLocation(file string, line int, startCharacter int, endCharacter int) lsp.Location {
	return lsp.Location{
		URI: lsp.FileToURI(file),
		Range: lsp.Range{
			Start: lsp.Position{Line: line - 1, Character: startCharacter - 1},
			End:   lsp.Position{Line: line - 1, Character: endCharacter - 1},
		},
	}
}
