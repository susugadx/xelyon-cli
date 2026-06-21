package common

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/stdio"
	"github.com/susugadx/xelyon-cli/internal/uiprompt"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

type recordingPrompter struct {
	reqs []uiprompt.PromptRequest
	resp uiprompt.PromptResponse
}

func (p *recordingPrompter) Prompt(_ context.Context, req uiprompt.PromptRequest) (uiprompt.PromptResponse, error) {
	p.reqs = append(p.reqs, req)
	return p.resp, nil
}

func TestConfirmOptions_EffectivePolicy(t *testing.T) {
	t.Run("explicit policy takes precedence", func(t *testing.T) {
		policy := config.ExecutionPolicy{Mode: config.ExecutionFullAuto, AutoApproveFullAuto: true}
		opts := ConfirmOptions{
			Config: &config.Config{Execution: config.ExecutionConfig{Mode: string(config.ExecutionBalanced)}},
			Policy: &policy,
		}

		got := opts.EffectivePolicy()
		if got.Mode != config.ExecutionFullAuto {
			t.Fatalf("Mode = %q, want %q", got.Mode, config.ExecutionFullAuto)
		}
		if !got.AutoApproveFullAuto {
			t.Fatal("AutoApproveFullAuto should come from explicit policy")
		}
	})

	t.Run("defaults to balanced when empty", func(t *testing.T) {
		got := (ConfirmOptions{}).EffectivePolicy()
		if got.Mode != config.ExecutionBalanced {
			t.Fatalf("Mode = %q, want %q", got.Mode, config.ExecutionBalanced)
		}
	})
}

func TestConfirmWithIO_InteractiveComment(t *testing.T) {
	t.Setenv("XELYON_INTERACTIVE_CONFIRM", "1")

	var out strings.Builder
	dec := ConfirmWithIO(
		uiruntime.NewPromptIO(strings.NewReader("c\nneeds more context\n\n"), &out, io.Discard, nil),
		"Proceed?",
	)

	if dec.Action != ConfirmComment {
		t.Fatalf("Action = %q, want %q", dec.Action, ConfirmComment)
	}
	if dec.Comment != "needs more context" {
		t.Fatalf("Comment = %q, want %q", dec.Comment, "needs more context")
	}
}

func TestConfirmWithFeedback_Comment(t *testing.T) {
	t.Setenv("XELYON_INTERACTIVE_CONFIRM", "1")

	var out strings.Builder
	stdio.SetDefaults(strings.NewReader("c\nneeds more context\n\n"), &out, io.Discard)
	t.Cleanup(func() {
		stdio.SetDefaults(nil, nil, nil)
	})

	approved, comment, image := ConfirmWithFeedback("Proceed?")
	if approved {
		t.Fatal("approved = true, want false for comment path")
	}
	if comment != "needs more context" {
		t.Fatalf("comment = %q, want %q", comment, "needs more context")
	}
	if image != nil {
		t.Fatalf("image = %v, want nil", image)
	}
}

