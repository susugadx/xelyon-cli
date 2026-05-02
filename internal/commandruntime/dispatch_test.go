package commandruntime

import "testing"

func TestParse_ResolvesAliasesAndQuotes(t *testing.T) {
	got, ok := Parse(`/h "hello world"`, map[string]string{"/x": "/status"})
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

func TestSplitStrict_QuoteGroupAfterTokenPrefixStaysSingleToken(t *testing.T) {
	parts, status := SplitStrict(`/note foo'bar baz'`)
	if !status.IsOK() {
		t.Fatalf("SplitStrict() status = %v, want ok", status)
	}
	if len(parts) != 2 || parts[0] != "/note" || parts[1] != "foobar baz" {
		t.Fatalf("parts = %#v, want [/note foobar baz]", parts)
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
	if _, ok := Parse(`/attach "foo`, nil); ok {
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
