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

func TestCommandNormalizesCommandTokenAndBareMatching(t *testing.T) {
	cmd := NewCommand("/CONFIG", "payload")

	if !cmd.IsBare("/config") {
		t.Fatal("command token should be normalized and bare /config")
	}
	if !cmd.Matches("/config") {
		t.Fatal("command should match normalized /config")
	}
	if cmd.Input != "/CONFIG" {
		t.Fatalf("Input = %q, want %q", cmd.Input, "/CONFIG")
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
	cmd := NewCommand("/quit now", "payload")
	if cmd.IsBare("/quit") {
		t.Fatal("command with args should not be bare")
	}
	if !cmd.Matches("/quit") {
		t.Fatal("command token should be used as resolved name")
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
	cmd := NewCommand(`/attach "screenshots/error shot.png"`, "payload")
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
	cmd := NewCommand(`/attach "foo`, "payload")
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

func TestCommandRawArgTextKeepsCommandRemainder(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "bare command",
			input: "/review",
			want:  "",
		},
		{
			name:  "raw instructions",
			input: "/review focus on regressions",
			want:  "focus on regressions",
		},
		{
			name:  "quoted remainder keeps quotes",
			input: `/review "focus on regressions"`,
			want:  `"focus on regressions"`,
		},
		{
			name:  "extra spacing trims only around remainder",
			input: "/review   focus on regressions  ",
			want:  "focus on regressions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCommand(tt.input, tt.input)
			if got := cmd.RawArgText(); got != tt.want {
				t.Fatalf("RawArgText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSuggestionsMatchesCommandAndAlias(t *testing.T) {
	matches := Suggestions("/q")
	if len(matches) == 0 {
		t.Fatal("Suggestions(/q) returned no matches")
	}
	if matches[0].InsertText != "/exit" {
		t.Fatalf("first suggestion InsertText = %q, want /exit alias owner", matches[0].InsertText)
	}
}

func TestSuggestionsIncludesReviewCommand(t *testing.T) {
	matches := Suggestions("/review")
	if len(matches) == 0 {
		t.Fatal("Suggestions(/review) returned no matches")
	}
	if matches[0].InsertText != "/review" {
		t.Fatalf("first suggestion InsertText = %q, want /review", matches[0].InsertText)
	}
	if matches[0].Description != "Review current changes and find issues" {
		t.Fatalf("review description = %q", matches[0].Description)
	}
}

func TestSuggestionsExposeDisplayLabelAndCompletionText(t *testing.T) {
	matches := Suggestions("/thinking")
	if len(matches) != 1 {
		t.Fatalf("Suggestions(/thinking) returned %d matches, want 1", len(matches))
	}
	suggestion := matches[0]
	if suggestion.Label != "/thinking [on|off|level]" {
		t.Fatalf("Label = %q, want %q", suggestion.Label, "/thinking [on|off|level]")
	}
	if got := suggestion.CompletionText(false); got != "/thinking" {
		t.Fatalf("CompletionText(false) = %q, want /thinking", got)
	}
	if got := suggestion.CompletionText(true); got != "/thinking " {
		t.Fatalf("CompletionText(true) = %q, want '/thinking '", got)
	}
	if got := suggestion.SubmissionText(); got != "/thinking" {
		t.Fatalf("SubmissionText() = %q, want /thinking", got)
	}
}

func TestSuggestionSubmissionTextCanDifferFromCompletionText(t *testing.T) {
	suggestion := Suggestion{
		InsertText:  "/plan",
		SubmitText:  "/plan toggle",
		HasArgs:     true,
		Description: "toggle plan mode",
	}

	if got := suggestion.CompletionText(true); got != "/plan " {
		t.Fatalf("CompletionText(true) = %q, want '/plan '", got)
	}
	if got := suggestion.SubmissionText(); got != "/plan toggle" {
		t.Fatalf("SubmissionText() = %q, want '/plan toggle'", got)
	}
}

func TestSuggestionsPlanSubmitsToggleWithoutChangingCompletion(t *testing.T) {
	matches := Suggestions("/plan")
	if len(matches) != 1 {
		t.Fatalf("Suggestions(/plan) returned %d matches, want 1", len(matches))
	}
	suggestion := matches[0]
	if got := suggestion.InsertText; got != "/plan" {
		t.Fatalf("InsertText = %q, want /plan", got)
	}
	if got := suggestion.CompletionText(true); got != "/plan " {
		t.Fatalf("CompletionText(true) = %q, want '/plan '", got)
	}
	if got := suggestion.SubmissionText(); got != "/plan toggle" {
		t.Fatalf("SubmissionText() = %q, want '/plan toggle'", got)
	}
}

func TestSuggestionsCanonicalizeThinkAlias(t *testing.T) {
	matches := Suggestions("/think")
	if len(matches) != 1 {
		t.Fatalf("Suggestions(/think) returned %d matches, want 1", len(matches))
	}
	if got := matches[0].InsertText; got != "/thinking" {
		t.Fatalf("InsertText = %q, want /thinking", got)
	}
}

func TestSuggestionsThinkingArguments(t *testing.T) {
	matches := Suggestions("/thinking ")
	if len(matches) != 6 {
		t.Fatalf("Suggestions(/thinking ) returned %d matches, want 6", len(matches))
	}
	if got := matches[5].Label; got != "/thinking xhigh (max)" {
		t.Fatalf("xhigh label = %q, want /thinking xhigh (max)", got)
	}
	if got := matches[5].InsertText; got != "/thinking xhigh" {
		t.Fatalf("xhigh InsertText = %q, want /thinking xhigh", got)
	}
	if got := matches[5].CompletionText(true); got != "/thinking xhigh" {
		t.Fatalf("xhigh CompletionText(true) = %q, want /thinking xhigh", got)
	}
	if matches[0].SubmitOnEnter {
		t.Fatal("empty-prefix thinking argument suggestions should not submit on Enter")
	}
}

func TestSuggestionsThinkingAliasArguments(t *testing.T) {
	matches := Suggestions("/think x")
	if len(matches) != 1 {
		t.Fatalf("Suggestions(/think x) returned %d matches, want 1", len(matches))
	}
	if got := matches[0].InsertText; got != "/thinking xhigh" {
		t.Fatalf("InsertText = %q, want /thinking xhigh", got)
	}
	if !matches[0].SubmitOnEnter {
		t.Fatal("non-empty thinking argument suggestions should submit on Enter")
	}
}

func TestSuggestionsSkillsSubcommands(t *testing.T) {
	matches := Suggestions("/skills ")
	if len(matches) != 3 {
		t.Fatalf("Suggestions(/skills ) returned %d matches, want 3", len(matches))
	}

	wantLabels := []string{"/skills list", "/skills show <name>", "/skills doctor"}
	for i, want := range wantLabels {
		if matches[i].Label != want {
			t.Fatalf("matches[%d].Label = %q, want %q", i, matches[i].Label, want)
		}
		if !matches[i].SubmitOnEnter {
			t.Fatalf("matches[%d].SubmitOnEnter = false, want true", i)
		}
	}

	if got := matches[1].InsertText; got != "/skills show" {
		t.Fatalf("show InsertText = %q, want /skills show", got)
	}
	if !matches[1].HasArgs {
		t.Fatal("show suggestion should keep HasArgs for Tab trailing space")
	}
	if got := matches[1].CompletionText(true); got != "/skills show " {
		t.Fatalf("show CompletionText(true) = %q, want '/skills show '", got)
	}
}

func TestSuggestionsSkillsSubcommandPrefix(t *testing.T) {
	matches := Suggestions("/skills d")
	if len(matches) != 1 {
		t.Fatalf("Suggestions(/skills d) returned %d matches, want 1", len(matches))
	}
	if got := matches[0].InsertText; got != "/skills doctor" {
		t.Fatalf("InsertText = %q, want /skills doctor", got)
	}
	if got := matches[0].Description; got != "Show parsing/duplicate diagnostics" {
		t.Fatalf("Description = %q", got)
	}
}

func TestSuggestionsDoNotExposeConfigSubcommandsYet(t *testing.T) {
	if matches := Suggestions("/config "); len(matches) != 0 {
		t.Fatalf("Suggestions(/config ) = %#v, want no matches", matches)
	}
}

func TestSuggestionsSortsDiscoverableCommands(t *testing.T) {
	matches := Suggestions("/")
	if len(matches) < 4 {
		t.Fatalf("Suggestions(/) returned %d matches, want at least 4", len(matches))
	}
	got := []string{matches[0].InsertText, matches[1].InsertText, matches[2].InsertText, matches[3].InsertText}
	want := []string{"/model", "/provider", "/providers", "/thinking"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("leading suggestions = %#v, want prefix %#v", got, want)
		}
	}
}

func TestSuggestionsHidesNonDiscoverableCommands(t *testing.T) {
	for _, input := range []string{"/use", "/version", "/help", "/lsp"} {
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
	if matches[0].InsertText != "/init" {
		t.Fatalf("first suggestion InsertText = %q, want /init", matches[0].InsertText)
	}
}

func TestSuggestionsIncludesProjectCommand(t *testing.T) {
	matches := Suggestions("/project")
	if len(matches) != 1 {
		t.Fatalf("Suggestions(/project) returned %d matches, want 1", len(matches))
	}
	if matches[0].InsertText != "/project" {
		t.Fatalf("first suggestion InsertText = %q, want /project", matches[0].InsertText)
	}
}
