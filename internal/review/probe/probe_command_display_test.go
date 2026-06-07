package probe

import "testing"

func TestFormatProbeCommandPreservesArgumentBoundaries(t *testing.T) {
	got := FormatProbeCommand("customtool", []string{"foo bar", "--flag=a;b", "$(printf token)"})
	want := `customtool "foo bar" "--flag=a;b" "$(printf token)"`
	if got != want {
		t.Fatalf("FormatProbeCommand() = %q, want %q", got, want)
	}
}
