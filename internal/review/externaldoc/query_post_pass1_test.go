package externaldoc

import (
	"strings"
	"testing"
)

func TestBuildSearchQueryCandidatesUsesPass1PlanAPIDocsIntent(t *testing.T) {
	got := BuildSearchQueryCandidates(SearchQueryPlanningInput{
		ImpactSurfaces: []SearchQueryPlanImpactSurface{
			{
				Summary:         "OpenAI Responses API request state may regress.",
				Category:        "prompt_contract",
				EvidenceSummary: "Diff mentions previous_response_id and Responses API.",
			},
		},
		CandidateRisks: []SearchQueryPlanCandidateRisk{
			{
				Summary:              "previous_response_id could be sent on the wrong request path.",
				VerificationStrategy: "Compare against OpenAI Responses API docs.",
			},
		},
	})

	if len(got) == 0 {
		t.Fatal("candidates empty, want plan-derived OpenAI Responses API query")
	}
	first := got[0]
	if first.Query() != "OpenAI Responses API previous_response_id official documentation" {
		t.Fatalf("query = %q, want plan-derived Responses query", first.Query())
	}
	if first.Intent() != QueryIntentAPIDocs || first.ExpectedSourceType() != QueryExpectedSourceAPIReference {
		t.Fatalf("intent/source = %q/%q, want api_docs/api_reference", first.Intent(), first.ExpectedSourceType())
	}
	if !strings.Contains(first.EvidenceReason(), "intent=api_docs") || !strings.Contains(first.EvidenceReason(), "expected_source_type=api_reference") || !strings.Contains(first.EvidenceReason(), "confidence=high") {
		t.Fatalf("EvidenceReason() = %q, want query metadata", first.EvidenceReason())
	}
}

func TestBuildSearchQueryCandidatesUsesEachMatchedPass1PlanFocus(t *testing.T) {
	got := BuildSearchQueryCandidates(SearchQueryPlanningInput{
		ImpactSurfaces: []SearchQueryPlanImpactSurface{
			{
				Summary:         "OpenAI API request formatting and tool dispatch changed.",
				Category:        "prompt_contract",
				EvidenceSummary: "Diff mentions response_format and tool_choice.",
			},
		},
		CandidateRisks: []SearchQueryPlanCandidateRisk{
			{
				Summary:              "response_format may be incompatible with structured output requests.",
				VerificationStrategy: "Compare response_format against OpenAI API docs.",
			},
			{
				Summary:              "tool_choice may route tool calls incorrectly.",
				VerificationStrategy: "Compare tool_choice against OpenAI API docs.",
			},
		},
	})

	assertSearchQueryCandidatesContainQueries(t, got,
		"OpenAI API response_format official documentation",
		"OpenAI API tool_choice official documentation",
	)
}

func TestBuildSearchQueryCandidatesUsesPass1PlanSpecIntentForOAuth(t *testing.T) {
	got := BuildSearchQueryCandidates(SearchQueryPlanningInput{
		ImpactSurfaces: []SearchQueryPlanImpactSurface{
			{
				Summary: "OAuth redirect URI validation changed.",
				Reason:  "Token exchange and redirect_uri matching need external protocol confirmation.",
			},
		},
		CandidateRisks: []SearchQueryPlanCandidateRisk{
			{Summary: "Authorization code could be accepted with a mismatched redirect URI."},
		},
	})

	if len(got) == 0 {
		t.Fatal("candidates empty, want OAuth specification query")
	}
	if got[0].Query() != "OAuth 2.0 redirect URI specification" {
		t.Fatalf("query = %q, want OAuth spec query", got[0].Query())
	}
	if got[0].Intent() != QueryIntentSpec || got[0].ExpectedSourceType() != QueryExpectedSourceTechnicalSpecification {
		t.Fatalf("intent/source = %q/%q, want spec/technical_specification", got[0].Intent(), got[0].ExpectedSourceType())
	}
}

func assertSearchQueryCandidatesContainQueries(t *testing.T, candidates []SearchQueryCandidate, queries ...string) {
	t.Helper()
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		seen[candidate.Query()] = struct{}{}
	}
	for _, query := range queries {
		if _, ok := seen[query]; !ok {
			t.Fatalf("candidates = %#v, want query %q", candidates, query)
		}
	}
}

func TestBuildSearchQueryCandidatesUsesGoFilepathSecuritySignal(t *testing.T) {
	got := BuildSearchQueryCandidates(SearchQueryPlanningInput{
		CorpusParts: []string{
			"internal/pathcheck/pathcheck.go",
			"+ cleaned := filepath.Clean(candidate)",
		},
		ImpactSurfaces: []SearchQueryPlanImpactSurface{
			{
				Summary: "Go path/filepath symlink handling may allow path traversal.",
				Reason:  "filepath.Clean and symlink traversal behavior need confirmation.",
			},
		},
	})

	if len(got) == 0 {
		t.Fatal("candidates empty, want Go filepath query")
	}
	if !strings.Contains(got[0].Query(), "Go filepath package") {
		t.Fatalf("query = %q, want Go filepath subject", got[0].Query())
	}
	if got[0].Intent() != QueryIntentSecurityAdvisory {
		t.Fatalf("intent = %q, want security_advisory for traversal signal", got[0].Intent())
	}
}

func TestBuildSearchQueryCandidatesSuppressesGenericRiskText(t *testing.T) {
	got := BuildSearchQueryCandidates(SearchQueryPlanningInput{
		ImpactSurfaces: []SearchQueryPlanImpactSurface{
			{Summary: "API error handling best practice could be wrong."},
		},
		CandidateRisks: []SearchQueryPlanCandidateRisk{
			{Summary: "Review bug may exist.", VerificationStrategy: "Check best practice."},
		},
		GenericImpactTokens: []string{"api", "configuration", "request"},
	})

	if len(got) != 0 {
		t.Fatalf("candidates = %#v, want no generic best-practice queries", got)
	}
}

func TestBuildSearchQueryCandidatesDedupesDuplicatePlanQueries(t *testing.T) {
	got := BuildSearchQueryCandidates(SearchQueryPlanningInput{
		CorpusParts: []string{"internal/api/providers/openai/client.go"},
		ImpactSurfaces: []SearchQueryPlanImpactSurface{
			{Summary: "OpenAI Responses API previous_response_id changed."},
			{Reason: "OpenAI Responses API previous_response_id compatibility risk."},
		},
		CandidateRisks: []SearchQueryPlanCandidateRisk{
			{Summary: "OpenAI Responses API previous_response_id may be invalid."},
		},
		GenericImpactTokens: []string{"previous_response_id"},
	})

	seen := map[string]struct{}{}
	for _, candidate := range got {
		key := strings.ToLower(candidate.Query())
		if _, exists := seen[key]; exists {
			t.Fatalf("duplicate query %q in candidates %#v", candidate.Query(), got)
		}
		seen[key] = struct{}{}
	}
}
