package common

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func TestConfirmWithAutoApprove_AutoApproveFlag(t *testing.T) {
	promptIO := uiruntime.NewPromptIO(strings.NewReader(""), io.Discard, io.Discard, nil)
	opts := ConfirmOptions{
		AutoApprove: true,
		Config:      config.DefaultConfig(),
	}

	// read_file は SafetyHigh → auto-approve フラグで承認される
	dec := ConfirmWithAutoApproveDecisionAndOptions(promptIO, opts, "read_file", "Allow?")
	if dec.Action != ConfirmYes {
		t.Errorf("expected ConfirmYes for auto-approved tool, got %q", dec.Action)
	}
}

func TestConfirmWithAutoApprove_SafeToolConfig(t *testing.T) {
	promptIO := uiruntime.NewPromptIO(strings.NewReader("n\n"), io.Discard, io.Discard, nil)
	cfg := config.DefaultConfig()
	cfg.ToolConfirm.AutoApproveSafe = true
	opts := ConfirmOptions{
		AutoApprove: false,
		Config:      cfg,
	}

	// list_dir は SafetyHigh → config.AutoApproveSafe で承認される
	dec := ConfirmWithAutoApproveDecisionAndOptions(promptIO, opts, "list_dir", "Allow?")
	if dec.Action != ConfirmYes {
		t.Errorf("expected ConfirmYes for safe tool, got %q", dec.Action)
	}
}

func TestConfirmWithAutoApprove_MediumToolConfig(t *testing.T) {
	promptIO := uiruntime.NewPromptIO(strings.NewReader("n\n"), io.Discard, io.Discard, nil)
	cfg := config.DefaultConfig()
	cfg.ToolConfirm.AutoApproveMedium = true
	opts := ConfirmOptions{
		AutoApprove: false,
		Config:      cfg,
	}

	// write_file は SafetyMedium → config.AutoApproveMedium で承認される
	dec := ConfirmWithAutoApproveDecisionAndOptions(promptIO, opts, "write_file", "Allow?")
	if dec.Action != ConfirmYes {
		t.Errorf("expected ConfirmYes for medium tool, got %q", dec.Action)
	}
}

func TestConfirmWithAutoApprove_LowToolAutoApprovedWithFlag(t *testing.T) {
	// bash は SafetyLow → --auto-approve 有効時は全ツール自動承認
	var out bytes.Buffer
	promptIO := uiruntime.NewPromptIO(strings.NewReader(""), &out, io.Discard, nil)
	opts := ConfirmOptions{
		AutoApprove: true,
		Config:      config.DefaultConfig(),
	}

	dec := ConfirmWithAutoApproveDecisionAndOptions(promptIO, opts, "bash", "Allow?")
	if dec.Action != ConfirmYes {
		t.Errorf("expected ConfirmYes for auto-approved low-safety tool, got %q", dec.Action)
	}
	if !strings.Contains(out.String(), "Auto-approved") {
		t.Errorf("expected Auto-approved message, got: %q", out.String())
	}
}

func TestConfirmWithAutoApprove_LowToolNotApprovedByConfig(t *testing.T) {
	// bash は SafetyLow → config.AutoApproveSafe/Medium では承認されない
	promptIO := uiruntime.NewPromptIO(strings.NewReader("n\n"), io.Discard, io.Discard, nil)
	cfg := config.DefaultConfig()
	cfg.ToolConfirm.AutoApproveSafe = true
	cfg.ToolConfirm.AutoApproveMedium = true
	opts := ConfirmOptions{
		AutoApprove: false,
		Config:      cfg,
	}

	dec := ConfirmWithAutoApproveDecisionAndOptions(promptIO, opts, "bash", "Allow?")
	if dec.Action != ConfirmNo {
		t.Errorf("expected ConfirmNo for low-safety tool without --auto-approve, got %q", dec.Action)
	}
}

func TestConfirmWithAutoApprove_PrintsAutoApprovedMessage(t *testing.T) {
	var out bytes.Buffer
	promptIO := uiruntime.NewPromptIO(strings.NewReader(""), &out, io.Discard, nil)
	opts := ConfirmOptions{
		AutoApprove: true,
		Config:      config.DefaultConfig(),
	}

	ConfirmWithAutoApproveDecisionAndOptions(promptIO, opts, "search_code", "Allow?")
	if !strings.Contains(out.String(), "Auto-approved") {
		t.Errorf("expected auto-approved message, got: %q", out.String())
	}
}
