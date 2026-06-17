package commanddocs

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
)

func TestFindExistingCommandsSupportsHyphenatedHeadings(t *testing.T) {
	content := strings.Join([]string{
		"### `/detach`",
		"",
		"### `/detach-all`",
		"",
		"### `/status`, `/stats`",
	}, "\n")

	got := findExistingCommands(content)
	for _, command := range []string{"/detach", "/detach-all", "/status", "/stats"} {
		if !got[command] {
			t.Fatalf("findExistingCommands() missing %q from %#v", command, got)
		}
	}
}

func TestMissingCommandsDetectsSkillsWhenUndocumented(t *testing.T) {
	documented := make(map[string]bool)
	for _, cmd := range commandcatalog.Commands {
		if cmd.Name == "/skills" {
			continue
		}
		documented[cmd.Name] = true
	}

	missing := missingCommands(commandcatalog.Commands, documented)
	if len(missing) != 1 || missing[0].Name != "/skills" {
		t.Fatalf("missingCommands() = %#v, want only /skills", missing)
	}
}

func TestMissingCommandsTreatsAliasHeadingAsDocumented(t *testing.T) {
	commands := []commandcatalog.CommandInfo{
		{Name: "/exit", Aliases: []string{"/quit", "/q"}},
	}
	documented := map[string]bool{"/quit": true}

	if missing := missingCommands(commands, documented); len(missing) != 0 {
		t.Fatalf("missingCommands() = %#v, want none", missing)
	}
}

func TestMissingCommandsSkipsHiddenCompatibilityCommands(t *testing.T) {
	commands := []commandcatalog.CommandInfo{
		{Name: "/thinking"},
		{Name: "/think", HiddenFromHelp: true},
	}
	documented := map[string]bool{"/thinking": true}

	if missing := missingCommands(commands, documented); len(missing) != 0 {
		t.Fatalf("missingCommands() = %#v, want none", missing)
	}
}

func TestAppendMissingCommandSkeleton(t *testing.T) {
	commands := []commandcatalog.CommandInfo{
		{Name: "/known", Description: "known"},
		{Name: "/new", Description: "new command", Args: "<value>"},
	}
	content := "### `/known`\n\nknown docs\n"

	got, missing := AppendMissingCommandSkeleton(content, commands)
	if len(missing) != 1 || missing[0].Name != "/new" {
		t.Fatalf("missing commands = %#v, want /new", missing)
	}
	for _, expected := range []string{
		content,
		"## 未ドキュメント化コマンド（自動追加）",
		"### `/new`",
		"> /new <value>",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("updated content missing %q in:\n%s", expected, got)
		}
	}
}

func TestAppendMissingCommandSkeletonKeepsContentWhenComplete(t *testing.T) {
	commands := []commandcatalog.CommandInfo{{Name: "/known", Description: "known"}}
	content := "### `/known`\n\nknown docs\n"

	got, missing := AppendMissingCommandSkeleton(content, commands)
	if len(missing) != 0 {
		t.Fatalf("missing commands = %#v, want none", missing)
	}
	if got != content {
		t.Fatalf("content changed unexpectedly:\n%s", got)
	}
}
