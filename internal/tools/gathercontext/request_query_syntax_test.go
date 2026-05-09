package gathercontext

import (
	"reflect"
	"strings"
	"testing"
)

func TestTopLevelQueryTokensKeepQuotedTextTogether(t *testing.T) {
	tokens := topLevelQueryTokens(`search for "foo or bar" in docs/`)
	got := make([]string, 0, len(tokens))
	for _, token := range tokens {
		got = append(got, token.text)
	}

	want := []string{"search", "for", `"foo or bar"`, "in", "docs/"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tokens = %#v, want %#v", got, want)
	}
}

func TestSplitQueryOnTopLevelWordIgnoresQuotedWords(t *testing.T) {
	parts, ok := splitQueryOnTopLevelWord(`foo or "bar or baz" or quux`, "or")
	if !ok {
		t.Fatal("expected top-level or split")
	}
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}

	want := []string{"foo", `"bar or baz"`, "quux"}
	if !reflect.DeepEqual(parts, want) {
		t.Fatalf("parts = %#v, want %#v", parts, want)
	}
}
