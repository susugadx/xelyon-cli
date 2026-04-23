package tools

import (
	"io"
	"testing"
)

func TestParseToolCalls_UnclosedCodeBlock_DefaultPolicyTreatsToEOF(t *testing.T) {
	input := "```json\nexample\n{\"tool\": \"read_file\", \"args\": {\"path\": \"main.go\"}}"

	defaultOptions := resolveParseRunOptions(newXMLTestRegistry(t), io.Discard)
	gotDefault := parseToolCalls(input, defaultOptions)
	if len(gotDefault) != 0 {
		t.Fatalf("default policy returned %d calls, want 0", len(gotDefault))
	}
}

func TestParseToolCalls_CodeBlockPolicyInjection(t *testing.T) {
	input := "```json\nexample\n{\"tool\": \"read_file\", \"args\": {\"path\": \"main.go\"}}"

	options := resolveParseRunOptions(newXMLTestRegistry(t), io.Discard)
	options.plan.codeBlockPolicy = markdownCodeBlockPolicy{
		unclosedFence: markdownUnclosedFencePolicyIgnore,
	}

	gotCustom := parseToolCalls(input, options)
	if len(gotCustom) != 1 {
		t.Fatalf("custom policy returned %d calls, want 1", len(gotCustom))
	}
	if gotCustom[0].Tool != "read_file" {
		t.Fatalf("Tool = %q, want 'read_file'", gotCustom[0].Tool)
	}
}
