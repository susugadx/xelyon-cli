package review

import (
	"strings"
	"testing"
)

func TestClassifyReviewExternalDocSourceCredibilityOfficialCandidate(t *testing.T) {
	got, reason := classifyReviewExternalDocSourceCredibility(
		ReviewExternalDocFetchRequest{
			SearchResultTitle: "OpenAI API Reference - Responses",
			QuerySubjectHint:  "OpenAI Responses API",
		},
		ReviewExternalDocEvidence{SourceDomain: "platform.openai.com"},
		"OpenAI API reference documentation for request parameters, responses, and authentication.",
	)

	if got != ReviewExternalDocSourceCredibilityOfficialCandidate {
		t.Fatalf("credibility = %q, want official_candidate; reason=%q", got, reason)
	}
	if !strings.Contains(reason, "trusted source domain") {
		t.Fatalf("reason = %q, want trusted domain signal explanation", reason)
	}
}

func TestClassifyReviewExternalDocSourceCredibilityRejectsOpenAILookalikeDomains(t *testing.T) {
	for _, sourceDomain := range []string{
		"docs.openai.evil.example",
		"openai.com.evil.example",
		"evilopenai.com",
	} {
		t.Run(sourceDomain, func(t *testing.T) {
			got, reason := classifyReviewExternalDocSourceCredibility(
				ReviewExternalDocFetchRequest{
					SearchResultTitle: "OpenAI API Reference - Responses",
					QuerySubjectHint:  "OpenAI Responses API",
				},
				ReviewExternalDocEvidence{SourceDomain: sourceDomain},
				"OpenAI API reference documentation for request parameters, responses, and authentication.",
			)

			if got != ReviewExternalDocSourceCredibilityUnknown {
				t.Fatalf("credibility = %q, want unknown; reason=%q", got, reason)
			}
			if !strings.Contains(reason, "does not match trusted domains") {
				t.Fatalf("reason = %q, want trusted domain mismatch explanation", reason)
			}
		})
	}
}

func TestClassifyReviewExternalDocSourceCredibilityKeepsOfficialDocWithThirdPartyBodyWording(t *testing.T) {
	got, reason := classifyReviewExternalDocSourceCredibility(
		ReviewExternalDocFetchRequest{
			SearchResultTitle: "OpenAI API third-party integration reference",
			QuerySubjectHint:  "OpenAI API",
		},
		ReviewExternalDocEvidence{SourceDomain: "platform.openai.com"},
		"OpenAI API reference documentation for third-party integration request parameters, responses, and authentication.",
	)

	if got != ReviewExternalDocSourceCredibilityOfficialCandidate {
		t.Fatalf("credibility = %q, want official_candidate; reason=%q", got, reason)
	}
}

func TestClassifyReviewExternalDocSourceCredibilityThirdPartyHost(t *testing.T) {
	got, reason := classifyReviewExternalDocSourceCredibility(
		ReviewExternalDocFetchRequest{
			SearchResultTitle: "OpenAI API reference",
			QuerySubjectHint:  "OpenAI API",
		},
		ReviewExternalDocEvidence{SourceDomain: "medium.com"},
		"OpenAI API request examples.",
	)

	if got != ReviewExternalDocSourceCredibilityThirdParty {
		t.Fatalf("credibility = %q, want third_party; reason=%q", got, reason)
	}
	if !strings.Contains(reason, "third-party host") {
		t.Fatalf("reason = %q, want third-party signal explanation", reason)
	}
}

func TestClassifyReviewExternalDocSourceCredibilityThirdPartyTitleMetadata(t *testing.T) {
	got, reason := classifyReviewExternalDocSourceCredibility(
		ReviewExternalDocFetchRequest{
			SearchResultTitle: "OpenAI API unofficial community tutorial",
			QuerySubjectHint:  "OpenAI API",
		},
		ReviewExternalDocEvidence{SourceDomain: "docs.example.test"},
		"OpenAI API request examples.",
	)

	if got != ReviewExternalDocSourceCredibilityThirdParty {
		t.Fatalf("credibility = %q, want third_party; reason=%q", got, reason)
	}
}

func TestClassifyReviewExternalDocSourceCredibilityUnknown(t *testing.T) {
	got, reason := classifyReviewExternalDocSourceCredibility(
		ReviewExternalDocFetchRequest{
			SearchResultTitle: "OpenAI API reference",
			QuerySubjectHint:  "OpenAI API",
		},
		ReviewExternalDocEvidence{SourceDomain: "example.test"},
		"OpenAI API request and response examples.",
	)

	if got != ReviewExternalDocSourceCredibilityUnknown {
		t.Fatalf("credibility = %q, want unknown; reason=%q", got, reason)
	}
	if !strings.Contains(reason, "does not match trusted domains") {
		t.Fatalf("reason = %q, want trusted domain mismatch explanation", reason)
	}
}

func TestClassifyReviewExternalDocSourceCredibilityUnknownSubject(t *testing.T) {
	got, reason := classifyReviewExternalDocSourceCredibility(
		ReviewExternalDocFetchRequest{
			SearchResultTitle: "OAuth reference",
			QuerySubjectHint:  "OAuth",
		},
		ReviewExternalDocEvidence{SourceDomain: "oauth.example.test"},
		"OAuth request and response examples.",
	)

	if got != ReviewExternalDocSourceCredibilityUnknown {
		t.Fatalf("credibility = %q, want unknown; reason=%q", got, reason)
	}
	if !strings.Contains(reason, "no trusted domain mapping") {
		t.Fatalf("reason = %q, want unmapped subject explanation", reason)
	}
}

func TestNormalizeReviewExternalDocSourceCredibilityDefaultsUnknown(t *testing.T) {
	if got := normalizeReviewExternalDocSourceCredibility(""); got != ReviewExternalDocSourceCredibilityUnknown {
		t.Fatalf("normalized credibility = %q, want unknown", got)
	}
	if got := normalizeReviewExternalDocSourceCredibility("surprising"); got != ReviewExternalDocSourceCredibilityUnknown {
		t.Fatalf("normalized credibility = %q, want unknown", got)
	}
	if got := normalizeReviewExternalDocSourceCredibilityReason("", ""); got == "" || !strings.Contains(got, "unknown") {
		t.Fatalf("normalized reason = %q, want unknown reason", got)
	}
}
