package transcript

import "testing"

func TestNormalizeLine(t *testing.T) {
	if got := NormalizeLine("ok\r"); got != "ok" {
		t.Fatalf("NormalizeLine(CR) = %q, want ok", got)
	}
	if got := NormalizeLine("a✅️b"); got != "a✅b" {
		t.Fatalf("NormalizeLine(VS16) = %q, want %q", got, "a✅b")
	}
}

func TestMessageLines_User(t *testing.T) {
	got := MessageLines("user", "alpha\nbeta")
	want := []string{"", "> alpha", "> beta", ""}
	if len(got) != len(want) {
		t.Fatalf("MessageLines() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("MessageLines()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestMessageLines_PlainRolesKeepRawLines(t *testing.T) {
	tests := []struct {
		role string
	}{
		{role: "assistant"},
		{role: "system_info"},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			got := MessageLines(tt.role, "alpha\nbeta")
			want := []string{"alpha", "beta"}
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
