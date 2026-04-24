package lsp

import (
	"context"
	"encoding/json"
	"fmt"
)

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
		s.debugf("[LSP %s] write header error: %v\n", s.name, err)
		return err
	}
	if _, err := s.stdin.Write(data); err != nil {
		s.debugf("[LSP %s] write body error: %v\n", s.name, err)
		return err
	}
	return nil
}
