package tui

import "testing"

func TestComposerSubmission_BuildsPlainChatPayload(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.textInput.SetValue("  hello world  ")

	submission, ok := m.buildComposerSubmission()
	if !ok {
		t.Fatal("buildComposerSubmission() = !ok, want ok")
	}
	if submission.kind != composerSubmissionChat {
		t.Fatalf("submission.kind = %d, want %d", submission.kind, composerSubmissionChat)
	}
	if submission.payload != "hello world" {
		t.Fatalf("submission.payload = %q, want %q", submission.payload, "hello world")
	}
}

func TestComposerSubmission_BuildsCommandPayloadFromFoldedComposer(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	m.handleComposerPaste("line1\nline2")
	m.textInput.SetValue("/clear")

	submission, ok := m.buildComposerSubmission()
	if !ok {
		t.Fatal("buildComposerSubmission() = !ok, want ok")
	}
	if submission.kind != composerSubmissionCommand {
		t.Fatalf("submission.kind = %d, want %d", submission.kind, composerSubmissionCommand)
	}
	if submission.commandInput != "/clear" {
		t.Fatalf("submission.commandInput = %q, want %q", submission.commandInput, "/clear")
	}
	if submission.payload != "line1\nline2/clear" {
		t.Fatalf("submission.payload = %q, want %q", submission.payload, "line1\nline2/clear")
	}
}

func TestComposerSubmission_RejectsWhitespaceOnlyInput(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.textInput.SetValue(" \t ")

	if _, ok := m.buildComposerSubmission(); ok {
		t.Fatal("buildComposerSubmission() = ok, want !ok")
	}
}
