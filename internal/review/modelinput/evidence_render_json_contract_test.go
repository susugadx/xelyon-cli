package modelinput

import (
	"encoding/json"
	"strings"
	"testing"

	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
)

func TestRenderReviewEvidenceJSONUsesModelInputDTO(t *testing.T) {
	bundle := newReviewEvidenceRenderTestBundle(t)

	data, err := RenderReviewEvidenceJSON(bundle)
	if err != nil {
		t.Fatalf("RenderReviewEvidenceJSON() error = %v", err)
	}
	payload := string(data)

	for _, forbidden := range []string{
		bundle.RepoRoot,
		bundle.CWD,
		`"RepoRoot"`,
		`"CWD"`,
		`"CommandTimeout"`,
		`"command_timeout":`,
		`"review_pressure_signals"`,
		`"production_changed_without_tests"`,
		`1500000000`,
	} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("json leaked forbidden fragment %q: %s", forbidden, payload)
		}
	}
	for _, want := range []string{
		`"repo_root": "<repo_root>"`,
		`"cwd_display": "src/work"`,
		`"external_support": {`,
		`"command_timeout_ms": 1500`,
	} {
		if !strings.Contains(payload, want) {
			t.Fatalf("json = %s, want fragment %q", payload, want)
		}
	}

	var input ReviewEvidenceModelInput
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if input.RepoRoot != reviewEvidenceRepoRootPathDisplay {
		t.Fatalf("RepoRoot = %q, want placeholder", input.RepoRoot)
	}
	if input.Limits.CommandTimeoutMS != 1500 {
		t.Fatalf("CommandTimeoutMS = %d, want 1500", input.Limits.CommandTimeoutMS)
	}
	if input.ExternalSupport.Level == "" {
		t.Fatal("ExternalSupport.Level is empty, want summarized support level")
	}
}

func TestBuildReviewEvidenceModelInputMinimalBundleUsesStableEmptySlices(t *testing.T) {
	input := BuildReviewEvidenceModelInput(reviewevidence.ReviewEvidenceBundle{})

	if input.ChangedFiles == nil {
		t.Fatal("ChangedFiles = nil, want empty slice")
	}
	if input.ChangedFileContext == nil {
		t.Fatal("ChangedFileContext = nil, want empty slice")
	}
	if input.RelatedContextFiles == nil {
		t.Fatal("RelatedContextFiles = nil, want empty slice")
	}
	if input.RelatedSearchHits == nil {
		t.Fatal("RelatedSearchHits = nil, want empty slice")
	}
	if input.GenericImpact.Tokens == nil {
		t.Fatal("GenericImpact.Tokens = nil, want empty slice")
	}
	if input.GenericImpact.Candidates == nil {
		t.Fatal("GenericImpact.Candidates = nil, want empty slice")
	}
	if input.ExternalSupport.Warnings == nil {
		t.Fatal("ExternalSupport.Warnings = nil, want stable empty-or-populated slice")
	}
	if input.ExternalSupport.Reasons == nil {
		t.Fatal("ExternalSupport.Reasons = nil, want stable empty-or-populated slice")
	}
	if input.RuleFiles == nil {
		t.Fatal("RuleFiles = nil, want empty slice")
	}
	if input.Diffs == nil {
		t.Fatal("Diffs = nil, want empty slice")
	}
	if input.UntrackedFiles == nil {
		t.Fatal("UntrackedFiles = nil, want empty slice")
	}
	if input.ChangeInventory.Generated == nil {
		t.Fatal("ChangeInventory.Generated = nil, want empty slice")
	}
	if input.TruncationFlags.Diffs == nil {
		t.Fatal("TruncationFlags.Diffs = nil, want empty slice")
	}
	if input.TruncationFlags.RuleFiles == nil {
		t.Fatal("TruncationFlags.RuleFiles = nil, want empty slice")
	}
	if input.TruncationFlags.ChangedFileContext == nil {
		t.Fatal("TruncationFlags.ChangedFileContext = nil, want empty slice")
	}
	if input.TruncationFlags.RelatedContextFiles == nil {
		t.Fatal("TruncationFlags.RelatedContextFiles = nil, want empty slice")
	}

	data, err := RenderReviewEvidenceJSON(reviewevidence.ReviewEvidenceBundle{})
	if err != nil {
		t.Fatalf("RenderReviewEvidenceJSON() error = %v", err)
	}
	payload := string(data)
	for _, want := range []string{
		`"changed_files": []`,
		`"changed_file_context": []`,
		`"related_context_files": []`,
		`"related_search_hits": []`,
		`"generic_impact_candidates": {`,
		`"tokens": []`,
		`"candidates": []`,
		`"external_support": {`,
		`"rule_files": []`,
		`"diffs": []`,
		`"untracked_files": []`,
	} {
		if !strings.Contains(payload, want) {
			t.Fatalf("json = %s, want empty slice fragment %q", payload, want)
		}
	}

	markdown := RenderReviewEvidenceMarkdown(reviewevidence.ReviewEvidenceBundle{})
	for _, want := range []string{
		"## git status --short\n```text\n",
		"## changed file context\n```json\n[]\n```",
		"## related tests/context files\n```json\n[]\n```",
		"## related search hits\n```json\n[]\n```",
		"## generic impact candidates\n```json\n",
		"## external support summary\n```json\n",
		"## diffs\n```json\n[]\n```",
		"## untracked files\n```json\n[]\n```",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown = %q, want empty section fragment %q", markdown, want)
		}
	}
}
