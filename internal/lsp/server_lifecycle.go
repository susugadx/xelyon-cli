package lsp

import (
	"context"
	"time"
)

// OpenDocument sends a textDocument/didOpen notification
func (s *Server) OpenDocument(path, languageID, content string) error {
	uri := FileToURI(path)

	s.mu.Lock()
	if _, ok := s.openDocs[uri]; ok {
		s.mu.Unlock()
		return nil
	}
	s.openDocs[uri] = struct{}{}
	s.mu.Unlock()

	params := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        uri,
			LanguageID: languageID,
			Version:    1,
			Text:       content,
		},
	}
	if err := s.notify("textDocument/didOpen", params); err != nil {
		s.mu.Lock()
		delete(s.openDocs, uri)
		s.mu.Unlock()
		return err
	}
	return nil
}

// CloseDocument sends a textDocument/didClose notification
func (s *Server) CloseDocument(path string) error {
	uri := FileToURI(path)

	s.mu.Lock()
	delete(s.openDocs, uri)
	s.mu.Unlock()

	params := DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{
			URI: uri,
		},
	}
	return s.notify("textDocument/didClose", params)
}

// Close shuts down the LSP server gracefully
func (s *Server) Close() error {
	s.closeOnce.Do(func() {
		close(s.done)

		// Send shutdown request with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Try to send shutdown request (ignore error during cleanup)
		_, _ = s.Call(ctx, "shutdown", nil)

		// Send exit notification (ignore error during cleanup)
		_ = s.notify("exit", nil)

		s.mu.Lock()
		s.cleanupLocked()
		s.mu.Unlock()

		// Wait for process with timeout
		if s.cmd != nil && s.cmd.Process != nil {
			done := make(chan error, 1)
			go func() { done <- s.cmd.Wait() }()

			select {
			case <-done:
			case <-time.After(3 * time.Second):
				_ = s.cmd.Process.Kill()
			}
		}
	})
	return nil
}

// cleanupLocked cleans up resources (must be called with lock held)
func (s *Server) cleanupLocked() {
	if s.stdin != nil {
		s.stdin.Close()
		s.stdin = nil
	}
	s.initialized = false
	s.openDocs = make(map[string]struct{})
}

// IsRunning returns true if the server is initialized and running
func (s *Server) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.initialized
}

// Name returns the server name
func (s *Server) Name() string {
	return s.name
}
