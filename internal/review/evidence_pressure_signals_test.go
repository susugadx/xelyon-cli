package review

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderReviewEvidenceMarkdownIncludesReviewPressureSignals(t *testing.T) {
	bundle := newReviewEvidenceRenderTestBundle(t)
	repo := bundle.RepoRoot
	bundle.Inventory = ReviewChangeInventory{
		Generated:  []string{filepath.Join(repo, "internal", "generated", "schema.pb.go")},
		Config:     []string{filepath.Join(repo, "config", "review.yaml")},
		Production: []string{filepath.Join(repo, "internal", "review", "runner_prompt.go")},
		DeletedFiles: []string{
			filepath.Join(repo, "internal", "review", "old_command.go"),
		},
		RenamedFiles: []string{
			filepath.Join(repo, "internal", "review", "new_command.go"),
		},
		Untracked: []string{filepath.Join(repo, "scratch.txt")},
	}
	bundle.RelatedContextFiles = []ReviewContextFileEvidence{
		{
			Path:       filepath.Join(repo, "internal", "review", "related.go"),
			Role:       reviewContextFileRoleRelatedGo,
			Skipped:    true,
			SkipReason: "not readable",
			Truncated:  true,
			SizeBytes:  256,
			ReadBytes:  128,
		},
	}
	bundle.RelatedSearchHits = nil

	markdown := RenderReviewEvidenceMarkdown(bundle)

	assertReviewEvidenceSubstringsInOrder(t, markdown, []string{
		"## change inventory\n",
		"## review pressure signals\n",
		`"signal": "production_changed_without_tests"`,
		`"signal": "config_or_schema_changed"`,
		`"signal": "prompt_contract_changed"`,
		`"signal": "deleted_or_renamed_files"`,
		`"signal": "untracked_files_present"`,
		`"signal": "related_context_empty_or_shallow"`,
		`"signal": "related_search_empty_or_truncated"`,
		`"signal": "diff_or_context_truncated"`,
		`"signal": "generated_files_changed"`,
		"## generic impact candidates\n",
	})
	for _, want := range []string{
		`"evidence": [`,
		`"config: config/review.yaml"`,
		`"schema_or_contract_path: internal/generated/schema.pb.go"`,
		`"prompt_or_instruction_path: internal/review/runner_prompt.go"`,
		`"related_context_file skipped: internal/review/related.go (not readable)"`,
		`"related_search_hits: []"`,
		`"status_short: truncated"`,
		`"generated: internal/generated/schema.pb.go"`,
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown = %q, want pressure signal fragment %q", markdown, want)
		}
	}
}

func TestRenderReviewEvidenceMarkdownReviewPressureSignalsEmptyArray(t *testing.T) {
	repo := filepath.Clean(t.TempDir())
	bundle := ReviewEvidenceBundle{
		TargetKind: TargetCurrentChanges,
		RepoRoot:   repo,
		RelatedContextFiles: []ReviewContextFileEvidence{
			{
				Path:      "internal/review/context.go",
				Role:      reviewContextFileRoleRelatedGo,
				Content:   "package review\n",
				SizeBytes: 15,
				ReadBytes: 15,
			},
		},
		RelatedSearchHits: []ReviewRelatedSearchHit{
			{
				Path:    "internal/review/context.go",
				Line:    1,
				Snippet: "package review",
				Reason:  "symbol:context",
			},
		},
	}

	markdown := RenderReviewEvidenceMarkdown(bundle)

	if !strings.Contains(markdown, "## review pressure signals\n```json\n[]\n```\n\n## generic impact candidates\n") {
		t.Fatalf("markdown = %q, want empty review pressure signals array", markdown)
	}
}

func TestRenderReviewEvidenceMarkdownIncludesGenericImpactPressureSignals(t *testing.T) {
	repo := filepath.Clean(t.TempDir())
	bundle := ReviewEvidenceBundle{
		TargetKind: TargetCurrentChanges,
		RepoRoot:   repo,
		Inventory: ReviewChangeInventory{
			Production: []string{"src/feature.ts"},
		},
		GenericImpactCandidates: ReviewGenericImpactCandidates{
			Tokens:    []string{"/review"},
			Truncated: true,
			Candidates: []ReviewGenericImpactCandidate{
				{
					Path:  "src/feature.test.ts",
					Role:  ReviewGenericImpactRoleSameStemTestOrSpec,
					Token: "feature",
				},
				{
					Path:  "docs/commands.md",
					Role:  ReviewGenericImpactRoleDocsReference,
					Token: "/review",
				},
			},
		},
	}

	markdown := RenderReviewEvidenceMarkdown(bundle)

	for _, want := range []string{
		`"signal": "generic_impact_candidates_present"`,
		`"signal": "generic_impact_candidates_truncated"`,
		`"signal": "generic_impact_candidates_include_tests_or_docs"`,
		`"generic_impact_candidate: same_stem_test_or_spec src/feature.test.ts token=feature"`,
		`"generic_impact_candidate: docs_reference docs/commands.md token=/review"`,
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown = %q, want generic impact pressure signal fragment %q", markdown, want)
		}
	}
}

func TestRenderReviewEvidenceMarkdownSignalsEmptyGenericImpactForNonGoChange(t *testing.T) {
	bundle := ReviewEvidenceBundle{
		TargetKind: TargetCurrentChanges,
		RepoRoot:   filepath.Clean(t.TempDir()),
		Inventory: ReviewChangeInventory{
			Production: []string{"src/feature.ts"},
		},
	}

	markdown := RenderReviewEvidenceMarkdown(bundle)

	if !strings.Contains(markdown, `"signal": "generic_impact_candidates_empty_for_non_go_change"`) {
		t.Fatalf("markdown = %q, want empty generic impact pressure signal", markdown)
	}
}

func TestRenderReviewEvidenceMarkdownPressureSignalsIncludeKnownRuleInstructionFiles(t *testing.T) {
	for _, rulePath := range reviewEvidenceRuleFilePaths {
		t.Run(rulePath, func(t *testing.T) {
			bundle := newReviewEvidenceRenderTestBundle(t)
			bundle.Inventory = ReviewChangeInventory{
				Docs: []string{filepath.Join(bundle.RepoRoot, rulePath)},
			}
			bundle.RelatedContextFiles = []ReviewContextFileEvidence{
				{
					Path:      filepath.Join(bundle.RepoRoot, "internal", "review", "context.go"),
					Role:      reviewContextFileRoleRelatedGo,
					Content:   "package review\n",
					SizeBytes: 15,
					ReadBytes: 15,
				},
			}
			bundle.RelatedSearchHits = []ReviewRelatedSearchHit{
				{
					Path:    filepath.Join(bundle.RepoRoot, "internal", "review", "context.go"),
					Line:    1,
					Snippet: "package review",
					Reason:  "symbol:context",
				},
			}

			markdown := RenderReviewEvidenceMarkdown(bundle)

			for _, want := range []string{
				`"signal": "prompt_contract_changed"`,
				`"prompt_or_instruction_path: ` + rulePath + `"`,
			} {
				if !strings.Contains(markdown, want) {
					t.Fatalf("markdown = %q, want pressure signal fragment %q", markdown, want)
				}
			}
		})
	}
}
