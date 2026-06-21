package attachments

import (
	"strings"
	"testing"
)

func TestDispatchPolicyBuildsImageAndContextInput(t *testing.T) {
	attachments := []Attachment{
		{Kind: KindImage, Path: "/tmp/screen.png"},
		{Kind: KindFile, Path: "/tmp/notes.txt"},
	}

	imagePath := SelectPrimaryImagePath(attachments)
	if imagePath != "/tmp/screen.png" {
		t.Fatalf("SelectPrimaryImagePath() = %q, want image path", imagePath)
	}
	base := ResolveDispatchBasePrompt("", imagePath, len(attachments))
	if base != "Please analyze this image." {
		t.Fatalf("ResolveDispatchBasePrompt() = %q, want image fallback", base)
	}
	input := BuildDispatchInput(base, []string{"[Attached file: notes.txt]\nbody"})
	if !strings.Contains(input, "Attached context:") {
		t.Fatalf("BuildDispatchInput() = %q, want context section", input)
	}
	display := BuildDispatchDisplay("", base, attachments)
	if !strings.Contains(display, "- image: screen.png") || !strings.Contains(display, "- file: notes.txt") {
		t.Fatalf("BuildDispatchDisplay() = %q, want attachment list", display)
	}
}

func TestDispatchPolicyUsesAttachmentFallbackWhenNoPayloadOrImage(t *testing.T) {
	base := ResolveDispatchBasePrompt("", "", 1)
	if base != "添付ファイルを確認して、要点をまとめてください。" {
		t.Fatalf("ResolveDispatchBasePrompt() = %q, want attachment fallback", base)
	}
	input := BuildDispatchInput("", []string{"[Attached file: notes.txt]\nbody"})
	if !strings.HasPrefix(input, "以下の添付コンテキストを確認してください。") {
		t.Fatalf("BuildDispatchInput() = %q, want context fallback prompt", input)
	}
}
