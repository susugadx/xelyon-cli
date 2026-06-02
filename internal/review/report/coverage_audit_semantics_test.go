package report

import "testing"

func TestClassifyCoverageText(t *testing.T) {
	tests := []struct {
		name string
		text string
		want coverageTextSemantics
	}{
		{
			name: "positive official documentation claim",
			text: "official documentation confirms this behavior",
			want: coverageTextSemantics{
				claimsConfirmedExternalSpec: true,
			},
		},
		{
			name: "positive official confirmation subject assertion",
			text: "official confirmation is available from the vendor docs",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
				claimsConfirmedExternalSpec:     true,
			},
		},
		{
			name: "positive official confirmation prefix assertion",
			text: "found official confirmation in the vendor docs",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
				claimsConfirmedExternalSpec:     true,
			},
		},
		{
			name: "later unnegated confirmation claim",
			text: "not a confirmed external spec, but confirmed external spec coverage is claimed",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
				mentionsExternalSupportState:    true,
				claimsConfirmedExternalSpec:     true,
			},
		},
		{
			name: "later unnegated confirmation claim without comma",
			text: "not a confirmed external spec but confirmed external spec coverage is claimed",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
				mentionsExternalSupportState:    true,
				claimsConfirmedExternalSpec:     true,
			},
		},
		{
			name: "official confirmation absent",
			text: "official confirmation is absent",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
				mentionsExternalSupportState:    true,
				mentionsExternalGapState:        true,
			},
		},
		{
			name: "missing official confirmation",
			text: "missing official confirmation",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
				mentionsExternalSupportState:    true,
				mentionsExternalGapState:        true,
			},
		},
		{
			name: "verify official confirmation before marking verified",
			text: "verify official confirmation before marking verified",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
			},
		},
		{
			name: "obtain confirmed external spec coverage before marking verified",
			text: "obtain confirmed external spec coverage before marking verified",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
			},
		},
		{
			name: "not confirmed external spec",
			text: "not a confirmed external spec",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
				mentionsExternalSupportState:    true,
			},
		},
		{
			name: "unconfirmed external spec",
			text: "unconfirmed external spec",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
				mentionsExternalSupportState:    true,
				mentionsExternalGapState:        true,
			},
		},
		{
			name: "cannot establish official confirmation",
			text: "cannot establish official confirmation",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
				mentionsExternalGapState:        true,
			},
		},
		{
			name: "needs official confirmation",
			text: "needs official confirmation",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
			},
		},
		{
			name: "pending official confirmation",
			text: "pending official confirmation",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
			},
		},
		{
			name: "requires official confirmation",
			text: "requires official confirmation",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
			},
		},
		{
			name: "official confirmation still needed",
			text: "official confirmation is still needed",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
			},
		},
		{
			name: "official confirmation pending",
			text: "official confirmation remains pending",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
			},
		},
		{
			name: "official confirmation cannot be confirmed",
			text: "official confirmation cannot be confirmed",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
				mentionsExternalGapState:        true,
			},
		},
		{
			name: "official confirmation cannot be established",
			text: "official confirmation cannot be established",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
				mentionsExternalGapState:        true,
			},
		},
		{
			name: "official confirmation could not be established",
			text: "official confirmation could not be established",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
				mentionsExternalGapState:        true,
			},
		},
		{
			name: "could not establish official confirmation",
			text: "could not establish official confirmation",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
				mentionsExternalGapState:        true,
			},
		},
		{
			name: "unable to establish official confirmation",
			text: "unable to establish official confirmation",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
				mentionsExternalGapState:        true,
			},
		},
		{
			name: "confirmed external spec unavailable",
			text: "confirmed external spec is not available",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
				mentionsExternalGapState:        true,
			},
		},
		{
			name: "unofficial confirmation",
			text: "unofficial confirmation only",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
			},
		},
		{
			name: "cannot confirm confirmed external spec",
			text: "cannot confirm confirmed external spec coverage",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
				mentionsExternalGapState:        true,
			},
		},
		{
			name: "failed gap state",
			text: "failed",
			want: coverageTextSemantics{
				mentionsExternalGapState: true,
			},
		},
		{
			name: "no result gap state",
			text: "no result",
			want: coverageTextSemantics{
				mentionsExternalGapState: true,
			},
		},
		{
			name: "no external docs gap state",
			text: "no external docs",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
				mentionsExternalGapState:        true,
			},
		},
		{
			name: "inconclusive gap state",
			text: "inconclusive",
			want: coverageTextSemantics{
				mentionsExternalGapState: true,
			},
		},
		{
			name: "truncated gap state",
			text: "truncated",
			want: coverageTextSemantics{
				mentionsExternalGapState: true,
			},
		},
		{
			name: "truncation gap state",
			text: "truncation",
			want: coverageTextSemantics{
				mentionsExternalGapState: true,
			},
		},
		{
			name: "weak external support state",
			text: "weak external support",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
				mentionsExternalSupportState:    true,
				mentionsExternalGapState:        true,
			},
		},
		{
			name: "post pass external search context",
			text: "post-pass1 external search",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
				mentionsPostPass1Evidence:       true,
			},
		},
		{
			name: "external evidence context",
			text: "external evidence",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
			},
		},
		{
			name: "external doc context",
			text: "external doc",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext: true,
			},
		},
		{
			name: "explicit added external doc",
			text: "new external_doc",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext:  true,
				mentionsExplicitAddedExternalDoc: true,
			},
		},
		{
			name: "added external doc is also post pass evidence",
			text: "added external doc",
			want: coverageTextSemantics{
				mentionsExternalEvidenceContext:  true,
				mentionsPostPass1Evidence:        true,
				mentionsExplicitAddedExternalDoc: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyCoverageText(tt.text)
			if got != tt.want {
				t.Fatalf("classifyCoverageText(%q) = %#v, want %#v", tt.text, got, tt.want)
			}
		})
	}
}

func TestCoverageTextCompatibilityHelpersUseSemantics(t *testing.T) {
	if !textReferencesExplicitAddedExternalDoc("post-pass1 external search") {
		t.Fatal("textReferencesExplicitAddedExternalDoc() = false, want true for legacy post-pass external wording")
	}
	if textClaimsConfirmedExternalSpec("pending official confirmation") {
		t.Fatal("textClaimsConfirmedExternalSpec() = true, want false for pending confirmation wording")
	}
	if textClaimsConfirmedExternalSpec("missing official confirmation") {
		t.Fatal("textClaimsConfirmedExternalSpec() = true, want false for missing confirmation wording")
	}
	if textClaimsConfirmedExternalSpec("verify official confirmation before marking verified") {
		t.Fatal("textClaimsConfirmedExternalSpec() = true, want false for verification action wording")
	}
	if textClaimsConfirmedExternalSpec("obtain confirmed external spec coverage before marking verified") {
		t.Fatal("textClaimsConfirmedExternalSpec() = true, want false for obtain action wording")
	}
	if !textClaimsConfirmedExternalSpec("official confirmation is available from the vendor docs") {
		t.Fatal("textClaimsConfirmedExternalSpec() = false, want true for asserted official confirmation")
	}
	if !textClaimsConfirmedExternalSpec("not a confirmed external spec, but confirmed external spec coverage is claimed") {
		t.Fatal("textClaimsConfirmedExternalSpec() = false, want true for later unnegated claim")
	}
	if !textClaimsConfirmedExternalSpec("not a confirmed external spec but confirmed external spec coverage is claimed") {
		t.Fatal("textClaimsConfirmedExternalSpec() = false, want true for later unnegated claim without comma")
	}
}
