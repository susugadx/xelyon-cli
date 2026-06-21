package mcptool

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/mcpapproval"
	"github.com/susugadx/xelyon-cli/internal/ui"
	"strings"
	"testing"
)

func TestWrapperRunDefaultConfirmIgnoresGlobalAutoApprove(t *testing.T) {
	t.Setenv("XELYON_INTERACTIVE_CONFIRM", "1")
	caller := &recordingCaller{}
	prompter := &responsePrompter{resp: ui.PromptResponse{Action: ui.PromptActionNo}}
	wrapper := NewWrapper(WrapperOptions{
		Caller:     caller,
		ServerName: "github",
		ToolName:   "list_issues",
	})

	execCtx := newPromptedExecutionContext(context.Background(), prompter)
	execCtx.AutoApprove = true
	result, fileChange, err := wrapper.Run(execCtx, map[string]string{})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil user rejection", err)
	}
	if fileChange != nil {
		t.Fatalf("fileChange = %#v, want nil", fileChange)
	}
	if result != "User rejected MCP tool execution" {
		t.Fatalf("result = %q, want user rejection", result)
	}
	if prompter.calls != 1 {
		t.Fatalf("prompt calls = %d, want 1 because MCP default confirm ignores global auto approve", prompter.calls)
	}
	if caller.calls != 0 {
		t.Fatalf("caller calls = %d, want 0 because MCP default confirm ignores global auto approve", caller.calls)
	}
}

func TestWrapperRunDefaultConfirmIgnoresFullAutoExecutionMode(t *testing.T) {
	t.Setenv("XELYON_INTERACTIVE_CONFIRM", "1")
	caller := &recordingCaller{}
	prompter := &responsePrompter{resp: ui.PromptResponse{Action: ui.PromptActionNo}}
	wrapper := NewWrapper(WrapperOptions{
		Caller:     caller,
		ServerName: "github",
		ToolName:   "list_issues",
	})
	cfg := config.DefaultConfig()
	cfg.Execution.Mode = string(config.ExecutionFullAuto)
	execCtx := newPromptedExecutionContext(context.Background(), prompter)
	execCtx.AutoApprove = false
	execCtx.Config = cfg

	result, _, err := wrapper.Run(execCtx, map[string]string{})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil user rejection", err)
	}
	if result != "User rejected MCP tool execution" {
		t.Fatalf("result = %q, want user rejection", result)
	}
	if prompter.calls != 1 {
		t.Fatalf("prompt calls = %d, want 1 because MCP confirm ignores full_auto", prompter.calls)
	}
	if caller.calls != 0 {
		t.Fatalf("caller calls = %d, want 0 because MCP confirm ignores full_auto", caller.calls)
	}
}

func TestWrapperRunAutoApprovalSkipsPromptAndCallsMCP(t *testing.T) {
	t.Setenv("XELYON_INTERACTIVE_CONFIRM", "1")
	caller := &recordingCaller{}
	prompter := &recordingPrompter{}
	wrapper := NewWrapper(WrapperOptions{
		Caller:     caller,
		ServerName: "github",
		ToolName:   "list_issues",
		Approval:   mcpapproval.ModeAuto,
	})

	result, fileChange, err := wrapper.Run(newPromptedExecutionContext(context.Background(), prompter), map[string]string{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if fileChange != nil {
		t.Fatalf("fileChange = %#v, want nil", fileChange)
	}
	if result != "called" {
		t.Fatalf("result = %q, want called", result)
	}
	if prompter.calls != 0 {
		t.Fatalf("prompt calls = %d, want 0 for explicit MCP auto", prompter.calls)
	}
	if caller.calls != 1 {
		t.Fatalf("caller calls = %d, want 1", caller.calls)
	}
}

func TestWrapperRunDenyDoesNotValidatePromptOrCall(t *testing.T) {
	t.Setenv("XELYON_INTERACTIVE_CONFIRM", "1")
	caller := &recordingCaller{}
	prompter := &recordingPrompter{}
	wrapper := NewWrapper(WrapperOptions{
		Caller:     caller,
		ServerName: "github",
		ToolName:   "delete_repository",
		Approval:   mcpapproval.ModeDeny,
		InputSchema: json.RawMessage(`{
			"type":"object",
			"properties":{"owner":{"type":"string"}},
			"required":["owner"]
		}`),
	})

	result, fileChange, err := wrapper.Run(newPromptedExecutionContext(context.Background(), prompter), map[string]string{})
	if !errors.Is(err, ErrApprovalDenied) {
		t.Fatalf("Run() error = %v, want ErrApprovalDenied", err)
	}
	if fileChange != nil {
		t.Fatalf("fileChange = %#v, want nil", fileChange)
	}
	if !strings.Contains(result, "MCP tool execution denied by approval policy") {
		t.Fatalf("result = %q, want denied policy message", result)
	}
	if prompter.calls != 0 {
		t.Fatalf("prompt calls = %d, want 0 for denied MCP tool", prompter.calls)
	}
	if caller.calls != 0 {
		t.Fatalf("caller calls = %d, want 0 for denied MCP tool", caller.calls)
	}
}

func TestWrapperRunHeadlessConfirmRequiresApproval(t *testing.T) {
	caller := &recordingCaller{}
	wrapper := NewWrapper(WrapperOptions{
		Caller:     caller,
		ServerName: "github",
		ToolName:   "list_issues",
	})
	execCtx := newAutoApprovedExecutionContext(context.Background())
	execCtx.Headless = true

	result, fileChange, err := wrapper.Run(execCtx, map[string]string{})
	if !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("Run() error = %v, want ErrApprovalRequired", err)
	}
	if fileChange != nil {
		t.Fatalf("fileChange = %#v, want nil", fileChange)
	}
	if !strings.Contains(result, "approval_required") {
		t.Fatalf("result = %q, want approval_required marker", result)
	}
	if caller.calls != 0 {
		t.Fatalf("caller calls = %d, want 0 for headless confirm MCP tool", caller.calls)
	}
}

func TestWrapperRunHeadlessAutoCallsMCP(t *testing.T) {
	caller := &recordingCaller{}
	wrapper := NewWrapper(WrapperOptions{
		Caller:     caller,
		ServerName: "github",
		ToolName:   "list_issues",
		Approval:   mcpapproval.ModeAuto,
	})
	execCtx := newAutoApprovedExecutionContext(context.Background())
	execCtx.Headless = true

	result, fileChange, err := wrapper.Run(execCtx, map[string]string{})
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
		t.Fatalf("caller calls = %d, want 1 for headless explicit MCP auto", caller.calls)
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
