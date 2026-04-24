package lsp

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
)

// Server manages a single LSP server process
type Server struct {
	name     string
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   *bufio.Reader
	rootURI  string
	debugOut io.Writer

	mu          sync.Mutex
	requestID   atomic.Int64
	pending     map[int]chan *Response
	initialized bool

	// Diagnostics storage (URI -> diagnostics)
	diagMu      sync.RWMutex
	diagnostics map[string][]Diagnostic
	openDocs    map[string]struct{}

	// Graceful shutdown
	done      chan struct{}
	closeOnce sync.Once
}

// NewServer creates a new LSP server instance (does not start yet)
func NewServer(name string) *Server {
	return &Server{
		name:        name,
		debugOut:    io.Discard,
		pending:     make(map[int]chan *Response),
		diagnostics: make(map[string][]Diagnostic),
		openDocs:    make(map[string]struct{}),
		done:        make(chan struct{}),
	}
}

// SetDebugOutput は LSP サーバーのデバッグ出力先を設定する。
func (s *Server) SetDebugOutput(w io.Writer) {
	if w == nil {
		w = io.Discard
	}
	s.debugOut = w
}

func (s *Server) debugf(format string, args ...interface{}) {
	if os.Getenv("XELYON_DEBUG_LSP") != "1" {
		return
	}
	_, _ = fmt.Fprintf(s.debugOut, format, args...)
}
