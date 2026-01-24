package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// ServerConfig holds configuration for an LSP server
type ServerConfig struct {
	Command  string
	Args     []string
	Disabled bool
}

// Client manages multiple LSP servers (one per language)
type Client struct {
	mu      sync.RWMutex
	servers map[string]*Server      // language key -> server
	configs map[string]ServerConfig // language key -> config
	rootURI string
}

// NewClient creates a new LSP client
func NewClient(rootPath string) *Client {
	return &Client{
		servers: make(map[string]*Server),
		configs: make(map[string]ServerConfig),
		rootURI: FileToURI(rootPath),
	}
}

// SetConfigs sets the server configurations
func (c *Client) SetConfigs(configs map[string]ServerConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.configs = configs
}

// GetServer returns the server for a language, starting it lazily if needed
func (c *Client) GetServer(ctx context.Context, language string) (*Server, error) {
	// Map language to server key
	serverKey := LanguageServerKey(language)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if server already exists and is running
	if server, ok := c.servers[serverKey]; ok {
		if server.IsRunning() {
			return server, nil
		}
		// Server died, clean up
		server.Close()
		delete(c.servers, serverKey)
	}

	// Get config for language
	config, ok := c.configs[serverKey]
	if !ok {
		return nil, fmt.Errorf("no LSP server configured for language: %s", language)
	}

	if config.Disabled {
		return nil, fmt.Errorf("LSP server for %s is disabled", language)
	}

	// Check if command exists
	if _, err := exec.LookPath(config.Command); err != nil {
		return nil, fmt.Errorf("LSP server command not found: %s (install it to use LSP features)", config.Command)
	}

	// Create and start server
	server := NewServer(serverKey)

	// Use timeout context for startup
	startCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := server.Start(startCtx, config.Command, config.Args, c.rootURI); err != nil {
		return nil, fmt.Errorf("failed to start LSP server for %s: %w", language, err)
	}

	c.servers[serverKey] = server
	fmt.Printf("🔌 LSP server '%s' started for %s\n", config.Command, language)
	return server, nil
}

// GetServerForFile returns the appropriate server for a file path
func (c *Client) GetServerForFile(ctx context.Context, filePath string) (*Server, error) {
	language := DetectLanguage(filePath)
	if language == "" {
		return nil, fmt.Errorf("could not detect language for file: %s", filePath)
	}
	return c.GetServer(ctx, language)
}

// Close shuts down all LSP servers
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for lang, server := range c.servers {
		if err := server.Close(); err != nil {
			fmt.Printf("Warning: failed to close LSP server for %s: %v\n", lang, err)
		}
	}
	c.servers = make(map[string]*Server)
}

// Status returns the status of all configured servers
func (c *Client) Status() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := make(map[string]string)
	for lang, config := range c.configs {
		if config.Disabled {
			status[lang] = "disabled"
			continue
		}

		if server, ok := c.servers[lang]; ok && server.IsRunning() {
			status[lang] = "running"
		} else {
			// Check if command exists
			if _, err := exec.LookPath(config.Command); err != nil {
				status[lang] = fmt.Sprintf("not installed (%s)", config.Command)
			} else {
				status[lang] = "not started (lazy)"
			}
		}
	}
	return status
}

// ===== LSP Operations =====

// FindReferences finds all references to a symbol
func (c *Client) FindReferences(ctx context.Context, filePath string, line, character int, includeDeclaration bool) ([]Location, error) {
	server, err := c.GetServerForFile(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Read file content and open document
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	language := DetectLanguage(filePath)
	if err := server.OpenDocument(filePath, language, string(content)); err != nil {
		return nil, fmt.Errorf("failed to open document: %w", err)
	}

	// Wait for server to process the document
	server.WaitForDocumentReady(ctx, FileToURI(filePath), 100*time.Millisecond)

	params := ReferenceParams{
		TextDocument: TextDocumentIdentifier{URI: FileToURI(filePath)},
		Position:     Position{Line: line - 1, Character: character - 1}, // Convert to 0-indexed
		Context:      ReferenceContext{IncludeDeclaration: includeDeclaration},
	}

	resp, err := server.Call(ctx, "textDocument/references", params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("LSP error: %s", resp.Error.Message)
	}

	if resp.Result == nil || string(resp.Result) == "null" {
		return []Location{}, nil
	}

	var locations []Location
	if err := json.Unmarshal(resp.Result, &locations); err != nil {
		return nil, fmt.Errorf("failed to parse references result: %w", err)
	}

	return locations, nil
}

// GoToDefinition finds the definition of a symbol
func (c *Client) GoToDefinition(ctx context.Context, filePath string, line, character int) ([]Location, error) {
	server, err := c.GetServerForFile(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Read file content and open document
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	language := DetectLanguage(filePath)
	if err := server.OpenDocument(filePath, language, string(content)); err != nil {
		return nil, fmt.Errorf("failed to open document: %w", err)
	}

	// Wait for server to process the document
	server.WaitForDocumentReady(ctx, FileToURI(filePath), 100*time.Millisecond)

	params := TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: FileToURI(filePath)},
		Position:     Position{Line: line - 1, Character: character - 1},
	}

	resp, err := server.Call(ctx, "textDocument/definition", params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("LSP error: %s", resp.Error.Message)
	}

	if resp.Result == nil || string(resp.Result) == "null" {
		return []Location{}, nil
	}

	// Result can be Location, []Location, or LocationLink[]
	// Try to parse as array first
	var locations []Location
	if err := json.Unmarshal(resp.Result, &locations); err != nil {
		// Try single Location
		var single Location
		if err := json.Unmarshal(resp.Result, &single); err != nil {
			return nil, fmt.Errorf("failed to parse definition result: %w", err)
		}
		locations = []Location{single}
	}

	return locations, nil
}

