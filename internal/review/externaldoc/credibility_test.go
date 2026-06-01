package externaldoc

import (
	"strings"
	"testing"
)

func TestClassifySourceCredibilityOfficialCandidate(t *testing.T) {
	got, reason := classifySourceCredibility(
		FetchRequest{
			SearchResultTitle: "OpenAI API Reference - Responses",
			QuerySubjectHint:  "OpenAI Responses API",
		},
		Evidence{SourceDomain: "platform.openai.com"},
		"OpenAI API reference documentation for request parameters, responses, and authentication.",
	)

	if got != SourceCredibilityOfficialCandidate {
		t.Fatalf("credibility = %q, want official_candidate; reason=%q", got, reason)
	}
	if !strings.Contains(reason, "trusted source domain") {
		t.Fatalf("reason = %q, want trusted domain signal explanation", reason)
	}
}

func TestClassifySourceCredibilityRejectsOpenAILookalikeDomains(t *testing.T) {
	for _, sourceDomain := range []string{
		"docs.openai.evil.example",
		"openai.com.evil.example",
		"evilopenai.com",
	} {
		t.Run(sourceDomain, func(t *testing.T) {
			got, reason := classifySourceCredibility(
				FetchRequest{
					SearchResultTitle: "OpenAI API Reference - Responses",
					QuerySubjectHint:  "OpenAI Responses API",
				},
				Evidence{SourceDomain: sourceDomain},
				"OpenAI API reference documentation for request parameters, responses, and authentication.",
			)

			if got != SourceCredibilityUnknown {
				t.Fatalf("credibility = %q, want unknown; reason=%q", got, reason)
			}
			if !strings.Contains(reason, "does not match trusted domains") {
				t.Fatalf("reason = %q, want trusted domain mismatch explanation", reason)
			}
		})
	}
}

func TestClassifySourceCredibilityKeepsOfficialDocWithThirdPartyBodyWording(t *testing.T) {
	got, reason := classifySourceCredibility(
		FetchRequest{
			SearchResultTitle: "OpenAI API third-party integration reference",
			QuerySubjectHint:  "OpenAI API",
		},
		Evidence{SourceDomain: "platform.openai.com"},
		"OpenAI API reference documentation for third-party integration request parameters, responses, and authentication.",
	)

	if got != SourceCredibilityOfficialCandidate {
		t.Fatalf("credibility = %q, want official_candidate; reason=%q", got, reason)
	}
}

func TestClassifySourceCredibilityThirdPartyHost(t *testing.T) {
	got, reason := classifySourceCredibility(
		FetchRequest{
			SearchResultTitle: "OpenAI API reference",
			QuerySubjectHint:  "OpenAI API",
		},
		Evidence{SourceDomain: "medium.com"},
		"OpenAI API request examples.",
	)

	if got != SourceCredibilityThirdParty {
		t.Fatalf("credibility = %q, want third_party; reason=%q", got, reason)
	}
	if !strings.Contains(reason, "third-party host") {
		t.Fatalf("reason = %q, want third-party signal explanation", reason)
	}
}

func TestClassifySourceCredibilityThirdPartyTitleMetadata(t *testing.T) {
	got, reason := classifySourceCredibility(
		FetchRequest{
			SearchResultTitle: "OpenAI API unofficial community tutorial",
			QuerySubjectHint:  "OpenAI API",
		},
		Evidence{SourceDomain: "docs.example.test"},
		"OpenAI API request examples.",
	)

	if got != SourceCredibilityThirdParty {
		t.Fatalf("credibility = %q, want third_party; reason=%q", got, reason)
	}
}

