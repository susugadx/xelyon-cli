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

func TestSurfaceFiltering(t *testing.T) {
	if got := MatchPrefixForSurface("/review", CommandSurfaceClassic); len(got) != 0 {
		t.Fatalf("classic MatchPrefixForSurface(/review) = %#v, want no matches", got)
	}
	if got := MatchPrefixForSurface("/review", CommandSurfaceTUI); len(got) != 1 || got[0].Name != "/review" {
		t.Fatalf("TUI MatchPrefixForSurface(/review) = %#v, want /review", got)
	}

	classicHelp := RenderCommandsTextForSurface(CommandSurfaceClassic)
	if strings.Contains(classicHelp, "/review") {
		t.Fatalf("classic help should not include /review:\n%s", classicHelp)
	}
	if strings.Contains(classicHelp, "/project") {
		t.Fatalf("classic help should not include TUI-only /project:\n%s", classicHelp)
	}
	if strings.Contains(classicHelp, "/attach") {
		t.Fatalf("classic help should not include TUI-only /attach:\n%s", classicHelp)
	}
	if strings.Contains(classicHelp, "/detach") {
		t.Fatalf("classic help should not include TUI-only /detach:\n%s", classicHelp)
	}
	if strings.Contains(classicHelp, "/detach-all") {
		t.Fatalf("classic help should not include TUI-only /detach-all:\n%s", classicHelp)
	}
	if strings.Contains(classicHelp, "/lsp") {
		t.Fatalf("classic help should not include removed /lsp:\n%s", classicHelp)
	}
	if strings.Contains(classicHelp, "/use") {
		t.Fatalf("classic help should not advertise compatibility /use:\n%s", classicHelp)
	}
	if !strings.Contains(classicHelp, "/provider") {
		t.Fatalf("classic help should include /provider:\n%s", classicHelp)
	}
	if !strings.Contains(classicHelp, "/ledger") {
		t.Fatalf("classic help should include /ledger:\n%s", classicHelp)
	}
	tuiHelp := RenderCommandsTextForSurface(CommandSurfaceTUI)
	if !strings.Contains(tuiHelp, "/review") {
		t.Fatalf("TUI help should include /review:\n%s", tuiHelp)
	}
	if strings.Contains(tuiHelp, "/lsp") {
		t.Fatalf("TUI help should not include removed /lsp:\n%s", tuiHelp)
	}
	if !strings.Contains(tuiHelp, "/init") {
		t.Fatalf("TUI help should include /init:\n%s", tuiHelp)
	}
	if !strings.Contains(tuiHelp, "/project") {
		t.Fatalf("TUI help should include /project:\n%s", tuiHelp)
	}
	if strings.Contains(tuiHelp, "/use") {
		t.Fatalf("TUI help should not advertise compatibility /use:\n%s", tuiHelp)
	}
	if !strings.Contains(tuiHelp, "/provider") {
		t.Fatalf("TUI help should include /provider:\n%s", tuiHelp)
	}
	if !strings.Contains(tuiHelp, "/ledger") {
		t.Fatalf("TUI help should include /ledger:\n%s", tuiHelp)
	}
	for _, cmd := range []string{"/attach", "/detach", "/detach-all"} {
		if !strings.Contains(tuiHelp, cmd) {
			t.Fatalf("TUI help should include %s:\n%s", cmd, tuiHelp)
		}
	}
}
