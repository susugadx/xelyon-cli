package transcript

import (
	"testing"
	"time"
)

func TestNormalizeLine(t *testing.T) {
	if got := NormalizeLine("ok\r"); got != "ok" {
		t.Fatalf("NormalizeLine(CR) = %q, want ok", got)
	}
	if got := NormalizeLine("a✅️b"); got != "a✅b" {
		t.Fatalf("NormalizeLine(VS16) = %q, want %q", got, "a✅b")
	}
}

func TestMessageLines_User(t *testing.T) {
	got := Lines(Message{
		Role:      "user",
		Content:   "alpha\nbeta",
		Timestamp: time.Date(2026, 5, 7, 15, 4, 0, 0, time.UTC),
	})
	want := []string{"── user · 15:04 · now ──", "│ > alpha", "│ > beta"}
	if len(got) != len(want) {
		t.Fatalf("MessageLines() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("MessageLines()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestMessageLines_ConversationRolesUseSeparatorAndGutter(t *testing.T) {
	tests := []struct {
		role      string
		separator string
	}{
		{role: "assistant", separator: "── assistant · 15:04 · now ──"},
		{role: "system_info", separator: "── system · 15:04 · now ──"},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			got := Lines(Message{
				Role:      tt.role,
				Content:   "alpha\nbeta",
				Timestamp: time.Date(2026, 5, 7, 15, 4, 0, 0, time.UTC),
			})
			want := []string{tt.separator, "│ alpha", "│ beta"}
			if len(got) != len(want) {
				t.Fatalf("MessageLines() len = %d, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("MessageLines()[%d] = %q, want %q", i, got[i], want[i])
				}
			}
		})
	}
}

func TestMessageLines_NonConversationRoleKeepsRawLines(t *testing.T) {
	got := MessageLines("tool_header", "alpha\nbeta")
	want := []string{"alpha", "beta"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("MessageLines()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestMessageLines_AssistantChunkKeepsRawLines(t *testing.T) {
	got := MessageLines("assistant_chunk", "alpha\nbeta")
	want := []string{"alpha", "beta"}
	if len(got) != len(want) {
		t.Fatalf("MessageLines() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("MessageLines()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