// GetHover gets hover information for a symbol
func (c *Client) GetHover(ctx context.Context, filePath string, line, character int) (*HoverResult, error) {
	server, err := c.GetServerForFile(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Read file content and open document
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	language := DetectLanguage(filePath)
	if err := server.OpenDocument(filePath, language, string(content)); err != nil {
		return nil, fmt.Errorf("failed to open document: %w", err)
	}

	// Wait for server to process the document
	server.WaitForDocumentReady(ctx, FileToURI(filePath), 100*time.Millisecond)

	params := HoverParams{
		TextDocument: TextDocumentIdentifier{URI: FileToURI(filePath)},
		Position:     Position{Line: line - 1, Character: character - 1},
	}

	resp, err := server.Call(ctx, "textDocument/hover", params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("LSP error: %s", resp.Error.Message)
	}

	if resp.Result == nil || string(resp.Result) == "null" {
		return nil, nil // No hover info available
	}

	var hover HoverResult
	if err := json.Unmarshal(resp.Result, &hover); err != nil {
		// Some servers return contents as a string directly
		var contents struct {
			Contents interface{} `json:"contents"`
			Range    *Range      `json:"range,omitempty"`
		}
		if err := json.Unmarshal(resp.Result, &contents); err != nil {
			return nil, fmt.Errorf("failed to parse hover result: %w", err)
		}

		// Handle different content formats
		switch v := contents.Contents.(type) {
		case string:
			hover.Contents = MarkupContent{Kind: "plaintext", Value: v}
		case map[string]interface{}:
			if kind, ok := v["kind"].(string); ok {
				if value, ok := v["value"].(string); ok {
					hover.Contents = MarkupContent{Kind: kind, Value: value}
				}
			} else if language, ok := v["language"].(string); ok {
				// MarkedString format: {language: "go", value: "..."}
				if value, ok := v["value"].(string); ok {
					hover.Contents = MarkupContent{Kind: "markdown", Value: fmt.Sprintf("```%s\n%s\n```", language, value)}
				}
			}
		case []interface{}:
			// Array of MarkedString
			var sb strings.Builder
			for _, item := range v {
				switch it := item.(type) {
				case string:
					sb.WriteString(it)
					sb.WriteString("\n")
				case map[string]interface{}:
					if language, ok := it["language"].(string); ok {
						if value, ok := it["value"].(string); ok {
							sb.WriteString(fmt.Sprintf("```%s\n%s\n```\n", language, value))
						}
					}
				}
			}
			hover.Contents = MarkupContent{Kind: "markdown", Value: sb.String()}
		}
		hover.Range = contents.Range
	}

	return &hover, nil
}

// GetDiagnostics gets diagnostics (errors, warnings) for a file
func (c *Client) GetDiagnostics(ctx context.Context, filePath string) ([]Diagnostic, error) {
	server, err := c.GetServerForFile(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Read file content and open document
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	language := DetectLanguage(filePath)
	if err := server.OpenDocument(filePath, language, string(content)); err != nil {
		return nil, fmt.Errorf("failed to open document: %w", err)
	}

	// Wait for the server to analyze and publish diagnostics
	server.WaitForDocumentReady(ctx, FileToURI(filePath), 500*time.Millisecond)

	// Get diagnostics from server's cache
	return server.GetLastDiagnostics(filePath), nil
}

// Rename renames a symbol at the given position to a new name
func (c *Client) Rename(ctx context.Context, filePath string, line, character int, newName string) (*WorkspaceEdit, error) {
	server, err := c.GetServerForFile(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Read file content and open document
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	language := DetectLanguage(filePath)
	if err := server.OpenDocument(filePath, language, string(content)); err != nil {
		return nil, fmt.Errorf("failed to open document: %w", err)
	}

	// Wait for server to process the document
	server.WaitForDocumentReady(ctx, FileToURI(filePath), 100*time.Millisecond)

	params := RenameParams{
		TextDocument: TextDocumentIdentifier{URI: FileToURI(filePath)},
		Position:     Position{Line: line - 1, Character: character - 1},
		NewName:      newName,
	}

	resp, err := server.Call(ctx, "textDocument/rename", params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("LSP error: %s", resp.Error.Message)
	}

	if resp.Result == nil || string(resp.Result) == "null" {
		return nil, fmt.Errorf("rename not supported or symbol not found")
	}

	var edit WorkspaceEdit
	if err := json.Unmarshal(resp.Result, &edit); err != nil {
		return nil, fmt.Errorf("failed to parse rename result: %w", err)
	}

	return &edit, nil
}
