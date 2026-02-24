package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Server manages a single LSP server process
type Server struct {
	name    string
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Reader
	rootURI string

	mu          sync.Mutex
	requestID   atomic.Int64
	pending     map[int]chan *Response
	initialized bool

	// Diagnostics storage (URI -> diagnostics)
	diagMu      sync.RWMutex
	diagnostics map[string][]Diagnostic

	// Graceful shutdown
	done      chan struct{}
	closeOnce sync.Once
}

// NewServer creates a new LSP server instance (does not start yet)
func NewServer(name string) *Server {
	return &Server{
		name:        name,
		pending:     make(map[int]chan *Response),
		diagnostics: make(map[string][]Diagnostic),
		done:        make(chan struct{}),
	}
}

// Start launches the LSP server process and initializes it
func (s *Server) Start(ctx context.Context, command string, args []string, rootURI string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cmd != nil && s.initialized {
		return nil // Already started
	}

	s.rootURI = rootURI
	// Don't use CommandContext - we manage the process lifecycle ourselves
	// CommandContext would kill the process when ctx is cancelled, which we don't want
	s.cmd = exec.Command(command, args...)

	// Capture stderr for debugging
	stderrPipe, stderrErr := s.cmd.StderrPipe()
	if stderrErr != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", stderrErr)
	}

	var err error
	s.stdin, err = s.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	s.stdout = bufio.NewReader(stdout)

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start LSP server '%s': %w", command, err)
	}

	// Start stderr reader goroutine (for debugging)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stderrPipe.Read(buf)
			if err != nil {
				return
			}
			if n > 0 && os.Getenv("XELYON_DEBUG_LSP") == "1" {
				fmt.Fprintf(os.Stderr, "[LSP %s stderr] %s", s.name, string(buf[:n]))
			}
		}
	}()

	// Start response reader goroutine
	go s.readResponses()

	// Initialize the server (must be done outside the lock)
	s.mu.Unlock()
	err = s.initialize(ctx)
	s.mu.Lock()

	if err != nil {
		s.cleanupLocked()
		return fmt.Errorf("failed to initialize LSP server: %w", err)
	}

	s.initialized = true
	return nil
}

// initialize sends the LSP initialize request and waits for response
func (s *Server) initialize(ctx context.Context) error {
	params := InitializeParams{
		ProcessID: os.Getpid(),
		RootURI:   s.rootURI,
		Capabilities: ClientCapabilities{
			TextDocument: TextDocumentClientCapabilities{
				References: &ReferencesCapability{DynamicRegistration: false},
			},
		},
	}

	// Send initialize request with timeout
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := s.Call(initCtx, "initialize", params)
	if err != nil {
		return fmt.Errorf("initialize request failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("initialize error: %s", resp.Error.Message)
	}

	// Send initialized notification
	_ = s.notify("initialized", struct{}{})

	return nil
}

// readResponses reads responses from the LSP server in a loop
func (s *Server) readResponses() {
	debug := os.Getenv("XELYON_DEBUG_LSP") == "1"
	for {
		select {
		case <-s.done:
			return
		default:
			// Read Content-Length header
			contentLength, err := s.readHeader()
			if err != nil {
				if debug {
					fmt.Fprintf(os.Stderr, "[LSP %s] readHeader error: %v\n", s.name, err)
				}
				return
			}

			if contentLength == 0 {
				continue
			}

			// Read body
			body := make([]byte, contentLength)
			if _, err := io.ReadFull(s.stdout, body); err != nil {
				if debug {
					fmt.Fprintf(os.Stderr, "[LSP %s] readBody error: %v\n", s.name, err)
				}
				return
			}

			// Try to parse as response (has ID)
			var msg struct {
				ID     *int            `json:"id"`
				Method string          `json:"method"`
				Params json.RawMessage `json:"params"`
				Result json.RawMessage `json:"result"`
				Error  *ResponseError  `json:"error"`
			}

			if err := json.Unmarshal(body, &msg); err != nil {
				continue
			}

			// If it has an ID, it's a response to our request
			if msg.ID != nil {
				resp := &Response{
					JSONRPC: "2.0",
					ID:      *msg.ID,
					Result:  msg.Result,
					Error:   msg.Error,
				}

				s.mu.Lock()
				if ch, ok := s.pending[*msg.ID]; ok {
					ch <- resp
					delete(s.pending, *msg.ID)
				}
				s.mu.Unlock()
			} else {
				// Handle notifications
				s.handleNotification(msg.Method, msg.Params)
			}
		}
	}
}

// readHeader reads the Content-Length header from the LSP stream
func (s *Server) readHeader() (int, error) {
	var contentLength int

	for {
		line, err := s.stdout.ReadString('\n')
		if err != nil {
			return 0, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			// Empty line marks end of headers
			break
		}

		if strings.HasPrefix(line, "Content-Length:") {
			lenStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			contentLength, _ = strconv.Atoi(lenStr)
		}
		// Ignore other headers like Content-Type
	}

	return contentLength, nil
}

// Call sends a JSON-RPC request and waits for the response
func (s *Server) Call(ctx context.Context, method string, params interface{}) (*Response, error) {
	id := int(s.requestID.Add(1))

	// Create response channel
	respCh := make(chan *Response, 1)
	s.mu.Lock()
	s.pending[id] = respCh
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
	}()

	// Send request
	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if err := s.send(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Wait for response or timeout
	select {
	case resp := <-respCh:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// notify sends a JSON-RPC notification (no response expected)
func (s *Server) notify(method string, params interface{}) error {
	notif := Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	return s.send(notif)
}

// send sends a JSON-RPC message to the LSP server
func (s *Server) send(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// LSP uses Content-Length header
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stdin == nil {
		return fmt.Errorf("server not started")
	}

	// Check if process is still alive
	if s.cmd != nil && s.cmd.ProcessState != nil && s.cmd.ProcessState.Exited() {
		return fmt.Errorf("LSP server process has exited")
	}

	if _, err := s.stdin.Write([]byte(header)); err != nil {
		if os.Getenv("XELYON_DEBUG_LSP") == "1" {
			fmt.Fprintf(os.Stderr, "[LSP %s] write header error: %v\n", s.name, err)
		}
		return err
	}
	if _, err := s.stdin.Write(data); err != nil {
		if os.Getenv("XELYON_DEBUG_LSP") == "1" {
			fmt.Fprintf(os.Stderr, "[LSP %s] write body error: %v\n", s.name, err)
		}
		return err
	}
	return nil
}

// OpenDocument sends a textDocument/didOpen notification
func (s *Server) OpenDocument(path, languageID, content string) error {
	params := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        FileToURI(path),
			LanguageID: languageID,
			Version:    1,
			Text:       content,
		},
	}
	return s.notify("textDocument/didOpen", params)
}

// CloseDocument sends a textDocument/didClose notification
func (s *Server) CloseDocument(path string) error {
	params := DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{
			URI: FileToURI(path),
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

// handleNotification processes LSP notifications from the server
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

// GetLastDiagnostics returns the last received diagnostics for a file
func (s *Server) GetLastDiagnostics(filePath string) []Diagnostic {
	uri := FileToURI(filePath)
	s.diagMu.RLock()
	defer s.diagMu.RUnlock()
	return s.diagnostics[uri]
}

// ClearDiagnostics clears diagnostics for a file
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
