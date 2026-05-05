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

	got := FindExistingCommands(content)
	for _, command := range []string{"/detach", "/detach-all", "/status", "/stats"} {
		if !got[command] {
			t.Fatalf("FindExistingCommands() missing %q from %#v", command, got)
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

	missing := MissingCommands(commandcatalog.Commands, documented)
	if len(missing) != 1 || missing[0].Name != "/skills" {
		t.Fatalf("MissingCommands() = %#v, want only /skills", missing)
	}
}

func TestMissingCommandsTreatsAliasHeadingAsDocumented(t *testing.T) {
	commands := []commandcatalog.CommandInfo{
		{Name: "/exit", Aliases: []string{"/quit", "/q"}},
	}
	documented := map[string]bool{"/quit": true}

	if missing := MissingCommands(commands, documented); len(missing) != 0 {
		t.Fatalf("MissingCommands() = %#v, want none", missing)
	}
}
