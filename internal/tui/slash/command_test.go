package slash

import "testing"

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
}

func TestCommandWithArgsIsNotBare(t *testing.T) {
	cmd := NewCommand("/quit now", "payload", nil)
	if cmd.IsBare("/quit") {
		t.Fatal("command with args should not be bare")
	}
	if !cmd.Matches("/quit") {
		t.Fatal("nil resolver should use the command token as resolved name")
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
