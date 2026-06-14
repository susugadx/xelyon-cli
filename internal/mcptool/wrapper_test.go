package mcptool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

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
		CallTimeout: 7 * time.Minute,
	}})

	tool := registry.GetTool("mcp_server_a_tool_one")
	if tool == nil {
		t.Fatal("registered MCP tool not found")
	}
	if tool.Description() != "First tool" {
		t.Fatalf("Description() = %q", tool.Description())
	}
	wrapper, ok := tool.(*Wrapper)
	if !ok {
		t.Fatalf("registered tool type = %T, want *Wrapper", tool)
	}
	if wrapper.callTimeoutDuration() != 7*time.Minute {
		t.Fatalf("callTimeoutDuration() = %v, want 7m", wrapper.callTimeoutDuration())
	}
}

func TestWrapperCallTimeoutDurationUsesLongDefault(t *testing.T) {
	wrapper := NewWrapper(WrapperOptions{
		Caller:     testCaller{},
		ServerName: "server",
		ToolName:   "tool",
	})

	if got := wrapper.callTimeoutDuration(); got != 600*time.Second {
		t.Fatalf("callTimeoutDuration() = %v, want 600s", got)
	}
}

func TestRegisterToRegistrySkipsDuplicateExportedNames(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterToRegistry(registry, testCaller{}, []Definition{
		{
			ServerName:  "server-a",
			Name:        "tool.one",
			Description: "First tool",
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
		{
			ServerName:  "server_a",
			Name:        "tool_one",
			Description: "Second tool",
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
	})

	tool := registry.GetTool("mcp_server_a_tool_one")
	if tool == nil {
		t.Fatal("registered MCP tool not found")
	}
	if tool.Description() != "First tool" {
		t.Fatalf("Description() = %q, want first duplicate to win", tool.Description())
	}
	defs := registry.GetToolDefinitions()
	if len(defs) != 1 {
		t.Fatalf("registered tool definitions = %d, want 1", len(defs))
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

type responsePrompter struct {
	calls int
	resp  ui.PromptResponse
}

func (p *responsePrompter) Prompt(context.Context, ui.PromptRequest) (ui.PromptResponse, error) {
	p.calls++
	return p.resp, nil
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
	runtime := ui.NewRuntime(strings.NewReader("y\n"), &stdout, &stdout)
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
	runtime := ui.NewRuntime(strings.NewReader("y\n"), &stdout, &stdout)
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

type argsRecordingCaller struct {
	calls int
	args  map[string]any
}

func (c *argsRecordingCaller) CallTool(_ context.Context, _, _ string, args map[string]any) (string, error) {
	c.calls++
	c.args = args
	return "called", nil
}

func TestWrapperRunPassesSchemaStructuredArgs(t *testing.T) {
	caller := &argsRecordingCaller{}
	wrapper := NewWrapper(WrapperOptions{
		Caller:     caller,
		ServerName: "github",
		ToolName:   "create_issue",
		InputSchema: json.RawMessage(`{
			"type":"object",
			"properties":{
				"labels":{"type":"array"},
				"metadata":{"type":"object"},
				"count":{"type":"integer"},
				"title":{"type":"string"}
			}
		}`),
	})

	result, fileChange, err := wrapper.Run(newAutoApprovedExecutionContext(context.Background()), map[string]string{
		"labels":   `["bug","urgent"]`,
		"metadata": `{"issue":123}`,
		"count":    "2",
		"title":    "Fix MCP",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if fileChange != nil {
		t.Fatalf("fileChange = %#v, want nil", fileChange)
	}
	if result != "called" {
		t.Fatalf("result = %q, want called", result)
	}
	if caller.calls != 1 {
		t.Fatalf("caller calls = %d, want 1", caller.calls)
	}
	labels, ok := caller.args["labels"].([]any)
	if !ok || len(labels) != 2 || labels[0] != "bug" || labels[1] != "urgent" {
		t.Fatalf("labels = %#v, want []any with bug/urgent", caller.args["labels"])
	}
	metadata, ok := caller.args["metadata"].(map[string]any)
	if !ok || metadata["issue"] != float64(123) {
		t.Fatalf("metadata = %#v, want map with issue 123", caller.args["metadata"])
	}
	if caller.args["count"] != int64(2) {
		t.Fatalf("count = %#v, want int64(2)", caller.args["count"])
	}
	if caller.args["title"] != "Fix MCP" {
		t.Fatalf("title = %#v, want original string", caller.args["title"])
	}
}

func TestWrapperConvertArgsKeepsScalarWhenSchemaParseFails(t *testing.T) {
	wrapper := NewWrapper(WrapperOptions{
		InputSchema: json.RawMessage(`{
			"type":"object",
			"properties":{
				"count":{"type":"integer"},
				"enabled":{"type":"boolean"},
				"ratio":{"type":"number"}
			}
		}`),
	})

	got := wrapper.ConvertArgsWithSchema(map[string]string{
		"count":   "not-an-int",
		"enabled": "not-a-bool",
		"ratio":   "not-a-number",
	})

	for key, want := range map[string]string{
		"count":   "not-an-int",
		"enabled": "not-a-bool",
		"ratio":   "not-a-number",
	} {
		if got[key] != want {
			t.Fatalf("converted[%s] = %#v, want scalar fallback %q", key, got[key], want)
		}
	}
}

func TestWrapperDefaultsForDescriptionParametersAndEmptyResult(t *testing.T) {
	wrapper := NewWrapper(WrapperOptions{
		ServerName: "github",
		ToolName:   "list_issues",
	})

	if got := wrapper.Description(); got != "MCP tool: list_issues from server github" {
		t.Fatalf("Description() = %q, want default description", got)
	}
	params := wrapper.Parameters()
	if params["type"] != "object" {
		t.Fatalf("Parameters()[type] = %#v, want object", params["type"])
	}
	props, ok := params["properties"].(map[string]interface{})
	if !ok || len(props) != 0 {
		t.Fatalf("Parameters()[properties] = %#v, want empty map", params["properties"])
	}
	if params["additionalProperties"] != false {
		t.Fatalf("Parameters()[additionalProperties] = %#v, want false", params["additionalProperties"])
	}
	if got := wrapper.FormatResult(""); got != "Tool executed successfully (no output)" {
		t.Fatalf("FormatResult(\"\") = %q, want empty-result default", got)
	}
}

func TestWrapperRunRejectAndCommentDoNotCallCaller(t *testing.T) {
	tests := []struct {
		name       string
		resp       ui.PromptResponse
		wantResult string
	}{
		{
			name:       "reject",
			resp:       ui.PromptResponse{Action: ui.PromptActionNo},
			wantResult: "User rejected MCP tool execution",
		},
		{
			name:       "comment",
			resp:       ui.PromptResponse{Action: ui.PromptActionComment, Text: "use a narrower query"},
			wantResult: "User provided feedback: use a narrower query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XELYON_INTERACTIVE_CONFIRM", "1")
			caller := &recordingCaller{}
			prompter := &responsePrompter{resp: tt.resp}
			wrapper := NewWrapper(WrapperOptions{
				Caller:     caller,
				ServerName: "github",
				ToolName:   "search_issues",
			})

			result, fileChange, err := wrapper.Run(newPromptedExecutionContext(context.Background(), prompter), map[string]string{"query": "bug"})
			if err != nil {
				t.Fatalf("Run() error = %v, want nil", err)
			}
			if fileChange != nil {
				t.Fatalf("fileChange = %#v, want nil", fileChange)
			}
			if result != tt.wantResult {
				t.Fatalf("result = %q, want %q", result, tt.wantResult)
			}
			if prompter.calls != 1 {
				t.Fatalf("prompt calls = %d, want 1", prompter.calls)
			}
			if caller.calls != 0 {
				t.Fatalf("caller calls = %d, want 0", caller.calls)
			}
		})
	}
}

type contextWaitingCaller struct {
	calls int
}

func (c *contextWaitingCaller) CallTool(ctx context.Context, _, _ string, _ map[string]any) (string, error) {
	c.calls++
	<-ctx.Done()
	return "", ctx.Err()
}

func TestWrapperRunUsesParentCancellationBeforeWrapperTimeout(t *testing.T) {
	caller := &contextWaitingCaller{}
	wrapper := NewWrapper(WrapperOptions{
		Caller:      caller,
		ServerName:  "github",
		ToolName:    "slow",
		CallTimeout: time.Second,
	})
	parentCtx, cancel := context.WithCancel(context.Background())
	cancel()

	started := time.Now()
	result, _, err := wrapper.Run(newAutoApprovedExecutionContext(parentCtx), map[string]string{})
	elapsed := time.Since(started)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context.Canceled", err)
	}
	if !strings.Contains(result, "request context") {
		t.Fatalf("result = %q, want request context cancellation message", result)
	}
	if caller.calls != 1 {
		t.Fatalf("caller calls = %d, want 1", caller.calls)
	}
	if elapsed >= 200*time.Millisecond {
		t.Fatalf("Run() elapsed = %v, want parent cancellation before wrapper timeout", elapsed)
	}
}

func TestWrapperRunUsesParentDeadlineBeforeWrapperTimeout(t *testing.T) {
	caller := &contextWaitingCaller{}
	wrapper := NewWrapper(WrapperOptions{
		Caller:      caller,
		ServerName:  "github",
		ToolName:    "slow",
		CallTimeout: time.Second,
	})
	parentCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	started := time.Now()
	result, _, err := wrapper.Run(newAutoApprovedExecutionContext(parentCtx), map[string]string{})
	elapsed := time.Since(started)
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run() error = %v, want context.DeadlineExceeded", err)
	}
	if !strings.Contains(result, "request deadline") {
		t.Fatalf("result = %q, want request deadline message", result)
	}
	if caller.calls != 1 {
		t.Fatalf("caller calls = %d, want 1", caller.calls)
	}
	if elapsed >= 500*time.Millisecond {
		t.Fatalf("Run() elapsed = %v, want parent deadline before wrapper timeout", elapsed)
	}
}

func TestWrapperRunUsesWrapperTimeoutWhenParentStillActive(t *testing.T) {
	caller := &contextWaitingCaller{}
	wrapper := NewWrapper(WrapperOptions{
		Caller:      caller,
		ServerName:  "github",
		ToolName:    "slow",
		CallTimeout: 20 * time.Millisecond,
	})

	started := time.Now()
	result, _, err := wrapper.Run(newAutoApprovedExecutionContext(context.Background()), map[string]string{})
	elapsed := time.Since(started)
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run() error = %v, want context.DeadlineExceeded", err)
	}
	if !strings.Contains(result, "timed out after 20ms") {
		t.Fatalf("result = %q, want wrapper timeout message", result)
	}
	if caller.calls != 1 {
		t.Fatalf("caller calls = %d, want 1", caller.calls)
	}
	if elapsed >= 500*time.Millisecond {
		t.Fatalf("Run() elapsed = %v, want wrapper timeout before long wait", elapsed)
	}
}

func newAutoApprovedExecutionContext(ctx context.Context) tools.ExecutionContext {
	runtime := ui.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	return tools.ExecutionContext{
		Context:     ctx,
		Stdin:       runtime.Input(),
		Stdout:      runtime.Output(),
		Stderr:      runtime.ErrorOutput(),
		Runtime:     runtime,
		Config:      config.DefaultConfig(),
		AutoApprove: true,
	}
}

func newPromptedExecutionContext(ctx context.Context, prompter ui.Prompter) tools.ExecutionContext {
	var stdout bytes.Buffer
	runtime := ui.NewRuntime(strings.NewReader(""), &stdout, &stdout)
	runtime.SetPrompter(prompter)
	return tools.ExecutionContext{
		Context: ctx,
		Stdin:   runtime.Input(),
		Stdout:  runtime.Output(),
		Stderr:  runtime.ErrorOutput(),
		Runtime: runtime,
		Config:  config.DefaultConfig(),
	}
}
