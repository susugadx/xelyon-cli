package commandcatalog

import (
	"os"
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
	if matches[0].Description != "Review current changes and find issues" {
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
	if matches[0].EffectiveOwner() != CommandOwnerTUIRouter {
		t.Fatalf("Owner = %q, want %q", matches[0].EffectiveOwner(), CommandOwnerTUIRouter)
	}
	if !matches[0].Discoverable {
		t.Fatal("/review should be discoverable")
	}
	if matches[0].EffectiveSortWeight() != 70 {
		t.Fatalf("SortWeight = %d, want 70", matches[0].EffectiveSortWeight())
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
	if !strings.Contains(classicHelp, "/lsp") {
		t.Fatalf("classic help should include legacy /lsp:\n%s", classicHelp)
	}
	tuiHelp := RenderCommandsTextForSurface(CommandSurfaceTUI)
	if !strings.Contains(tuiHelp, "/review") {
		t.Fatalf("TUI help should include /review:\n%s", tuiHelp)
	}
	if strings.Contains(tuiHelp, "/lsp") {
		t.Fatalf("TUI help should not include classic-only /lsp:\n%s", tuiHelp)
	}
	if !strings.Contains(tuiHelp, "/init") {
		t.Fatalf("TUI help should include /init:\n%s", tuiHelp)
	}
	if !strings.Contains(tuiHelp, "/project") {
		t.Fatalf("TUI help should include /project:\n%s", tuiHelp)
	}
	for _, cmd := range []string{"/attach", "/detach", "/detach-all"} {
		if !strings.Contains(tuiHelp, cmd) {
			t.Fatalf("TUI help should include %s:\n%s", cmd, tuiHelp)
		}
	}
}

func TestTUILocalCommandOwnership(t *testing.T) {
	for _, name := range []string{"/review", "/project", "/attach", "/detach", "/detach-all"} {
		t.Run(name, func(t *testing.T) {
			cmd, ok := Find(name)
			if !ok {
				t.Fatalf("Find(%q) ok = false, want true", name)
			}
			if cmd.EffectiveOwner() != CommandOwnerTUIRouter {
				t.Fatalf("%s owner = %q, want %q", name, cmd.EffectiveOwner(), CommandOwnerTUIRouter)
			}
			if !cmd.SupportsSurface(CommandSurfaceTUI) {
				t.Fatalf("%s should support TUI surface", name)
			}
			if cmd.SupportsSurface(CommandSurfaceClassic) {
				t.Fatalf("%s should not support classic surface", name)
			}
		})
	}

	for _, name := range []string{"/config", "/copy", "/exit"} {
		t.Run(name, func(t *testing.T) {
			cmd, ok := Find(name)
			if !ok {
				t.Fatalf("Find(%q) ok = false, want true", name)
			}
			if cmd.EffectiveOwner() != CommandOwnerAgent {
				t.Fatalf("%s owner = %q, want %q", name, cmd.EffectiveOwner(), CommandOwnerAgent)
			}
			if !cmd.SupportsSurface(CommandSurfaceTUI) {
				t.Fatalf("%s should support TUI surface", name)
			}
			if !cmd.SupportsSurface(CommandSurfaceClassic) {
				t.Fatalf("%s should support classic surface", name)
			}
			if cmd.EffectiveTUILocalAction() == TUILocalActionNone {
				t.Fatalf("%s should declare TUI local action", name)
			}
		})
	}
}

func TestTUILocalArgPolicyAndAction(t *testing.T) {
	tests := []struct {
		name           string
		withArgsInput  string
		wantWithArgsOK bool
		wantAction     TUILocalAction
		wantOwner      CommandOwner
	}{
		{
			name:           "/attach",
			withArgsInput:  "/attach ./notes.txt",
			wantWithArgsOK: true,
			wantAction:     TUILocalActionManageAttachments,
			wantOwner:      CommandOwnerTUIRouter,
		},
		{
			name:           "/detach",
			withArgsInput:  "/detach 1",
			wantWithArgsOK: true,
			wantAction:     TUILocalActionManageAttachments,
			wantOwner:      CommandOwnerTUIRouter,
		},
		{
			name:           "/detach-all",
			withArgsInput:  "/detach-all now",
			wantWithArgsOK: true,
			wantAction:     TUILocalActionManageAttachments,
			wantOwner:      CommandOwnerTUIRouter,
		},
		{
			name:           "/review",
			withArgsInput:  "/review staged",
			wantWithArgsOK: false,
			wantAction:     TUILocalActionOpenReview,
			wantOwner:      CommandOwnerTUIRouter,
		},
		{
			name:           "/project",
			withArgsInput:  "/project rules",
			wantWithArgsOK: false,
			wantAction:     TUILocalActionOpenProject,
			wantOwner:      CommandOwnerTUIRouter,
		},
		{
			name:           "/config",
			withArgsInput:  "/config show",
			wantWithArgsOK: false,
			wantAction:     TUILocalActionOpenConfig,
			wantOwner:      CommandOwnerAgent,
		},
		{
			name:           "/copy",
			withArgsInput:  "/copy code",
			wantWithArgsOK: false,
			wantAction:     TUILocalActionCopyMouseSelection,
			wantOwner:      CommandOwnerAgent,
		},
		{
			name:           "/exit",
			withArgsInput:  "/exit now",
			wantWithArgsOK: true,
			wantAction:     TUILocalActionQuit,
			wantOwner:      CommandOwnerAgent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, ok := Find(tt.name)
			if !ok {
				t.Fatalf("Find(%q) ok = false, want true", tt.name)
			}
			if cmd.EffectiveOwner() != tt.wantOwner {
				t.Fatalf("%s owner = %q, want %q", tt.name, cmd.EffectiveOwner(), tt.wantOwner)
			}
			if cmd.EffectiveTUILocalAction() != tt.wantAction {
				t.Fatalf("%s local action = %q, want %q", tt.name, cmd.EffectiveTUILocalAction(), tt.wantAction)
			}
			withArgs := strings.Fields(tt.withArgsInput)
			if got := cmd.AcceptsTUILocalArgs(withArgs[1:]); got != tt.wantWithArgsOK {
				t.Fatalf("%s AcceptsTUILocalArgs(%q) = %v, want %v", tt.name, tt.withArgsInput, got, tt.wantWithArgsOK)
			}
			if tt.name == "/copy" {
				if cmd.AcceptsTUILocalContext(TUILocalContext{HasMouseSelection: false}) {
					t.Fatalf("%s should require mouse selection", tt.name)
				}
				if !cmd.AcceptsTUILocalContext(TUILocalContext{HasMouseSelection: true}) {
					t.Fatalf("%s should accept context with mouse selection", tt.name)
				}
			}
		})
	}
}

func TestAttachDescriptionMentionsCombinedLimit(t *testing.T) {
	cmd, ok := Find("/attach")
	if !ok {
		t.Fatal("Find(/attach) ok = false, want true")
	}
	if !strings.Contains(cmd.Description, "up to 12 attachments per draft") {
		t.Fatalf("/attach description should mention limit, got %q", cmd.Description)
	}
}

func TestAttachLimitDocumentationConsistency(t *testing.T) {
	paths := []string{
		"../../README.md",
		"../../docs/commands.md",
		"../../docs/usage.md",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile(%q) error = %v", path, err)
			}
			text := string(body)
			if !strings.Contains(text, "/attach") {
				t.Fatalf("%s should mention /attach", path)
			}
			if !strings.Contains(text, "最大12件") && !strings.Contains(text, "最大 12 件") {
				t.Fatalf("%s should mention attachment limit (12)", path)
			}
		})
	}
}

func TestConfigProjectInitCommandBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		wantDesc    string
		wantTUI     bool
		wantClassic bool
		wantOwner   CommandOwner
	}{
		{
			name:        "/config",
			wantDesc:    "Edit global config.yaml settings",
			wantTUI:     true,
			wantClassic: true,
			wantOwner:   CommandOwnerAgent,
		},
		{
			name:        "/init",
			wantDesc:    "Create xelyon.yaml project template",
			wantTUI:     true,
			wantClassic: true,
			wantOwner:   CommandOwnerAgent,
		},
		{
			name:        "/project",
			wantDesc:    "Edit project xelyon.yaml interactively",
			wantTUI:     true,
			wantClassic: false,
			wantOwner:   CommandOwnerTUIRouter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, ok := Find(tt.name)
			if !ok {
				t.Fatalf("Find(%q) ok = false, want true", tt.name)
			}
			if cmd.Description != tt.wantDesc {
				t.Fatalf("%s description = %q, want %q", tt.name, cmd.Description, tt.wantDesc)
			}
			if cmd.SupportsSurface(CommandSurfaceTUI) != tt.wantTUI {
				t.Fatalf("%s TUI support = %v, want %v", tt.name, cmd.SupportsSurface(CommandSurfaceTUI), tt.wantTUI)
			}
			if cmd.SupportsSurface(CommandSurfaceClassic) != tt.wantClassic {
				t.Fatalf("%s classic support = %v, want %v", tt.name, cmd.SupportsSurface(CommandSurfaceClassic), tt.wantClassic)
			}
			if cmd.EffectiveOwner() != tt.wantOwner {
				t.Fatalf("%s owner = %q, want %q", tt.name, cmd.EffectiveOwner(), tt.wantOwner)
			}
		})
	}
}

