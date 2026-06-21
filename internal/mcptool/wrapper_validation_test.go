package mcptool

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
	"io"
	"strings"
	"testing"
)

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

	t.Run("property required marker is ignored", func(t *testing.T) {
		wrapper := NewWrapper(WrapperOptions{
			ToolName: "echo",
			InputSchema: json.RawMessage(`{
				"type":"object",
				"properties":{"path":{"type":"string","required":true}}
			}`),
		})

		err := wrapper.ValidateArgs(io.Discard, map[string]string{})
		if err != nil {
			t.Fatalf("ValidateArgs() error = %v, want nil because required belongs at schema top-level", err)
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

	t.Run("array argument must be JSON array", func(t *testing.T) {
		wrapper := NewWrapper(WrapperOptions{
			ToolName: "create_issue",
			InputSchema: json.RawMessage(`{
				"type":"object",
				"properties":{"labels":{"type":"array"}}
			}`),
		})

		err := wrapper.ValidateArgs(io.Discard, map[string]string{"labels": `"bug"`})
		if err == nil || !strings.Contains(err.Error(), "must be a JSON array") {
			t.Fatalf("ValidateArgs() error = %v, want JSON array validation error", err)
		}
	})

	t.Run("object argument must be JSON object", func(t *testing.T) {
		wrapper := NewWrapper(WrapperOptions{
			ToolName: "upsert",
			InputSchema: json.RawMessage(`{
				"type":"object",
				"properties":{"payload":{"type":"object"}}
			}`),
		})

		err := wrapper.ValidateArgs(io.Discard, map[string]string{"payload": `[1,2]`})
		if err == nil || !strings.Contains(err.Error(), "must be a JSON object") {
			t.Fatalf("ValidateArgs() error = %v, want JSON object validation error", err)
		}
	})
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
	runtime := uiruntime.NewRuntime(strings.NewReader("y\n"), &stdout, &stdout)
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

func TestWrapperRunInvalidStructuredArgBeforePromptAndCaller(t *testing.T) {
	t.Setenv("XELYON_INTERACTIVE_CONFIRM", "1")

	caller := &recordingCaller{}
	prompter := &recordingPrompter{}
	wrapper := NewWrapper(WrapperOptions{
		Caller:     caller,
		ServerName: "github",
		ToolName:   "create_issue",
		InputSchema: json.RawMessage(`{
			"type":"object",
			"properties":{"labels":{"type":"array"}}
		}`),
	})
	var stdout bytes.Buffer
	runtime := uiruntime.NewRuntime(strings.NewReader("y\n"), &stdout, &stdout)
	runtime.SetPrompter(prompter)

	result, _, err := wrapper.Run(tools.ExecutionContext{
		Context: context.Background(),
		Stdin:   runtime.Input(),
		Stdout:  runtime.Output(),
		Stderr:  runtime.ErrorOutput(),
		Runtime: runtime,
		Config:  config.DefaultConfig(),
	}, map[string]string{"labels": `"bug"`})
	if err == nil || !strings.Contains(err.Error(), "must be a JSON array") {
		t.Fatalf("Run() error = %v, want structured validation error", err)
	}
	if !strings.Contains(result, "Validation Error: argument 'labels' must be a JSON array") {
		t.Fatalf("result = %q, want structured validation message", result)
	}
	if prompter.calls != 0 {
		t.Fatalf("prompt calls = %d, want 0", prompter.calls)
	}
	if caller.calls != 0 {
		t.Fatalf("caller calls = %d, want 0", caller.calls)
	}
}

func TestWrapperRunInvalidObjectArgBeforePromptAndCaller(t *testing.T) {
	t.Setenv("XELYON_INTERACTIVE_CONFIRM", "1")

	caller := &recordingCaller{}
	prompter := &recordingPrompter{}
	wrapper := NewWrapper(WrapperOptions{
		Caller:     caller,
		ServerName: "github",
		ToolName:   "upsert_issue",
		InputSchema: json.RawMessage(`{
			"type":"object",
			"properties":{"payload":{"type":"object"}}
		}`),
	})
	var stdout bytes.Buffer
	runtime := uiruntime.NewRuntime(strings.NewReader("y\n"), &stdout, &stdout)
	runtime.SetPrompter(prompter)

	result, _, err := wrapper.Run(tools.ExecutionContext{
		Context: context.Background(),
		Stdin:   runtime.Input(),
		Stdout:  runtime.Output(),
		Stderr:  runtime.ErrorOutput(),
		Runtime: runtime,
		Config:  config.DefaultConfig(),
	}, map[string]string{"payload": `[1,2]`})
	if err == nil || !strings.Contains(err.Error(), "must be a JSON object") {
		t.Fatalf("Run() error = %v, want structured object validation error", err)
	}
	if !strings.Contains(result, "Validation Error: argument 'payload' must be a JSON object") {
		t.Fatalf("result = %q, want structured object validation message", result)
	}
	if prompter.calls != 0 {
		t.Fatalf("prompt calls = %d, want 0", prompter.calls)
	}
	if caller.calls != 0 {
		t.Fatalf("caller calls = %d, want 0", caller.calls)
	}
}
