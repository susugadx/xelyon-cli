package transcript

import (
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

func TestAssistantBodyStylesMarkdownStructureWithoutChangingPlainText(t *testing.T) {
	content := strings.Join([]string{
		"# 方針",
		"- run `go test ./...`",
		"1. check `internal/tui/theme.go`",
		"> quote `value`",
		"Warning: retry `network`",
		"**結論** 白ベースで読む",
		"```go",
		`fmt.Println("ok")`,
		"```",
		"plain `code` text",
	}, "\n")

	got := Lines(Message{
		Role:      "assistant",
		Content:   content,
		Timestamp: time.Date(2026, 5, 7, 15, 4, 0, 0, time.UTC),
	})

	wantPlain := []string{
		"── assistant · 15:04 · now ──",
		"│ # 方針",
		"│ - run `go test ./...`",
		"│ 1. check `internal/tui/theme.go`",
		"│ > quote `value`",
		"│ Warning: retry `network`",
		"│ **結論** 白ベースで読む",
		"│ ```go",
		`│ fmt.Println("ok")`,
		"│ ```",
		"│ plain `code` text",
	}
	gotPlain := stripTranscriptANSI(got)
	if len(gotPlain) != len(wantPlain) {
		t.Fatalf("Lines() len = %d, want %d", len(gotPlain), len(wantPlain))
	}
	for i := range wantPlain {
		if gotPlain[i] != wantPlain[i] {
			t.Fatalf("Lines()[%d] plain = %q, want %q", i, gotPlain[i], wantPlain[i])
		}
	}

	assertLineHasStyle(t, got[1], theme.Transcript.AssistantHeading)
	assertLineHasStyle(t, got[2], theme.Transcript.AssistantListMarker)
	assertLineHasStyle(t, got[2], theme.Transcript.AssistantInlineCode)
	assertLineHasStyle(t, got[3], theme.Transcript.AssistantListMarker)
	assertLineHasStyle(t, got[4], theme.Transcript.AssistantQuote)
	assertLineHasStyle(t, got[5], theme.Transcript.AssistantWarning)
	assertLineHasStyle(t, got[6], theme.Transcript.AssistantHeading)
	assertLineHasStyle(t, got[7], theme.Transcript.AssistantCodeFence)
	assertLineHasStyle(t, got[8], theme.Transcript.AssistantCodeBlock)
	assertLineHasStyle(t, got[9], theme.Transcript.AssistantCodeFence)
	assertLineHasStyle(t, got[10], theme.Transcript.AssistantText)
	assertLineHasStyle(t, got[10], theme.Transcript.AssistantInlineCode)
}

func TestAssistantBodyPreservesExistingANSILines(t *testing.T) {
	const red = "\033[31m"
	const reset = "\033[0m"
	content := red + "red" + reset

	got := Lines(Message{
		Role:      "assistant",
		Content:   content,
		Timestamp: time.Date(2026, 5, 7, 15, 4, 0, 0, time.UTC),
	})

	if got[1] != "│ "+content {
		t.Fatalf("ANSI body line = %q, want raw content with gutter", got[1])
	}
	if strings.Contains(got[1], theme.Transcript.AssistantText) {
		t.Fatalf("existing ANSI line should not receive assistant text style: %q", got[1])
	}
}

func assertLineHasStyle(t *testing.T, line string, style string) {
	t.Helper()
	if style == "" {
		t.Fatal("style must not be empty")
	}
	if !strings.Contains(line, style) {
		t.Fatalf("line %q should contain style %q", line, style)
	}
}