func TestClassifySourceCredibilityGitHubCollaborationSourcesAreThirdParty(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		title string
	}{
		{
			name:  "issue path",
			url:   "https://github.com/openai/openai-python/issues/123",
			title: "OpenAI API official issue thread",
		},
		{
			name:  "discussion path",
			url:   "https://github.com/openai/openai-python/discussions/456",
			title: "OpenAI API discussion",
		},
		{
			name:  "pull request title",
			url:   "https://github.com/openai/openai-python/pull/789",
			title: "OpenAI API GitHub pull request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := classifySourceCredibility(
				FetchRequest{
					URL:               tt.url,
					SearchResultTitle: tt.title,
					QuerySubjectHint:  "OpenAI API",
				},
				Evidence{SourceDomain: "github.com"},
				"OpenAI API request examples and responses.",
			)

			if got != SourceCredibilityThirdParty {
				t.Fatalf("credibility = %q, want third_party; reason=%q", got, reason)
			}
		})
	}
}

func TestClassifySourceCredibilityUnknown(t *testing.T) {
	got, reason := classifySourceCredibility(
		FetchRequest{
			SearchResultTitle: "OpenAI API reference",
			QuerySubjectHint:  "OpenAI API",
		},
		Evidence{SourceDomain: "example.test"},
		"OpenAI API request and response examples.",
	)

	if got != SourceCredibilityUnknown {
		t.Fatalf("credibility = %q, want unknown; reason=%q", got, reason)
	}
	if !strings.Contains(reason, "does not match trusted domains") {
		t.Fatalf("reason = %q, want trusted domain mismatch explanation", reason)
	}
}

func TestClassifySourceCredibilityUnknownWhenSourceMetadataIsMissingOrMalformed(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		sourceDomain string
	}{
		{name: "missing URL and source domain"},
		{name: "malformed URL", url: "://not-a-url"},
		{name: "malformed source URL", sourceDomain: "https://[bad-host"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := classifySourceCredibility(
				FetchRequest{
					URL:               tt.url,
					SearchResultTitle: "Official OpenAI API Reference",
					QuerySubjectHint:  "OpenAI API",
				},
				Evidence{SourceDomain: tt.sourceDomain},
				"OpenAI API reference documentation for request parameters, responses, and authentication.",
			)

			if got != SourceCredibilityUnknown {
				t.Fatalf("credibility = %q, want unknown; reason=%q", got, reason)
			}
		})
	}
}

func TestClassifySourceCredibilityTitleSayingOfficialDoesNotOverrideUntrustedDomain(t *testing.T) {
	got, reason := classifySourceCredibility(
		FetchRequest{
			URL:               "https://openai-docs.example.test/reference",
			SearchResultTitle: "Official OpenAI API Reference",
			QuerySubjectHint:  "OpenAI API",
		},
		Evidence{SourceDomain: "openai-docs.example.test"},
		"OpenAI API reference documentation for request parameters, responses, and authentication.",
	)

	if got == SourceCredibilityOfficialCandidate {
		t.Fatalf("credibility = %q, want not official_candidate; reason=%q", got, reason)
	}
	if got != SourceCredibilityUnknown {
		t.Fatalf("credibility = %q, want unknown for untrusted domain; reason=%q", got, reason)
	}
}

func TestClassifySourceCredibilityUnknownSubject(t *testing.T) {
	got, reason := classifySourceCredibility(
		FetchRequest{
			SearchResultTitle: "OAuth reference",
			QuerySubjectHint:  "OAuth",
		},
		Evidence{SourceDomain: "oauth.example.test"},
		"OAuth request and response examples.",
	)

	if got != SourceCredibilityUnknown {
		t.Fatalf("credibility = %q, want unknown; reason=%q", got, reason)
	}
	if !strings.Contains(reason, "no trusted domain mapping") {
		t.Fatalf("reason = %q, want unmapped subject explanation", reason)
	}
}

func TestNormalizeSourceCredibilityDefaultsUnknown(t *testing.T) {
	if got := normalizeSourceCredibility(""); got != SourceCredibilityUnknown {
		t.Fatalf("normalized credibility = %q, want unknown", got)
	}
	if got := normalizeSourceCredibility("surprising"); got != SourceCredibilityUnknown {
		t.Fatalf("normalized credibility = %q, want unknown", got)
	}
	if got := normalizeSourceCredibilityReason("", ""); got == "" || !strings.Contains(got, "unknown") {
		t.Fatalf("normalized reason = %q, want unknown reason", got)
	}
}
