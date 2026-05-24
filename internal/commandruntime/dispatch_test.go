package commandruntime

import "testing"

func TestParse_NormalizesCommandAndQuotes(t *testing.T) {
	got, ok := Parse(`/HELP "hello world"`)
	if !ok {
		t.Fatal("Parse() ok = false, want true")
	}
	if got.Command != "/help" {
		t.Fatalf("Command = %q, want /help", got.Command)
	}
	if len(got.Args) != 1 || got.Args[0] != "hello world" {
		t.Fatalf("Args = %#v, want quoted argument", got.Args)
	}
}

func TestQuoteArgRoundTripsSplitStrict(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "plain", value: "demo", want: "demo"},
		{name: "space", value: "my skill", want: `"my skill"`},
		{name: "double quote", value: `my "quoted" skill`, want: `"my \"quoted\" skill"`},
		{name: "single quote", value: "don't stop", want: `"don't stop"`},
		{name: "backslash before quote", value: `a\"b`, want: `"a\\"b"`},
		{name: "trailing backslash", value: `my skill\`, want: `"my skill"\`},
		{name: "quote and trailing backslash", value: `my "quoted" skill\`, want: `"my \"quoted\" skill"\`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quoted := QuoteArg(tt.value)
			if quoted != tt.want {
				t.Fatalf("QuoteArg(%q) = %q, want %q", tt.value, quoted, tt.want)
			}
			parts, status := SplitStrict("/cmd " + quoted)
			if !status.IsOK() {
				t.Fatalf("SplitStrict() status = %v, want ok", status)
			}
			if len(parts) != 2 || parts[1] != tt.value {
				t.Fatalf("SplitStrict round trip parts = %#v, want [/cmd %q]", parts, tt.value)
			}
		})
	}
}

func TestSplitStrict_SupportsWhitespaceDelimiters(t *testing.T) {
	parts, status := SplitStrict("/attach\t\"a b\"\nextra")
	if !status.IsOK() {
		t.Fatalf("SplitStrict() status = %v, want ok", status)
	}
	if len(parts) != 3 {
		t.Fatalf("len(parts) = %d, want 3", len(parts))
	}
	if parts[0] != "/attach" || parts[1] != "a b" || parts[2] != "extra" {
		t.Fatalf("parts = %#v", parts)
	}
}

func TestSplitStrict_UnterminatedQuote(t *testing.T) {
	parts, status := SplitStrict(`/attach "foo`)
	if status != SplitStatusUnterminatedQuote {
		t.Fatalf("SplitStrict() status = %v, want %v", status, SplitStatusUnterminatedQuote)
	}
	if len(parts) != 2 || parts[0] != "/attach" || parts[1] != "foo" {
		t.Fatalf("parts = %#v, want [/attach foo]", parts)
	}
}

func TestSplitStrict_DoesNotStartQuoteInsideToken(t *testing.T) {
	parts, status := SplitStrict(`/note don't stop`)
	if !status.IsOK() {
		t.Fatalf("SplitStrict() status = %v, want ok", status)
	}
	if len(parts) != 3 || parts[0] != "/note" || parts[1] != "don't" || parts[2] != "stop" {
		t.Fatalf("parts = %#v, want [/note don't stop]", parts)
	}
}

func TestSplitStrict_QuoteInsideTokenIsLiteral(t *testing.T) {
	parts, status := SplitStrict(`/note abc"def`)
	if !status.IsOK() {
		t.Fatalf("SplitStrict() status = %v, want ok", status)
	}
	if len(parts) != 2 || parts[0] != "/note" || parts[1] != `abc"def` {
		t.Fatalf("parts = %#v, want [/note abc\"def]", parts)
	}
}

