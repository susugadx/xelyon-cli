package tools

import "testing"

func TestXMLToolCallCandidateTagNames_MatchingOutsideCodeBlock(t *testing.T) {
	response := "before <mcp_github_create_issue><title>bug</title></mcp_github_create_issue> after"

	got := XMLToolCallCandidateTagNames(response)

	if len(got) != 1 || got[0] != "mcp_github_create_issue" {
		t.Fatalf("XMLToolCallCandidateTagNames() = %#v, want mcp_github_create_issue", got)
	}
}

func TestXMLToolCallCandidateTagNames_IgnoresCodeBlock(t *testing.T) {
	response := "example:\n```xml\n<mcp_github_create_issue><title>bug</title></mcp_github_create_issue>\n```\n"

	got := XMLToolCallCandidateTagNames(response)

	if len(got) != 0 {
		t.Fatalf("XMLToolCallCandidateTagNames() = %#v, want no candidates from code block", got)
	}
}

func TestXMLToolCallCandidateTagNames_IgnoresUnmatchedOpenTag(t *testing.T) {
	response := "before <mcp_github_create_issue>missing close after"

	got := XMLToolCallCandidateTagNames(response)

	if len(got) != 0 {
		t.Fatalf("XMLToolCallCandidateTagNames() = %#v, want no unmatched candidate", got)
	}
}
