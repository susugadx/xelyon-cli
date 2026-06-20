package analysis

import (
	"slices"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
)

func TestBuildPressureSignalsIncludesReviewPressureSignals(t *testing.T) {
	input := EvidenceInput{
		ChangeInventory: ChangeInventory{
			Generated: []string{"internal/generated/schema.pb.go"},
			Config:    []string{"config/review.yaml"},
			Production: []string{
				"internal/review/runner_prompt.go",
			},
			DeletedFiles: []string{"internal/review/old_command.go"},
			RenamedFiles: []string{"internal/review/new_command.go"},
			Untracked:    []string{"scratch.txt"},
		},
		RelatedContextFiles: []ContextFile{
			{
				Path:       "internal/review/related.go",
				Skipped:    true,
				SkipReason: "not readable",
				Truncated:  true,
			},
		},
		TruncationFlags: TruncationFlags{
			StatusShort: true,
		},
	}

	signals := BuildPressureSignals(input, PressureSignalOptions{
		KnownRuleFilePaths: []string{"AGENTS.md", "CLAUDE.md"},
	})

	for _, want := range []string{
		"production_changed_without_tests",
		"config_or_schema_changed",
		"prompt_contract_changed",
		"deleted_or_renamed_files",
		"untracked_files_present",
		"related_context_empty_or_shallow",
		"related_search_empty_or_truncated",
		"diff_or_context_truncated",
		"generated_files_changed",
	} {
		if !pressureSignalsContain(signals, want) {
			t.Fatalf("signals = %#v, want %s", signals, want)
		}
	}
	for _, want := range []string{
		"config: config/review.yaml",
		"schema_or_contract_path: internal/generated/schema.pb.go",
		"prompt_or_instruction_path: internal/review/runner_prompt.go",
		"related_context_file skipped: internal/review/related.go (not readable)",
		"related_search_hits: []",
		"status_short: truncated",
		"generated: internal/generated/schema.pb.go",
	} {
		if !pressureSignalsEvidenceContains(signals, want) {
			t.Fatalf("signals = %#v, want evidence %q", signals, want)
		}
	}
}

func TestBuildPressureSignalsEmptyArray(t *testing.T) {
	input := EvidenceInput{
		RelatedContextFiles: []ContextFile{{Path: "internal/review/context.go"}},
		RelatedSearchHits:   []RelatedSearchHit{{Path: "internal/review/context.go"}},
	}

	if got := BuildPressureSignals(input, PressureSignalOptions{}); len(got) != 0 {
		t.Fatalf("signals = %#v, want empty", got)
	}
}

func TestBuildPressureSignalsIncludesGenericImpactPressureSignals(t *testing.T) {
	input := EvidenceInput{
		ChangeInventory: ChangeInventory{
			Production: []string{"src/feature.ts"},
		},
		GenericImpact: GenericImpact{
			Tokens:    []string{"/review"},
			Truncated: true,
			Candidates: []GenericImpactCandidate{
				{
					Path:  "src/feature.test.ts",
					Role:  "same_stem_test_or_spec",
					Token: "feature",
				},
				{
					Path:  "docs/commands.md",
					Role:  "docs_reference",
					Token: "/review",
				},
			},
		},
	}

	signals := BuildPressureSignals(input, PressureSignalOptions{})

	for _, want := range []string{
		"generic_impact_candidates_present",
		"generic_impact_candidates_truncated",
		"generic_impact_candidates_include_tests_or_docs",
	} {
		if !pressureSignalsContain(signals, want) {
			t.Fatalf("signals = %#v, want %s", signals, want)
		}
	}
	for _, want := range []string{
		"generic_impact_candidate: same_stem_test_or_spec src/feature.test.ts token=feature",
		"generic_impact_candidate: docs_reference docs/commands.md token=/review",
	} {
		if !pressureSignalsEvidenceContains(signals, want) {
			t.Fatalf("signals = %#v, want evidence %q", signals, want)
		}
	}
}

func TestBuildPressureSignalsSignalsEmptyGenericImpactForNonGoChange(t *testing.T) {
	input := EvidenceInput{
		ChangeInventory: ChangeInventory{
			Production: []string{"src/feature.ts"},
		},
	}

	signals := BuildPressureSignals(input, PressureSignalOptions{})

	if !pressureSignalsContain(signals, "generic_impact_candidates_empty_for_non_go_change") {
		t.Fatalf("signals = %#v, want empty generic impact signal", signals)
	}
}

func TestBuildPressureSignalsIncludesKnownRuleInstructionFiles(t *testing.T) {
	for _, rulePath := range []string{"AGENTS.md", "CLAUDE.md", ".codex/config.toml"} {
		t.Run(rulePath, func(t *testing.T) {
			input := EvidenceInput{
				ChangeInventory: ChangeInventory{
					Docs: []string{rulePath},
				},
				RelatedContextFiles: []ContextFile{{Path: "internal/review/context.go"}},
				RelatedSearchHits:   []RelatedSearchHit{{Path: "internal/review/context.go"}},
			}

			signals := BuildPressureSignals(input, PressureSignalOptions{
				KnownRuleFilePaths: []string{"AGENTS.md", "CLAUDE.md", ".codex/config.toml"},
			})

			if !pressureSignalsContain(signals, "prompt_contract_changed") {
				t.Fatalf("signals = %#v, want prompt contract signal", signals)
			}
			if !pressureSignalsEvidenceContains(signals, "prompt_or_instruction_path: "+rulePath) {
				t.Fatalf("signals = %#v, want rule path evidence", signals)
			}
		})
	}
}

func TestBuildPressureSignalsIncludeWebSearchEvidenceStates(t *testing.T) {
	disabledSignals := BuildPressureSignals(EvidenceInput{
		ChangeInventory: ChangeInventory{
			Production: []string{"internal/api/providers/openai/web_search.go"},
		},
		Diffs: []Diff{
			{
				Diff: TextBlock{Content: `+ Tools: []map[string]any{{"type":"web_search"}}`},
			},
		},
		GenericImpact: GenericImpact{
			Tokens: []string{"web_search"},
		},
	}, PressureSignalOptions{})
	if !pressureSignalsContain(disabledSignals, "web_search_evidence_disabled_for_external_contract_change") {
		t.Fatalf("signals = %#v, want disabled external contract signal", disabledSignals)
	}

	enabledSignals := BuildPressureSignals(EvidenceInput{
		WebSearchEvidence: externaldoc.WebSearchEvidence{
			Enabled:      true,
			Error:        "fetch failed",
			Truncated:    true,
			Inconclusive: true,
			ExternalDocs: []externaldoc.Evidence{
				{
					DocID:        "external-doc-1",
					SourceDomain: "docs.example.test",
					Error:        "fetch failed",
					Truncated:    true,
				},
			},
		},
	}, PressureSignalOptions{})
	for _, signal := range []string{
		"web_search_evidence_failed",
		"web_search_evidence_truncated",
		"web_search_evidence_inconclusive",
	} {
		if !pressureSignalsContain(enabledSignals, signal) {
			t.Fatalf("signals = %#v, want %s", enabledSignals, signal)
		}
	}
}

func pressureSignalsContain(signals []PressureSignal, want string) bool {
	return slices.ContainsFunc(signals, func(signal PressureSignal) bool {
		return signal.Signal == want
	})
}

func pressureSignalsEvidenceContains(signals []PressureSignal, want string) bool {
	return slices.ContainsFunc(signals, func(signal PressureSignal) bool {
		return slices.ContainsFunc(signal.Evidence, func(evidence string) bool {
			return strings.Contains(evidence, want)
		})
	})
}