func TestLSPCommandIsClassicOnlyLegacyDiagnostic(t *testing.T) {
	cmd, ok := Find("/lsp")
	if !ok {
		t.Fatal("Find(/lsp) ok = false, want true")
	}
	if cmd.SupportsSurface(CommandSurfaceTUI) {
		t.Fatal("/lsp should not support TUI surface")
	}
	if !cmd.SupportsSurface(CommandSurfaceClassic) {
		t.Fatal("/lsp should support classic surface")
	}
	if cmd.Discoverable {
		t.Fatal("/lsp should not be discoverable")
	}
}

func TestDiscoverableCommandsForTUISurface(t *testing.T) {
	commands := DiscoverableCommandsForSurface(CommandSurfaceTUI)
	if len(commands) < 4 {
		t.Fatalf("DiscoverableCommandsForSurface(TUI) returned %d commands, want at least 4", len(commands))
	}
	gotNames := commandNames(commands)
	wantNames := []string{
		"/model",
		"/use",
		"/providers",
		"/think",
		"/status",
		"/tokens",
		"/review",
		"/project",
		"/config",
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

	for _, hidden := range []string{"/version", "/help", "/lsp"} {
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
}

func TestDefaultCommandMetadata(t *testing.T) {
	matches := MatchPrefix("/copy")
	if len(matches) == 0 {
		t.Fatal("MatchPrefix(/copy) returned no matches")
	}
	cmd := matches[0]
	if len(cmd.Surfaces) == 0 {
		t.Fatal("/copy should explicitly declare legacy classic support")
	}
	if !cmd.SupportsSurface(CommandSurfaceTUI) || !cmd.SupportsSurface(CommandSurfaceClassic) {
		t.Fatalf("/copy surfaces should include TUI and classic, got %#v", cmd.Surfaces)
	}
	if cmd.EffectiveLifecycle() != CommandLifecycleStable {
		t.Fatalf("/copy lifecycle = %q, want stable", cmd.EffectiveLifecycle())
	}
	if cmd.EffectiveCategory() != CommandCategorySession {
		t.Fatalf("/copy category = %q, want session", cmd.EffectiveCategory())
	}
	if cmd.EffectiveOwner() != CommandOwnerAgent {
		t.Fatalf("/copy owner = %q, want %q", cmd.EffectiveOwner(), CommandOwnerAgent)
	}

	empty := CommandInfo{}
	if !empty.SupportsSurface(CommandSurfaceTUI) {
		t.Fatal("empty/default command should support TUI surface")
	}
	if empty.SupportsSurface(CommandSurfaceClassic) {
		t.Fatal("empty/default command should not support classic surface")
	}
	if empty.EffectiveCategory() != CommandCategoryOther {
		t.Fatalf("empty category = %q, want other", empty.EffectiveCategory())
	}
	if empty.EffectiveSortWeight() != 1000 {
		t.Fatalf("empty sort weight = %d, want 1000", empty.EffectiveSortWeight())
	}
}

func TestCatalogCommandsDeclareSurfacePolicy(t *testing.T) {
	for _, cmd := range Commands {
		if len(cmd.Surfaces) == 0 {
			t.Fatalf("%s should explicitly declare its surface policy", cmd.Name)
		}
	}
}

func TestClassicSurfaceIsExplicitStableFallbackOrClassicOnly(t *testing.T) {
	for _, cmd := range Commands {
		if !cmd.SupportsSurface(CommandSurfaceClassic) {
			continue
		}
		if len(cmd.Surfaces) == 0 {
			t.Fatalf("%s supports classic without explicit surface policy", cmd.Name)
		}
		if !cmd.SupportsSurface(CommandSurfaceTUI) {
			if cmd.Discoverable {
				t.Fatalf("%s is classic-only and should not be discoverable", cmd.Name)
			}
			if cmd.EffectiveOwner() != CommandOwnerAgent {
				t.Fatalf("%s is classic-only with owner %q, want %q", cmd.Name, cmd.EffectiveOwner(), CommandOwnerAgent)
			}
			continue
		}
		if cmd.EffectiveOwner() == CommandOwnerTUIRouter {
			if cmd.EffectiveTUILocalAction() == TUILocalActionNone {
				t.Fatalf("%s is TUI-router owned without TUI local action", cmd.Name)
			}
			continue
		}
		if cmd.EffectiveLifecycle() != CommandLifecycleStable {
			t.Fatalf("%s supports classic with lifecycle %q, want stable", cmd.Name, cmd.EffectiveLifecycle())
		}
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

func commandNames(commands []CommandInfo) []string {
	names := make([]string, 0, len(commands))
	for _, cmd := range commands {
		names = append(names, cmd.Name)
	}
	return names
}
