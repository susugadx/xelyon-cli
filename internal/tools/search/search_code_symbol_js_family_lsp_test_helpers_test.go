package search

import (
	"context"

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