func TestSplitStrict_QuoteGroupAfterTokenPrefix_Table(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "basic quoted group",
			input: `/note foo'bar baz'`,
			want:  []string{"/note", "foobar baz"},
		},
		{
			name:  "quoted group with suffix",
			input: `/note foo'bar baz'qux`,
			want:  []string{"/note", "foobar bazqux"},
		},
		{
			name:  "quoted group with trailing token",
			input: `/note foo'bar baz' qux`,
			want:  []string{"/note", "foobar baz", "qux"},
		},
		{
			name:  "short first word in quote",
			input: `/note foo'a b'qux`,
			want:  []string{"/note", "fooa bqux"},
		},
		{
			name:  "short first word i",
			input: `/note foo'i am'bar`,
			want:  []string{"/note", "fooi ambar"},
		},
		{
			name:  "short first word to",
			input: `/note foo'to be'bar`,
			want:  []string{"/note", "footo bebar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts, status := SplitStrict(tt.input)
			if !status.IsOK() {
				t.Fatalf("SplitStrict(%q) status = %v, want ok", tt.input, status)
			}
			if len(parts) != len(tt.want) {
				t.Fatalf("SplitStrict(%q) len(parts) = %d, want %d (%#v)", tt.input, len(parts), len(tt.want), parts)
			}
			for i := range tt.want {
				if parts[i] != tt.want[i] {
					t.Fatalf("SplitStrict(%q)[%d] = %q, want %q (%#v)", tt.input, i, parts[i], tt.want[i], parts)
				}
			}
		})
	}
}

func TestSplitStrict_ApostrophesInMultipleWordsStayLiteral(t *testing.T) {
	parts, status := SplitStrict(`/note don't it's`)
	if !status.IsOK() {
		t.Fatalf("SplitStrict() status = %v, want ok", status)
	}
	if len(parts) != 3 || parts[0] != "/note" || parts[1] != "don't" || parts[2] != "it's" {
		t.Fatalf("parts = %#v, want [/note don't it's]", parts)
	}
}

func TestSplitStrict_ContractionFollowedByPossessiveStaysLiteral(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "cant with possessive",
			input: "/note can't's",
			want:  "can't's",
		},
		{
			name:  "were with possessive",
			input: "/note we're's",
			want:  "we're's",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts, status := SplitStrict(tt.input)
			if !status.IsOK() {
				t.Fatalf("SplitStrict(%q) status = %v, want ok", tt.input, status)
			}
			if len(parts) != 2 || parts[0] != "/note" || parts[1] != tt.want {
				t.Fatalf("parts = %#v, want [/note %s]", parts, tt.want)
			}
		})
	}
}

func TestSplitStrict_QuotedWindowsPathKeepsBackslashes(t *testing.T) {
	input := `/attach "C:\Users\me\file with space.txt"`
	parts, status := SplitStrict(input)
	if !status.IsOK() {
		t.Fatalf("SplitStrict() status = %v, want ok", status)
	}
	if len(parts) != 2 {
		t.Fatalf("len(parts) = %d, want 2", len(parts))
	}
	if parts[1] != `C:\Users\me\file with space.txt` {
		t.Fatalf("parts[1] = %q, want %q", parts[1], `C:\Users\me\file with space.txt`)
	}
}

func TestSplitStrict_QuotedWindowsUNCPathKeepsLeadingDoubleBackslash(t *testing.T) {
	input := `/attach "\\server\share\file name.txt"`
	parts, status := SplitStrict(input)
	if !status.IsOK() {
		t.Fatalf("SplitStrict() status = %v, want ok", status)
	}
	if len(parts) != 2 {
		t.Fatalf("len(parts) = %d, want 2", len(parts))
	}
	if parts[1] != `\\server\share\file name.txt` {
		t.Fatalf("parts[1] = %q, want %q", parts[1], `\\server\share\file name.txt`)
	}
}

func TestParse_UnterminatedQuoteReturnsFalse(t *testing.T) {
	if _, ok := Parse(`/attach "foo`); ok {
		t.Fatal("Parse() ok = true, want false")
	}
}

func TestSplitStatus_ErrorSummary(t *testing.T) {
	if got := SplitStatusUnterminatedQuote.ErrorSummary(); got != "unmatched quote" {
		t.Fatalf("SplitStatusUnterminatedQuote.ErrorSummary() = %q, want %q", got, "unmatched quote")
	}
	if got := SplitStatus(999).ErrorSummary(); got != "invalid token" {
		t.Fatalf("SplitStatus(999).ErrorSummary() = %q, want %q", got, "invalid token")
	}
}

func TestIsNonInteractiveConfigSubcommand(t *testing.T) {
	if !IsNonInteractiveConfigSubcommand([]string{"show"}) {
		t.Fatal("show should be non-interactive")
	}
	if !IsNonInteractiveConfigSubcommand([]string{"model", "gpt-test"}) {
		t.Fatal("model <name> should be non-interactive")
	}
	if IsNonInteractiveConfigSubcommand(nil) {
		t.Fatal("bare /config should be interactive")
	}
}
