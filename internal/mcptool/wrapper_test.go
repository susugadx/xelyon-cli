package mcptool

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type testCaller struct{}

func (testCaller) CallTool(context.Context, string, string, map[string]any) (string, error) {
	return "ok", nil
}

func TestRegisterToRegistry(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterToRegistry(registry, testCaller{}, []Definition{{
		ServerName:  "server-a",
		Name:        "tool-one",
		Description: "First tool",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}})

	tool := registry.GetTool("mcp_server_a_tool_one")
	if tool == nil {
		t.Fatal("registered MCP tool not found")
	}
	if tool.Description() != "First tool" {
		t.Fatalf("Description() = %q", tool.Description())
	}
}

func TestWrapperValidateArgs(t *testing.T) {
	t.Run("top-level required missing returns error", func(t *testing.T) {
		wrapper := NewWrapper(WrapperOptions{
			ToolName: "echo",
			InputSchema: json.RawMessage(`{
				"type":"object",
				"properties":{"name":{"type":"string"}},
				"required":["name"]
			}`),
		})

		err := wrapper.ValidateArgs(io.Discard, map[string]string{})
		if err == nil || !strings.Contains(err.Error(), "required argument 'name' is missing") {
			t.Fatalf("ValidateArgs() error = %v, want missing required argument", err)
		}
	})

	t.Run("legacy property required missing returns error", func(t *testing.T) {
		wrapper := NewWrapper(WrapperOptions{
			ToolName: "echo",
			InputSchema: json.RawMessage(`{
				"type":"object",
				"properties":{"path":{"type":"string","required":true}}
			}`),
		})

		err := wrapper.ValidateArgs(io.Discard, map[string]string{})
		if err == nil || !strings.Contains(err.Error(), "required argument 'path' is missing") {
			t.Fatalf("ValidateArgs() error = %v, want legacy missing required argument", err)
		}
	})

	t.Run("top-level required is satisfied by present empty value", func(t *testing.T) {
		wrapper := NewWrapper(WrapperOptions{
			ToolName: "echo",
			InputSchema: json.RawMessage(`{
				"type":"object",
				"properties":{"name":{"type":"string"}},
				"required":["name"]
			}`),
		})

		if err := wrapper.ValidateArgs(io.Discard, map[string]string{"name": ""}); err != nil {
			t.Fatalf("ValidateArgs() error = %v, want nil for present empty value", err)
		}
	})

	t.Run("invalid schema emits warning and continues", func(t *testing.T) {
		wrapper := NewWrapper(WrapperOptions{
			ToolName:    "echo",
			InputSchema: json.RawMessage(`{invalid json`),
		})
		var out bytes.Buffer

		if err := wrapper.ValidateArgs(&out, map[string]string{"name": "tester"}); err != nil {
			t.Fatalf("ValidateArgs() error = %v, want nil", err)
		}
		if !strings.Contains(out.String(), "Failed to parse input schema") {
			t.Fatalf("warning output = %q, want schema parse warning", out.String())
		}
	})
}

type recordingCaller struct {
	calls int
}

func (c *recordingCaller) CallTool(context.Context, string, string, map[string]any) (string, error) {
	c.calls++
	return "called", nil
}

type recordingPrompter struct {
	calls int
}

func (p *recordingPrompter) Prompt(context.Context, ui.PromptRequest) (ui.PromptResponse, error) {
	p.calls++
	return ui.PromptResponse{Action: ui.PromptActionYes}, nil
}

func TestWrapperRunValidationErrorBeforePromptAndCaller(t *testing.T) {
	t.Setenv("XELYON_INTERACTIVE_CONFIRM", "1")

	caller := &recordingCaller{}
	prompter := &recordingPrompter{}
	wrapper := NewWrapper(WrapperOptions{
		Caller:     caller,
		ServerName: "github",
		ToolName:   "get_issue",
		InputSchema: json.RawMessage(`{
			"type":"object",
			"properties":{"owner":{"type":"string"}},
			"required":["owner"]
		}`),
	})
	var stdout bytes.Buffer
	runtime := ui.NewRuntime(strings.NewReader("y\n"), &stdout, &stdout)
	runtime.SetPrompter(prompter)

	result, fileChange, err := wrapper.Run(tools.ExecutionContext{
		Context: context.Background(),
		Stdin:   runtime.Input(),
		Stdout:  runtime.Output(),
		Stderr:  runtime.ErrorOutput(),
		Runtime: runtime,
		Config:  config.DefaultConfig(),
	}, map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "required argument 'owner' is missing") {
		t.Fatalf("Run() error = %v, want validation error", err)
	}
	if fileChange != nil {
		t.Fatalf("fileChange = %#v, want nil", fileChange)
	}
	if !strings.Contains(result, "Validation Error: required argument 'owner' is missing") {
		t.Fatalf("result = %q, want validation error message", result)
	}
	if prompter.calls != 0 {
		t.Fatalf("prompt calls = %d, want 0", prompter.calls)
	}
	if caller.calls != 0 {
		t.Fatalf("caller calls = %d, want 0", caller.calls)
	}
}
