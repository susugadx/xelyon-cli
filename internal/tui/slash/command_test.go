package slash

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/commandruntime"
)

func TestTrimmedInput(t *testing.T) {
	input, ok := TrimmedInput("  /help  ")
	if !ok {
		t.Fatal("TrimmedInput() ok = false, want true")
	}
	if input != "/help" {
		t.Fatalf("TrimmedInput() input = %q, want %q", input, "/help")
	}

	if _, ok := TrimmedInput("hello"); ok {
		t.Fatal("TrimmedInput(non-command) ok = true, want false")
	}
}

func TestCommandAliasAndBareMatching(t *testing.T) {
	cmd := NewCommand("/c", "payload", func(name string) string {
		if name == "/c" {
			return "/config"
		}
		return name
	})

	if !cmd.IsBare("/config") {
		t.Fatal("alias command should be bare /config")
	}
	if !cmd.Matches("/config") {
		t.Fatal("alias command should match /config")
	}
	if cmd.Input != "/c" {
		t.Fatalf("Input = %q, want %q", cmd.Input, "/c")
	}
	if cmd.Payload != "payload" {
		t.Fatalf("Payload = %q, want %q", cmd.Payload, "payload")
	}
	if !cmd.ParseOK() {
		t.Fatal("ParseOK = false, want true")
	}
	if got := cmd.ParseStatus; got != commandruntime.SplitStatusOK {
		t.Fatalf("ParseStatus = %v, want %v", got, commandruntime.SplitStatusOK)
	}
	if got := len(cmd.Args); got != 0 {
		t.Fatalf("len(Args) = %d, want 0", got)
	}
}

func TestCommandWithArgsIsNotBare(t *testing.T) {
	cmd := NewCommand("/quit now", "payload", nil)
	if cmd.IsBare("/quit") {
		t.Fatal("command with args should not be bare")
	}
	if !cmd.Matches("/quit") {
		t.Fatal("nil resolver should use the command token as resolved name")
	}
	if !cmd.ParseOK() {
		t.Fatal("ParseOK = false, want true")
	}
	if got := cmd.ParseStatus; got != commandruntime.SplitStatusOK {
		t.Fatalf("ParseStatus = %v, want %v", got, commandruntime.SplitStatusOK)
	}
	if got := len(cmd.Args); got != 1 || cmd.Args[0] != "now" {
		t.Fatalf("Args = %#v, want [now]", cmd.Args)
	}
}

func TestCommandParsesQuotedArgs(t *testing.T) {
	cmd := NewCommand(`/attach "screenshots/error shot.png"`, "payload", nil)
	if !cmd.Matches("/attach") {
		t.Fatal("quoted arg command should match /attach")
	}
	if !cmd.ParseOK() {
		t.Fatal("ParseOK = false, want true")
	}
	if got := cmd.ParseStatus; got != commandruntime.SplitStatusOK {
		t.Fatalf("ParseStatus = %v, want %v", got, commandruntime.SplitStatusOK)
	}
	if got := len(cmd.Args); got != 1 {
		t.Fatalf("len(Args) = %d, want 1", got)
	}
	if got := cmd.Args[0]; got != "screenshots/error shot.png" {
		t.Fatalf("Args[0] = %q, want %q", got, "screenshots/error shot.png")
	}
}

func TestCommandUnterminatedQuoteMarkedInvalid(t *testing.T) {
	cmd := NewCommand(`/attach "foo`, "payload", nil)
	if cmd.ParseOK() {
		t.Fatal("ParseOK = true, want false")
	}
	if got := cmd.ParseStatus; got != commandruntime.SplitStatusUnterminatedQuote {
		t.Fatalf("ParseStatus = %v, want %v", got, commandruntime.SplitStatusUnterminatedQuote)
	}
	if !cmd.Matches("/attach") {
		t.Fatal("unterminated quote should still preserve command token")
	}
	if got := len(cmd.Args); got != 1 || cmd.Args[0] != "foo" {
		t.Fatalf("Args = %#v, want [foo]", cmd.Args)
	}
}

func TestSuggestionsMatchesCommandAndAlias(t *testing.T) {
	matches := Suggestions("/q")
	if len(matches) == 0 {
		t.Fatal("Suggestions(/q) returned no matches")
	}
	if matches[0].Name != "/exit" {
		t.Fatalf("first suggestion = %q, want /exit alias owner", matches[0].Name)
	}
}

func TestSuggestionsIncludesReviewCommand(t *testing.T) {
	matches := Suggestions("/review")
	if len(matches) == 0 {
		t.Fatal("Suggestions(/review) returned no matches")
	}
	if matches[0].Name != "/review" {
		t.Fatalf("first suggestion = %q, want /review", matches[0].Name)
	}
	if matches[0].Description != "Review current changes and find issues" {
		t.Fatalf("review description = %q", matches[0].Description)
	}
}

func TestSuggestionsSortsDiscoverableCommands(t *testing.T) {
	matches := Suggestions("/")
	if len(matches) < 4 {
		t.Fatalf("Suggestions(/) returned %d matches, want at least 4", len(matches))
	}
	got := []string{matches[0].Name, matches[1].Name, matches[2].Name, matches[3].Name}
	want := []string{"/model", "/use", "/providers", "/think"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("leading suggestions = %#v, want prefix %#v", got, want)
		}
	}
}

func TestSuggestionsHidesNonDiscoverableCommands(t *testing.T) {
	for _, input := range []string{"/version", "/help", "/lsp"} {
		if matches := Suggestions(input); len(matches) != 0 {
			t.Fatalf("Suggestions(%q) = %#v, want no matches", input, matches)
		}
	}
}

func TestSuggestionsIncludesInitCommand(t *testing.T) {
	matches := Suggestions("/init")
	if len(matches) != 1 {
		t.Fatalf("Suggestions(/init) returned %d matches, want 1", len(matches))
	}
	if matches[0].Name != "/init" {
		t.Fatalf("first suggestion = %q, want /init", matches[0].Name)
	}
}

func TestSuggestionsIncludesProjectCommand(t *testing.T) {
	matches := Suggestions("/project")
	if len(matches) != 1 {
		t.Fatalf("Suggestions(/project) returned %d matches, want 1", len(matches))
	}
	if matches[0].Name != "/project" {
		t.Fatalf("first suggestion = %q, want /project", matches[0].Name)
	}
}