func TestConfirmToolAction_PolicyPaths(t *testing.T) {
	t.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")

	t.Run("always confirm overrides full auto", func(t *testing.T) {
		policy := config.ResolveExecutionPolicy(config.ExecutionConfig{
			Mode:          string(config.ExecutionFullAuto),
			AlwaysConfirm: []string{string(config.ConfirmDeleteFile)},
		})
		var out strings.Builder
		dec := ConfirmToolAction(
			uiruntime.NewPromptIO(strings.NewReader("n\n"), &out, io.Discard, nil),
			ConfirmOptions{Policy: &policy},
			"delete_file",
			"Delete file?",
			ToolConfirmContext{},
		)

		if dec.Action != ConfirmNo {
			t.Fatalf("Action = %q, want %q", dec.Action, ConfirmNo)
		}
		if strings.Contains(out.String(), "Auto-approved") {
			t.Fatalf("output = %q, want interactive confirmation path", out.String())
		}
	})

	t.Run("trusted write auto-approves inside workspace", func(t *testing.T) {
		policy := config.ResolveExecutionPolicy(config.ExecutionConfig{Mode: string(config.ExecutionTrusted)})
		var out strings.Builder
		dec := ConfirmToolAction(
			uiruntime.NewPromptIO(strings.NewReader("n\n"), &out, io.Discard, nil),
			ConfirmOptions{Policy: &policy},
			"write_file",
			"Write file?",
			ToolConfirmContext{TargetPath: "internal/config/config.go"},
		)

		if dec.Action != ConfirmYes {
			t.Fatalf("Action = %q, want %q", dec.Action, ConfirmYes)
		}
		if !strings.Contains(out.String(), "Trusted write") {
			t.Fatalf("output = %q, want trusted write auto-approve", out.String())
		}
	})

	t.Run("trusted write prompts for workspace outside path", func(t *testing.T) {
		policy := config.ResolveExecutionPolicy(config.ExecutionConfig{
			Mode:          string(config.ExecutionTrusted),
			AlwaysConfirm: []string{string(config.ConfirmWorkspaceOutside)},
		})
		var out strings.Builder
		dec := ConfirmToolAction(
			uiruntime.NewPromptIO(strings.NewReader("n\n"), &out, io.Discard, nil),
			ConfirmOptions{Policy: &policy},
			"write_file",
			"Write outside workspace?",
			ToolConfirmContext{TargetPath: "/etc/hosts"},
		)

		if dec.Action != ConfirmNo {
			t.Fatalf("Action = %q, want %q", dec.Action, ConfirmNo)
		}
		if strings.Contains(out.String(), "Trusted write") || strings.Contains(out.String(), "Auto-approved") {
			t.Fatalf("output = %q, want prompt path", out.String())
		}
	})

	t.Run("full auto approves low safety tool", func(t *testing.T) {
		policy := config.ResolveExecutionPolicy(config.ExecutionConfig{Mode: string(config.ExecutionFullAuto)})
		var out strings.Builder
		dec := ConfirmToolAction(
			uiruntime.NewPromptIO(strings.NewReader("n\n"), &out, io.Discard, nil),
			ConfirmOptions{Policy: &policy},
			"bash",
			"Run bash?",
			ToolConfirmContext{},
		)

		if dec.Action != ConfirmYes {
			t.Fatalf("Action = %q, want %q", dec.Action, ConfirmYes)
		}
		if !strings.Contains(out.String(), "Full auto") {
			t.Fatalf("output = %q, want full auto message", out.String())
		}
	})
}

func TestConfirmToolAction_UsesPrompterWithoutReadingLegacyInput(t *testing.T) {
	t.Setenv("XELYON_INTERACTIVE_CONFIRM", "1")

	prompter := &recordingPrompter{resp: uiprompt.PromptResponse{Action: uiprompt.PromptActionYes}}
	runtime := uiruntime.NewRuntime(strings.NewReader("n\n"), io.Discard, io.Discard)
	runtime.SetPrompter(prompter)

	policy := config.ResolveExecutionPolicy(config.ExecutionConfig{Mode: string(config.ExecutionBalanced)})
	dec := ConfirmToolAction(
		runtime.PromptIO(),
		ConfirmOptions{Policy: &policy},
		"bash",
		"Run bash?",
		ToolConfirmContext{},
	)

	if dec.Action != ConfirmYes {
		t.Fatalf("Action = %q, want yes from prompter", dec.Action)
	}
	if len(prompter.reqs) != 1 {
		t.Fatalf("prompt calls = %d, want 1", len(prompter.reqs))
	}
	if !prompter.reqs[0].AllowComment {
		t.Fatal("interactive confirm should allow comments")
	}
}

