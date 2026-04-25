package commandcatalog

import (
	"strings"
	"testing"
)

func TestRenderCommandsTextIncludesSubcommands(t *testing.T) {
	got := RenderCommandsText()
	for _, fragment := range []string{
		"Commands:\n",
		"/exit, /quit, /q",
		"/config show - Show all settings with diff from defaults",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("RenderCommandsText() missing %q\n%s", fragment, got)
		}
	}
}

func TestMatchPrefixMatchesNameAliasAndSubcommand(t *testing.T) {
	tests := []struct {
		prefix string
		want   string
	}{
		{prefix: "/co", want: "/copy"},
		{prefix: "/q", want: "/exit"},
		{prefix: "/config m", want: "/config"},
	}

	for _, tt := range tests {
		got := MatchPrefix(tt.prefix)
		if len(got) == 0 {
			t.Fatalf("MatchPrefix(%q) returned no matches", tt.prefix)
		}
		if got[0].Name != tt.want {
			t.Fatalf("MatchPrefix(%q)[0].Name = %q, want %q", tt.prefix, got[0].Name, tt.want)
		}
	}
}
