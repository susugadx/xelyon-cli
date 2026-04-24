package lsp

import (
	"context"
	"encoding/json"
	"time"
)

// handleNotification processes LSP notifications from the server.
func (s *Server) handleNotification(method string, params json.RawMessage) {
	if method == "textDocument/publishDiagnostics" {
		var diagParams PublishDiagnosticsParams
		if err := json.Unmarshal(params, &diagParams); err != nil {
			return
		}

		s.diagMu.Lock()
		s.diagnostics[diagParams.URI] = diagParams.Diagnostics
		s.diagMu.Unlock()
	}
}

// GetLastDiagnostics returns the last received diagnostics for a file.
func (s *Server) GetLastDiagnostics(filePath string) []Diagnostic {
	uri := FileToURI(filePath)
	s.diagMu.RLock()
	defer s.diagMu.RUnlock()
	return s.diagnostics[uri]
}

// ClearDiagnostics clears diagnostics for a file.
func (s *Server) ClearDiagnostics(filePath string) {
	uri := FileToURI(filePath)
	s.diagMu.Lock()
	defer s.diagMu.Unlock()
	delete(s.diagnostics, uri)
}

// WaitForDocumentReady polls for diagnostics readiness with timeout.
// It returns early if diagnostics are already available.
// Never returns an error on timeout - diagnostics may simply not exist for all files.
func (s *Server) WaitForDocumentReady(ctx context.Context, uri string, maxWait time.Duration) {
	// Check if diagnostics already available
	s.diagMu.RLock()
	if _, exists := s.diagnostics[uri]; exists {
		s.diagMu.RUnlock()
		return
	}
	s.diagMu.RUnlock()

	// Initial wait
	initialWait := 10 * time.Millisecond
	timer := time.NewTimer(initialWait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}

	// Poll until timeout
	deadline := time.Now().Add(maxWait - initialWait)
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.diagMu.RLock()
			_, exists := s.diagnostics[uri]
			s.diagMu.RUnlock()
			if exists || time.Now().After(deadline) {
				return
			}
		}
	}
}
