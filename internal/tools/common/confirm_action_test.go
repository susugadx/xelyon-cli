package common

import (
	"io"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestConfirmWithIO_InteractiveDecisionPaths(t *testing.T) {
	t.Setenv("XELYON_INTERACTIVE_CONFIRM", "1")

	t.Run("yes", func(t *testing.T) {
		var out strings.Builder
		dec := ConfirmWithIO(
			ui.NewPromptIO(strings.NewReader("y\n"), &out, io.Discard, nil),
			"Proceed?",
		)

		if dec.Action != ConfirmYes {
			t.Fatalf("Action = %q, want %q", dec.Action, ConfirmYes)
		}
	})

	t.Run("no", func(t *testing.T) {
		var out strings.Builder
		dec := ConfirmWithIO(
			ui.NewPromptIO(strings.NewReader("n\n"), &out, io.Discard, nil),
			"Proceed?",
		)

		if dec.Action != ConfirmNo {
			t.Fatalf("Action = %q, want %q", dec.Action, ConfirmNo)
		}
	})
}

func TestConfirmInteractiveRequestWithIO_ExplicitPolicyRePromptsBeforeFeedback(t *testing.T) {
	var out strings.Builder
	result := ConfirmInteractiveRequestWithIO(
		ui.NewPromptIO(strings.NewReader("\n2\nneeds more context\n\n"), &out, io.Discard, nil),
		ui.NewPlanApprovalPromptRequest(),
	)

	if result.Action != string(ConfirmComment) {
		t.Fatalf("Action = %q, want %q", result.Action, ConfirmComment)
	}
	if result.Comment != "needs more context" {
		t.Fatalf("Comment = %q, want feedback after retry", result.Comment)
	}
	if !strings.Contains(out.String(), "Please choose one of the listed options.") {
		t.Fatalf("output = %q, want retry guidance", out.String())
	}
}

func TestConfirmToolAction_DecisionMatrix(t *testing.T) {
	t.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")

	t.Run("auto approve flag bypasses policy", func(t *testing.T) {
		var out strings.Builder
		dec := ConfirmToolAction(
			ui.NewPromptIO(strings.NewReader("n\n"), &out, io.Discard, nil),
			ConfirmOptions{AutoApprove: true},
			"delete_file",
			"Delete file?",
			ToolConfirmContext{},
		)

		if dec.Action != ConfirmYes {
			t.Fatalf("Action = %q, want %q", dec.Action, ConfirmYes)
		}
		if !strings.Contains(out.String(), "Dangerous (destructive operation)") {
			t.Fatalf("output = %q, want auto-approve safety message", out.String())
		}
	})

	t.Run("balanced auto approves safe reads", func(t *testing.T) {
		policy := config.ResolveExecutionPolicy(config.ExecutionConfig{Mode: string(config.ExecutionBalanced)})
		var out strings.Builder
		dec := ConfirmToolAction(
			ui.NewPromptIO(strings.NewReader("n\n"), &out, io.Discard, nil),
			ConfirmOptions{Policy: &policy},
			"read_file",
			"Read file?",
			ToolConfirmContext{},
		)

		if dec.Action != ConfirmYes {
			t.Fatalf("Action = %q, want %q", dec.Action, ConfirmYes)
		}
		if !strings.Contains(out.String(), "Safe read-only") {
			t.Fatalf("output = %q, want safe read message", out.String())
		}
	})

	t.Run("full auto approves medium tools outside trusted write list", func(t *testing.T) {
		policy := config.ResolveExecutionPolicy(config.ExecutionConfig{Mode: string(config.ExecutionFullAuto)})
		var out strings.Builder
		dec := ConfirmToolAction(
			ui.NewPromptIO(strings.NewReader("n\n"), &out, io.Discard, nil),
			ConfirmOptions{Policy: &policy},
			"web_search",
			"Search web?",
			ToolConfirmContext{},
		)

		if dec.Action != ConfirmYes {
			t.Fatalf("Action = %q, want %q", dec.Action, ConfirmYes)
		}
		if !strings.Contains(out.String(), "Full auto") {
			t.Fatalf("output = %q, want full auto message", out.String())
		}
	})

	t.Run("legacy safe fallback still works when policy disables read auto approve", func(t *testing.T) {
		manualPolicy := config.ExecutionPolicy{Mode: config.ExecutionBalanced}
		var out strings.Builder
		dec := ConfirmToolAction(
			ui.NewPromptIO(strings.NewReader("n\n"), &out, io.Discard, nil),
			ConfirmOptions{
				Policy: &manualPolicy,
				Config: &config.Config{ToolConfirm: config.ToolConfirmConfig{AutoApproveSafe: true}},
			},
			"read_file",
			"Read file?",
			ToolConfirmContext{},
		)

		if dec.Action != ConfirmYes {
			t.Fatalf("Action = %q, want %q", dec.Action, ConfirmYes)
		}
		if !strings.Contains(out.String(), "Safe read-only") {
			t.Fatalf("output = %q, want legacy safe fallback", out.String())
		}
	})

	t.Run("legacy medium fallback still works when policy does not auto approve writes", func(t *testing.T) {
		manualPolicy := config.ExecutionPolicy{Mode: config.ExecutionBalanced}
		var out strings.Builder
		dec := ConfirmToolAction(
			ui.NewPromptIO(strings.NewReader("n\n"), &out, io.Discard, nil),
			ConfirmOptions{
				Policy: &manualPolicy,
				Config: &config.Config{ToolConfirm: config.ToolConfirmConfig{AutoApproveMedium: true}},
			},
			"write_file",
			"Write file?",
			ToolConfirmContext{TargetPath: "internal/config/config.go"},
		)

		if dec.Action != ConfirmYes {
			t.Fatalf("Action = %q, want %q", dec.Action, ConfirmYes)
		}
		if !strings.Contains(out.String(), "Medium write") {
			t.Fatalf("output = %q, want legacy medium fallback", out.String())
		}
	})

	t.Run("falls back to prompt when no policy branch matches", func(t *testing.T) {
		manualPolicy := config.ExecutionPolicy{Mode: config.ExecutionBalanced}
		var out strings.Builder
		dec := ConfirmToolAction(
			ui.NewPromptIO(strings.NewReader("n\n"), &out, io.Discard, nil),
			ConfirmOptions{Policy: &manualPolicy},
			"write_file",
			"Write file?",
			ToolConfirmContext{TargetPath: "internal/config/config.go"},
		)

		if dec.Action != ConfirmNo {
			t.Fatalf("Action = %q, want %q", dec.Action, ConfirmNo)
		}
		if strings.Contains(out.String(), "Auto-approved") {
			t.Fatalf("output = %q, want prompt path without auto-approve", out.String())
		}
	})
}

func TestConfirmToolAction_MCPDynamicToolPolicy(t *testing.T) {
	t.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")

	t.Run("balanced prompts", func(t *testing.T) {
		policy := config.ResolveExecutionPolicy(config.ExecutionConfig{Mode: string(config.ExecutionBalanced)})
		var out strings.Builder
		dec := ConfirmToolAction(
			ui.NewPromptIO(strings.NewReader("n\n"), &out, io.Discard, nil),
			ConfirmOptions{Policy: &policy},
			"mcp_github_get_issue",
			"Run MCP tool?",
			ToolConfirmContext{},
		)

		if dec.Action != ConfirmNo {
			t.Fatalf("Action = %q, want %q", dec.Action, ConfirmNo)
		}
		if strings.Contains(out.String(), "Auto-approved") {
			t.Fatalf("output = %q, want prompt path", out.String())
		}
	})

	t.Run("trusted prompts", func(t *testing.T) {
		policy := config.ResolveExecutionPolicy(config.ExecutionConfig{Mode: string(config.ExecutionTrusted)})
		var out strings.Builder
		dec := ConfirmToolAction(
			ui.NewPromptIO(strings.NewReader("n\n"), &out, io.Discard, nil),
			ConfirmOptions{Policy: &policy},
			"mcp_github_get_issue",
			"Run MCP tool?",
			ToolConfirmContext{},
		)

		if dec.Action != ConfirmNo {
			t.Fatalf("Action = %q, want %q", dec.Action, ConfirmNo)
		}
		if strings.Contains(out.String(), "Auto-approved") {
			t.Fatalf("output = %q, want prompt path", out.String())
		}
	})

	t.Run("full auto approves", func(t *testing.T) {
		policy := config.ResolveExecutionPolicy(config.ExecutionConfig{Mode: string(config.ExecutionFullAuto)})
		var out strings.Builder
		dec := ConfirmToolAction(
			ui.NewPromptIO(strings.NewReader("n\n"), &out, io.Discard, nil),
			ConfirmOptions{Policy: &policy},
			"mcp_github_get_issue",
			"Run MCP tool?",
			ToolConfirmContext{},
		)

		if dec.Action != ConfirmYes {
			t.Fatalf("Action = %q, want %q", dec.Action, ConfirmYes)
		}
		if !strings.Contains(out.String(), "Full auto") {
			t.Fatalf("output = %q, want full auto auto-approve message", out.String())
		}
	})

	t.Run("auto approve flag approves", func(t *testing.T) {
		var out strings.Builder
		dec := ConfirmToolAction(
			ui.NewPromptIO(strings.NewReader(""), &out, io.Discard, nil),
			ConfirmOptions{AutoApprove: true},
			"mcp_github_get_issue",
			"Run MCP tool?",
			ToolConfirmContext{},
		)

		if dec.Action != ConfirmYes {
			t.Fatalf("Action = %q, want %q", dec.Action, ConfirmYes)
		}
		if !strings.Contains(out.String(), "Auto-approved") {
			t.Fatalf("output = %q, want auto-approve message", out.String())
		}
	})
}