func TestConfirmToolAction_AlwaysConfirmBeatsAutoApproveWithPrompter(t *testing.T) {
	t.Setenv("XELYON_INTERACTIVE_CONFIRM", "1")

	prompter := &recordingPrompter{resp: uiprompt.PromptResponse{Action: uiprompt.PromptActionNo}}
	runtime := uiruntime.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	runtime.SetPrompter(prompter)
	policy := config.ResolveExecutionPolicy(config.ExecutionConfig{
		Mode:          string(config.ExecutionFullAuto),
		AlwaysConfirm: []string{string(config.ConfirmDeleteFile)},
	})

	dec := ConfirmToolAction(
		runtime.PromptIO(),
		ConfirmOptions{AutoApprove: true, Policy: &policy},
		"delete_file",
		"Delete?",
		ToolConfirmContext{},
	)

	if dec.Action != ConfirmNo {
		t.Fatalf("Action = %q, want prompted no", dec.Action)
	}
	if len(prompter.reqs) != 1 {
		t.Fatalf("prompt calls = %d, want 1", len(prompter.reqs))
	}
}

func TestConfirmWithIO_InteractiveDisabledPrompterOmitsComment(t *testing.T) {
	t.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")

	prompter := &recordingPrompter{resp: uiprompt.PromptResponse{Action: uiprompt.PromptActionComment, Text: "feedback"}}
	runtime := uiruntime.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	runtime.SetPrompter(prompter)

	dec := ConfirmWithIO(runtime.PromptIO(), "Proceed?")
	if dec.Action != ConfirmNo {
		t.Fatalf("Action = %q, want no because comment is not offered", dec.Action)
	}
	if len(prompter.reqs) != 1 {
		t.Fatalf("prompt calls = %d, want 1", len(prompter.reqs))
	}
	if prompter.reqs[0].AllowComment {
		t.Fatal("disabled interactive confirm should not allow comments")
	}
	if prompter.reqs[0].ConfirmSubmitPolicy != uiprompt.PromptConfirmSubmitExplicit {
		t.Fatalf("ConfirmSubmitPolicy = %q, want explicit", prompter.reqs[0].ConfirmSubmitPolicy)
	}
}

func TestConfirmWithIO_PrompterCommentParsesImageLine(t *testing.T) {
	t.Setenv("XELYON_INTERACTIVE_CONFIRM", "1")

	imagePath := filepath.Join(t.TempDir(), "comment.png")
	if err := os.WriteFile(imagePath, []byte("png"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	prompter := &recordingPrompter{
		resp: uiprompt.PromptResponse{
			Action: uiprompt.PromptActionComment,
			Text:   "needs more context\nimage:" + imagePath,
		},
	}
	runtime := uiruntime.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	runtime.SetPrompter(prompter)

	dec := ConfirmWithIO(runtime.PromptIO(), "Proceed?")
	if dec.Action != ConfirmComment {
		t.Fatalf("Action = %q, want comment", dec.Action)
	}
	if dec.Comment != "needs more context" {
		t.Fatalf("Comment = %q, want feedback without image line", dec.Comment)
	}
	if dec.Image == nil || dec.Image.Path != imagePath {
		t.Fatalf("Image = %#v, want loaded image %q", dec.Image, imagePath)
	}
}

func TestAllPathsInsideWorkspace_TargetAndMove(t *testing.T) {
	if !AllPathsInsideWorkspace(ToolConfirmContext{TargetPath: "internal/config/config.go", MovePath: "internal/config/config_copy.go"}) {
		t.Fatal("AllPathsInsideWorkspace() = false, want true for inside target and move path")
	}
	if AllPathsInsideWorkspace(ToolConfirmContext{TargetPath: "internal/config/config.go", MovePath: "/etc/hosts"}) {
		t.Fatal("AllPathsInsideWorkspace() = true, want false when move path escapes workspace")
	}
}

func TestIsSubPathCheck(t *testing.T) {
	if !isSubPathCheck("/tmp/work", "/tmp/work") {
		t.Fatal("same path should be treated as sub path")
	}
	if !isSubPathCheck("/tmp/work/file.go", "/tmp/work") {
		t.Fatal("child path should be treated as sub path")
	}
	if isSubPathCheck("/tmp/work-other/file.go", "/tmp/work") {
		t.Fatal("prefix-only sibling should not be treated as sub path")
	}
}
