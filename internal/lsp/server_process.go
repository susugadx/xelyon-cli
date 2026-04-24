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
	"time"
)

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
			if n > 0 {
				s.debugf("[LSP %s stderr] %s", s.name, string(buf[:n]))
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
				References:     &ReferencesCapability{DynamicRegistration: false},
				Definition:     &DefinitionCapability{DynamicRegistration: false},
				Implementation: &ImplementationCapability{DynamicRegistration: false},
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
	for {
		select {
		case <-s.done:
			return
		default:
			// Read Content-Length header
			contentLength, err := s.readHeader()
			if err != nil {
				s.debugf("[LSP %s] readHeader error: %v\n", s.name, err)
				return
			}

			if contentLength == 0 {
				continue
			}

			// Read body
			body := make([]byte, contentLength)
			if _, err := io.ReadFull(s.stdout, body); err != nil {
				s.debugf("[LSP %s] readBody error: %v\n", s.name, err)
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
