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
		{prefix: "/rev", want: "/review"},
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

func TestReviewCommandMetadata(t *testing.T) {
	matches := MatchPrefix("/review")
	if len(matches) == 0 {
		t.Fatal("MatchPrefix(/review) returned no matches")
	}
	if matches[0].Name != "/review" {
		t.Fatalf("MatchPrefix(/review)[0].Name = %q, want /review", matches[0].Name)
	}
	if matches[0].Description != "review my current changes and find issues" {
		t.Fatalf("Description = %q", matches[0].Description)
	}
	if !matches[0].SupportsSurface(CommandSurfaceTUI) {
		t.Fatal("/review should support TUI surface")
	}
	if matches[0].SupportsSurface(CommandSurfaceClassic) {
		t.Fatal("/review should not support classic surface")
	}
	if matches[0].EffectiveLifecycle() != CommandLifecyclePreview {
		t.Fatalf("Lifecycle = %q, want %q", matches[0].EffectiveLifecycle(), CommandLifecyclePreview)
	}
	if matches[0].EffectiveCategory() != CommandCategoryReview {
		t.Fatalf("Category = %q, want %q", matches[0].EffectiveCategory(), CommandCategoryReview)
	}
	if !matches[0].Discoverable {
		t.Fatal("/review should be discoverable")
	}
	if matches[0].EffectiveSortWeight() != 10 {
		t.Fatalf("SortWeight = %d, want 10", matches[0].EffectiveSortWeight())
	}
}

func TestFindMatchesNameAndAlias(t *testing.T) {
	if cmd, ok := Find("/status"); !ok || cmd.Name != "/status" {
		t.Fatalf("Find(/status) = %#v, %v, want /status", cmd, ok)
	}
	if cmd, ok := Find("/stats"); !ok || cmd.Name != "/status" {
		t.Fatalf("Find(/stats) = %#v, %v, want /status alias owner", cmd, ok)
	}
	if _, ok := Find("/missing"); ok {
		t.Fatal("Find(/missing) ok = true, want false")
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
	tuiHelp := RenderCommandsTextForSurface(CommandSurfaceTUI)
	if !strings.Contains(tuiHelp, "/review") {
		t.Fatalf("TUI help should include /review:\n%s", tuiHelp)
	}
	if !strings.Contains(tuiHelp, "/init") {
		t.Fatalf("TUI help should include /init:\n%s", tuiHelp)
	}
	if !strings.Contains(tuiHelp, "/project") {
		t.Fatalf("TUI help should include /project:\n%s", tuiHelp)
	}
}

func TestDiscoverableCommandsForTUISurface(t *testing.T) {
	commands := DiscoverableCommandsForSurface(CommandSurfaceTUI)
	if len(commands) < 4 {
		t.Fatalf("DiscoverableCommandsForSurface(TUI) returned %d commands, want at least 4", len(commands))
	}
	gotLeading := []string{commands[0].Name, commands[1].Name, commands[2].Name, commands[3].Name}
	wantLeading := []string{"/review", "/model", "/config", "/copy"}
	for i := range wantLeading {
		if gotLeading[i] != wantLeading[i] {
			t.Fatalf("leading discoverable commands = %#v, want prefix %#v", gotLeading, wantLeading)
		}
	}

	for _, hidden := range []string{"/version", "/help"} {
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
	if got := MatchDiscoverablePrefixForSurface("/project", CommandSurfaceTUI); len(got) != 1 || got[0].Name != "/project" {
		t.Fatalf("TUI discoverable /project = %#v, want /project", got)
	}
}

func TestDefaultCommandMetadata(t *testing.T) {
	matches := MatchPrefix("/copy")
	if len(matches) == 0 {
		t.Fatal("MatchPrefix(/copy) returned no matches")
	}
	cmd := matches[0]
	if !cmd.SupportsSurface(CommandSurfaceTUI) || !cmd.SupportsSurface(CommandSurfaceClassic) {
		t.Fatalf("/copy default surfaces should include TUI and classic, got %#v", cmd.Surfaces)
	}
	if cmd.EffectiveLifecycle() != CommandLifecycleStable {
		t.Fatalf("/copy lifecycle = %q, want stable", cmd.EffectiveLifecycle())
	}
	if cmd.EffectiveCategory() != CommandCategorySession {
		t.Fatalf("/copy category = %q, want session", cmd.EffectiveCategory())
	}

	empty := CommandInfo{}
	if empty.EffectiveCategory() != CommandCategoryOther {
		t.Fatalf("empty category = %q, want other", empty.EffectiveCategory())
	}
	if empty.EffectiveSortWeight() != 1000 {
		t.Fatalf("empty sort weight = %d, want 1000", empty.EffectiveSortWeight())
	}
}

func containsCommandName(commands []CommandInfo, name string) bool {
	for _, cmd := range commands {
		if cmd.Name == name {
			return true
		}
	}
	return false
}
