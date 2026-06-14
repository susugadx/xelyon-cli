package commandcatalog

import (
	"strings"
	"testing"
)

func TestMatchPrefixMatchesNameAliasAndSubcommand(t *testing.T) {
	tests := []struct {
		prefix string
		want   string
	}{
		{prefix: "/co", want: "/copy"},
		{prefix: "/q", want: "/exit"},
		{prefix: "/config m", want: "/config"},
		{prefix: "/skills s", want: "/skills"},
		{prefix: "/skills u", want: "/skills"},
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
	if cmd, ok := Find("/think"); !ok || cmd.Name != "/think" || !cmd.HiddenFromHelp || cmd.Discoverable {
		t.Fatalf("Find(/think) = %#v, %v, want hidden compatibility command", cmd, ok)
	}
	if cmd, ok := Find("/h"); !ok || cmd.Name != "/help" {
		t.Fatalf("Find(/h) = %#v, %v, want /help alias owner", cmd, ok)
	}
	if _, ok := Find("/missing"); ok {
		t.Fatal("Find(/missing) ok = true, want false")
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

	for _, name := range []string{"/config", "/copy", "/exit", "/provider", "/model"} {
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
		wantRawArgs    bool
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
			wantWithArgsOK: true,
			wantAction:     TUILocalActionOpenReview,
			wantOwner:      CommandOwnerTUIRouter,
			wantRawArgs:    true,
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
			name:           "/provider",
			withArgsInput:  "/provider openai",
			wantWithArgsOK: false,
			wantAction:     TUILocalActionOpenProviderPicker,
			wantOwner:      CommandOwnerAgent,
		},
		{
			name:           "/model",
			withArgsInput:  "/model gpt-5.4",
			wantWithArgsOK: false,
			wantAction:     TUILocalActionOpenModelPicker,
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
			if got := cmd.UsesRawTUILocalArgs(); got != tt.wantRawArgs {
				t.Fatalf("%s UsesRawTUILocalArgs() = %v, want %v", tt.name, got, tt.wantRawArgs)
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
			wantDesc:    "Create project AGENTS.md guidance file",
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
	if cmd.EffectiveCategoryDisplayLabel() != "session" {
		t.Fatalf("/copy category display = %q, want session", cmd.EffectiveCategoryDisplayLabel())
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
	if empty.EffectiveCategoryDisplayLabel() != "other" {
		t.Fatalf("empty category display = %q, want other", empty.EffectiveCategoryDisplayLabel())
	}
	if empty.EffectiveSortWeight() != 1000 {
		t.Fatalf("empty sort weight = %d, want 1000", empty.EffectiveSortWeight())
	}
}

func TestCommandCategoryDisplayLabelUsesSpecificLLMCommandLabels(t *testing.T) {
	for _, tt := range []struct {
		name string
		want string
	}{
		{name: "/model", want: "model"},
		{name: "/provider", want: "provider"},
		{name: "/thinking", want: "thinking"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd, ok := Find(tt.name)
			if !ok {
				t.Fatalf("Find(%s) ok = false, want true", tt.name)
			}
			if cmd.EffectiveCategory() != CommandCategoryModel {
				t.Fatalf("%s internal category = %q, want model", tt.name, cmd.EffectiveCategory())
			}
			if got := cmd.EffectiveCategoryDisplayLabel(); got != tt.want {
				t.Fatalf("%s category display = %q, want %q", tt.name, got, tt.want)
			}
		})
	}

	if got := (CommandInfo{Category: CommandCategoryModel}).EffectiveCategoryDisplayLabel(); got != "llm" {
		t.Fatalf("generic model category display = %q, want llm", got)
	}
}

func TestLSPCommandRemovedFromCatalog(t *testing.T) {
	if _, ok := Find("/lsp"); ok {
		t.Fatal("Find(/lsp) ok = true, want false after legacy command removal")
	}
}

func TestCatalogCommandsDeclareSurfacePolicy(t *testing.T) {
	for _, cmd := range Commands {
		if len(cmd.Surfaces) == 0 {
			t.Fatalf("%s should explicitly declare its surface policy", cmd.Name)
		}
	}
}

func TestValidateCommands(t *testing.T) {
	if err := ValidateCommands(Commands); err != nil {
		t.Fatalf("ValidateCommands() error = %v", err)
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
