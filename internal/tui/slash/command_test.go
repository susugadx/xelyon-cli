package slash

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
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
	if suggestion.Label != "/thinking <on|off|level>" {
		t.Fatalf("Label = %q, want %q", suggestion.Label, "/thinking <on|off|level>")
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
	if got := suggestion.CategoryDisplayLabel(); got != "thinking" {
		t.Fatalf("CategoryDisplayLabel() = %q, want thinking", got)
	}
	if !suggestion.ExpandOnEnter {
		t.Fatal("/thinking root suggestion should expand to argument suggestions on Enter")
	}
	if suggestion.SubmitOnEnter {
		t.Fatal("/thinking root suggestion should not submit on Enter; it should open argument suggestions")
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

func TestSuggestionCategoryDisplayLabelUsesExplicitLabelFirst(t *testing.T) {
	suggestion := Suggestion{
		Category:      commandcatalog.CommandCategoryModel,
		CategoryLabel: "provider",
	}
	if got := suggestion.CategoryDisplayLabel(); got != "provider" {
		t.Fatalf("CategoryDisplayLabel() = %q, want provider", got)
	}
	suggestion.HideCategory = true
	if got := suggestion.CategoryDisplayLabel(); got != "" {
		t.Fatalf("hidden CategoryDisplayLabel() = %q, want empty", got)
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

func TestSuggestionsCompleteThinkingPrefix(t *testing.T) {
	matches := Suggestions("/think")
	if len(matches) != 1 {
		t.Fatalf("Suggestions(/think) returned %d matches, want 1", len(matches))
	}
	if got := matches[0].InsertText; got != "/thinking" {
		t.Fatalf("InsertText = %q, want /thinking prefix completion", got)
	}
}

func TestSuggestionsThinkingArguments(t *testing.T) {
	matches := Suggestions("/thinking ")
	if len(matches) != 6 {
		t.Fatalf("Suggestions(/thinking ) returned %d matches, want 6", len(matches))
	}
	if got := matches[5].Label; got != "xhigh (max)" {
		t.Fatalf("xhigh label = %q, want xhigh (max)", got)
	}
	if got := matches[5].InsertText; got != "/thinking xhigh" {
		t.Fatalf("xhigh InsertText = %q, want /thinking xhigh", got)
	}
	if got := matches[5].CompletionText(true); got != "/thinking xhigh" {
		t.Fatalf("xhigh CompletionText(true) = %q, want /thinking xhigh", got)
	}
	if !matches[0].SubmitOnEnter {
		t.Fatal("empty-prefix thinking argument suggestions should submit on Enter")
	}
}

func TestSuggestionsThinkingAliasArgumentsRemoved(t *testing.T) {
	matches := Suggestions("/think x")
	if len(matches) != 0 {
		t.Fatalf("Suggestions(/think x) returned %#v, want no removed alias argument suggestions", matches)
	}
}

func TestSuggestionsSkillsSubcommands(t *testing.T) {
	root := Suggestions("/skills")
	if len(root) != 1 {
		t.Fatalf("Suggestions(/skills) returned %d matches, want 1", len(root))
	}
	if !root[0].ExpandOnEnter {
		t.Fatal("/skills root suggestion should expand to subcommand suggestions on Enter")
	}
	if root[0].SubmitOnEnter {
		t.Fatal("/skills root suggestion should not submit on Enter; /skills overview remains the executable overview command")
	}
	for _, fragment := range []string{
		"overview: Print skill catalog overview",
		"show <name>: Show SKILL.md body and resource listings",
		"suggest <text>: Preview ranked Skill Router recommendations",
		"usage: Show local Skill Router usage diagnostics",
		"doctor: Show parsing/duplicate diagnostics",
	} {
		if !strings.Contains(root[0].Detail, fragment) {
			t.Fatalf("/skills detail missing %q: %q", fragment, root[0].Detail)
		}
	}

	matches := Suggestions("/skills ")
	if len(matches) != 5 {
		t.Fatalf("Suggestions(/skills ) returned %d matches, want 5", len(matches))
	}

	wantLabels := []string{"overview", "show <name>", "suggest <text>", "usage", "doctor"}
	for i, want := range wantLabels {
		if matches[i].Label != want {
			t.Fatalf("matches[%d].Label = %q, want %q", i, matches[i].Label, want)
		}
		if want != "show <name>" && want != "suggest <text>" && !matches[i].SubmitOnEnter {
			t.Fatalf("matches[%d].SubmitOnEnter = false, want true", i)
		}
	}

	if got := matches[1].InsertText; got != "/skills show" {
		t.Fatalf("show InsertText = %q, want /skills show", got)
	}
	if !matches[1].HasArgs {
		t.Fatal("show suggestion should keep HasArgs for Tab trailing space")
	}
	if !matches[1].ExpandOnEnter {
		t.Fatal("show suggestion should expand on Enter because it requires <name>")
	}
	if matches[1].SubmitOnEnter {
		t.Fatal("show suggestion should expand before submit because it requires <name>")
	}
	if got := matches[2].InsertText; got != "/skills suggest" {
		t.Fatalf("suggest InsertText = %q, want /skills suggest", got)
	}
	if !matches[2].HasArgs {
		t.Fatal("suggest suggestion should keep HasArgs for Tab trailing space")
	}
	if !matches[2].ExpandOnEnter {
		t.Fatal("suggest suggestion should expand on Enter because it requires <text>")
	}
	if matches[2].SubmitOnEnter {
		t.Fatal("suggest suggestion should expand before submit because it requires <text>")
	}
	if got := matches[0].InsertText; got != "/skills overview" {
		t.Fatalf("overview InsertText = %q, want /skills overview", got)
	}
	if matches[0].ExpandOnEnter {
		t.Fatal("overview suggestion should submit on Enter")
	}
	if matches[3].ExpandOnEnter {
		t.Fatal("usage suggestion should submit on Enter")
	}
	if matches[4].ExpandOnEnter {
		t.Fatal("doctor suggestion should submit on Enter")
	}
	if got := matches[1].CompletionText(true); got != "/skills show " {
		t.Fatalf("show CompletionText(true) = %q, want '/skills show '", got)
	}
	if got := matches[2].CompletionText(true); got != "/skills suggest " {
		t.Fatalf("suggest CompletionText(true) = %q, want '/skills suggest '", got)
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

	matches = Suggestions("/skills s")
	if len(matches) != 2 {
		t.Fatalf("Suggestions(/skills s) returned %d matches, want 2", len(matches))
	}
	wantSuggestLabels := []string{"show <name>", "suggest <text>"}
	for i, want := range wantSuggestLabels {
		if got := matches[i].Label; got != want {
			t.Fatalf("matches[%d].Label = %q, want %q", i, got, want)
		}
	}

	matches = Suggestions("/skills u")
	if len(matches) != 1 {
		t.Fatalf("Suggestions(/skills u) returned %d matches, want 1", len(matches))
	}
	if got := matches[0].InsertText; got != "/skills usage" {
		t.Fatalf("InsertText = %q, want /skills usage", got)
	}
	if got := matches[0].Description; got != "Show local Skill Router usage diagnostics" {
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
	want := []string{"/model", "/provider", "/thinking", "/status"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("leading suggestions = %#v, want prefix %#v", got, want)
		}
	}
}

func TestSuggestionsHidesNonDiscoverableCommands(t *testing.T) {
	for _, input := range []string{"/providers", "/use", "/version", "/help"} {
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
