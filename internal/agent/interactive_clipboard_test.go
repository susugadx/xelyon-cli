package agent

import (
	"errors"
	"strings"
	"testing"
)

func TestInteractiveCopyTextAndCopyLastOutput(t *testing.T) {
	oldClipboardWriteAll := clipboardWriteAll
	t.Cleanup(func() { clipboardWriteAll = oldClipboardWriteAll })

	var copied []string
	clipboardWriteAll = func(text string) error {
		copied = append(copied, text)
		return nil
	}

	agent := &Agent{agentConversationState: agentConversationState{lastOutputs: []string{"line1\nline2"}}}

	if err := CopyText("hello"); err != nil {
		t.Fatalf("CopyText() error = %v", err)
	}
	msg, err := agent.CopyLastOutput()
	if err != nil {
		t.Fatalf("CopyLastOutput() error = %v", err)
	}
	if msg != "Copied 2 lines" {
		t.Fatalf("CopyLastOutput() message = %q, want %q", msg, "Copied 2 lines")
	}
	if len(copied) != 2 || copied[0] != "hello" || copied[1] != "line1\nline2" {
		t.Fatalf("clipboard writes = %v, want [hello line1\\nline2]", copied)
	}
}

func TestInteractiveCopyLastOutputErrorPaths(t *testing.T) {
	if _, err := (&Agent{}).CopyLastOutput(); err == nil || !strings.Contains(err.Error(), "no AI output") {
		t.Fatalf("CopyLastOutput() error = %v, want no output error", err)
	}

	oldClipboardWriteAll := clipboardWriteAll
	t.Cleanup(func() { clipboardWriteAll = oldClipboardWriteAll })
	clipboardWriteAll = func(text string) error {
		return errors.New("clipboard unavailable")
	}

	agent := &Agent{agentConversationState: agentConversationState{lastOutputs: []string{"result"}}}
	if _, err := agent.CopyLastOutput(); err == nil || !strings.Contains(err.Error(), "clipboard unavailable") {
		t.Fatalf("CopyLastOutput() error = %v, want clipboard error", err)
	}
}
