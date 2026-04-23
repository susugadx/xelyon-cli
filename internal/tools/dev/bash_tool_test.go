package dev

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestBashTool_EmptyCommand(t *testing.T) {
	tool := &BashTool{}
	_, _, err := tool.Run(tools.ExecutionContext{}, map[string]string{"command": ""})
	if err == nil {
		t.Error("BashTool.Run() should return error for empty command")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error = %v, want to contain 'empty'", err)
	}
}

func TestBashTool_WhitespaceOnlyCommand(t *testing.T) {
	tool := &BashTool{}
	_, _, err := tool.Run(tools.ExecutionContext{}, map[string]string{"command": "   "})
	if err == nil {
		t.Error("BashTool.Run() should return error for whitespace-only command")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error = %v, want to contain 'empty'", err)
	}
}

func TestBashTool_MissingCommandArg(t *testing.T) {
	tool := &BashTool{}
	_, _, err := tool.Run(tools.ExecutionContext{}, map[string]string{})
	if err == nil {
		t.Error("BashTool.Run() should return error when command arg is missing")
	}
}

func TestBashTool_ValidCommand(t *testing.T) {
	tool := &BashTool{}
	output, _, err := tool.Run(tools.ExecutionContext{}, map[string]string{"command": "echo hello"})
	if err != nil {
		t.Fatalf("BashTool.Run() unexpected error: %v", err)
	}
	if !strings.Contains(output, "hello") {
		t.Errorf("output = %q, want to contain 'hello'", output)
	}
}

func TestBashTool_UsesExecutionContextCancellation(t *testing.T) {
	tool := &BashTool{}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)

	output, _, err := tool.Run(tools.ExecutionContext{
		Context: ctx,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}, map[string]string{"command": "tail -f /dev/null"})
	if err != nil {
		t.Fatalf("BashTool.Run() unexpected error: %v", err)
	}
	if !strings.Contains(output, "Command interrupted.") {
		t.Fatalf("expected interrupted output, got %q", output)
	}
}
