package commandcatalog

import "testing"

func TestDiscoverableCommandsForTUISurface(t *testing.T) {
	commands := DiscoverableCommandsForSurface(CommandSurfaceTUI)
	if len(commands) < 4 {
		t.Fatalf("DiscoverableCommandsForSurface(TUI) returned %d commands, want at least 4", len(commands))
	}
	gotNames := commandNames(commands)
	wantNames := []string{
		"/model",
		"/provider",
		"/thinking",
		"/status",
		"/tokens",
		"/ledger",
		"/review",
		"/rawoutputs",
		"/project",
		"/setup",
		"/config",
		"/skills",
		"/copy",
		"/attach",
		"/detach",
		"/detach-all",
		"/compress",
		"/plan",
		"/save",
		"/load",
		"/sessions",
		"/clear",
		"/history",
		"/init",
		"/exit",
	}
	if len(gotNames) != len(wantNames) {
		t.Fatalf("discoverable commands = %#v, want %#v", gotNames, wantNames)
	}
	for i := range wantNames {
		if gotNames[i] != wantNames[i] {
			t.Fatalf("discoverable commands = %#v, want %#v", gotNames, wantNames)
		}
	}

	for _, hidden := range []string{"/providers", "/use", "/think", "/version", "/help"} {
		if containsCommandName(commands, hidden) {
			t.Fatalf("%s should not be TUI-discoverable", hidden)
		}
	}
}

func TestDiscoverablePrefixFiltering(t *testing.T) {
	if got := MatchDiscoverablePrefixForSurface("/review", CommandSurfaceClassic); len(got) != 0 {
		t.Fatalf("classic discoverable /review = %#v, want no matches", got)
	}
	if got := MatchDiscoverablePrefixForSurface("/review", CommandSurfaceTUI); len(got) != 1 || got[0].Name != "/review" {
		t.Fatalf("TUI discoverable /review = %#v, want /review", got)
	}
	if got := MatchDiscoverablePrefixForSurface("/init", CommandSurfaceTUI); len(got) != 1 || got[0].Name != "/init" {
		t.Fatalf("TUI discoverable /init = %#v, want /init", got)
	}
	if got := MatchDiscoverablePrefixForSurface("/version", CommandSurfaceTUI); len(got) != 0 {
		t.Fatalf("TUI discoverable /version = %#v, want no matches", got)
	}
	if got := MatchDiscoverablePrefixForSurface("/help", CommandSurfaceTUI); len(got) != 0 {
		t.Fatalf("TUI discoverable /help = %#v, want no matches", got)
	}
	if got := MatchDiscoverablePrefixForSurface("/lsp", CommandSurfaceTUI); len(got) != 0 {
		t.Fatalf("TUI discoverable /lsp = %#v, want no matches", got)
	}
	if got := MatchDiscoverablePrefixForSurface("/project", CommandSurfaceTUI); len(got) != 1 || got[0].Name != "/project" {
		t.Fatalf("TUI discoverable /project = %#v, want /project", got)
	}
	if got := MatchDiscoverablePrefixForSurface("/project", CommandSurfaceClassic); len(got) != 0 {
		t.Fatalf("classic discoverable /project = %#v, want no matches", got)
	}
	if got := MatchDiscoverablePrefixForSurface("/setup", CommandSurfaceTUI); len(got) != 1 || got[0].Name != "/setup" {
		t.Fatalf("TUI discoverable /setup = %#v, want /setup", got)
	}
	if got := MatchDiscoverablePrefixForSurface("/use", CommandSurfaceTUI); len(got) != 0 {
		t.Fatalf("TUI discoverable /use = %#v, want no matches", got)
	}
	if got := MatchDiscoverablePrefixForSurface("/provider", CommandSurfaceTUI); len(got) != 1 || got[0].Name != "/provider" {
		t.Fatalf("TUI discoverable /provider = %#v, want /provider", got)
	}
}
