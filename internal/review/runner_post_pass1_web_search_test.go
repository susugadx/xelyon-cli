package review

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	reviewartifact "github.com/susugadx/xelyon-cli/internal/review/artifact"
	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
)

func TestReviewRunnerRunUsesPostPass1WebSearchEvidenceForReport(t *testing.T) {
	events := []string{}
	initialBundle := newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")
	initialBundle.WebSearchEvidence = externaldoc.WebSearchEvidence{
		Enabled: true,
		Queries: []externaldoc.WebSearchEvidenceQuery{
			{Query: "OpenAI API web_search official documentation", Reason: "pre-pass1"},
		},
		Inconclusive: true,
	}
	mergedEvidence := initialBundle.WebSearchEvidence
	mergedEvidence.Queries = append(mergedEvidence.Queries, externaldoc.WebSearchEvidenceQuery{
		Query:  "OAuth 2.0 redirect URI specification",
		Reason: "intent=spec; expected_source_type=technical_specification; confidence=high; reason=pass1 plan protocol/spec signal",
	})
	mergedEvidence.ExternalDocs = []externaldoc.Evidence{
		{
			DocID:             "external-doc-post",
			URL:               "https://docs.example.test/oauth",
			SourceCredibility: externaldoc.SourceCredibilityUnknown,
			Snippets: []externaldoc.SnippetEvidence{
				{
					SnippetID:   "external-doc-post-snippet-1",
					Content:     "Post-pass1 OAuth redirect URI snippet.",
					ContentHash: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				},
			},
		},
	}
	mergedEvidence.Inconclusive = false
	evidence := &runnerPostPass1WebSearchEvidenceBuilder{
		runnerFakeEvidenceBuilder: runnerFakeEvidenceBuilder{
			bundle: initialBundle,
			events: &events,
		},
		postEvidence: mergedEvidence,
	}
	plan := newRunnerNoProbePlanForTest()
	report := newRunnerCleanReportForTest(nil)
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "surface-1 checked after reviewing weak external evidence external-doc-post; no confirmed external spec."
	report.ScopeCoverage.ReviewedCandidateRisks[0].Summary = "risk-1 dismissed after reviewing weak external evidence external-doc-post; no confirmed external spec."
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, report))},
			saturatedRunnerModelResponseForTest(t),
		},
		events: &events,
	}
	runner := newReviewRunnerForTest(t, evidence, &runnerFakeProbeRunner{events: &events}, model)

	if _, err := runner.Run(context.Background(), NewCurrentChangesRequest("")); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	assertStringSliceEqualForRunnerTest(t, events, []string{
		"evidence",
		"model:probe_plan",
		"post_search",
		"model:report",
		"model:saturation_check",
	})
	if evidence.postCalls != 1 {
		t.Fatalf("postCalls = %d, want 1", evidence.postCalls)
	}
	if evidence.seenPlan.SchemaVersion != reviewprobeplan.ReviewProbePlanSchemaVersionV2 {
		t.Fatalf("seen plan schema = %q, want %q", evidence.seenPlan.SchemaVersion, reviewprobeplan.ReviewProbePlanSchemaVersionV2)
	}
	if strings.Contains(model.requests[0].Prompt, "external-doc-post") {
		t.Fatalf("Pass1 prompt contains post-pass1 evidence:\n%s", model.requests[0].Prompt)
	}
	if !strings.Contains(model.requests[1].Prompt, "external-doc-post") || !strings.Contains(model.requests[1].Prompt, "OAuth 2.0 redirect URI specification") {
		t.Fatalf("Pass2 prompt missing merged post-pass1 evidence:\n%s", model.requests[1].Prompt)
	}
}

func TestReviewRunnerWebSearchDiscoveryCompactKeepsRawArtifactsAndExternalDocSnippets(t *testing.T) {
	tests := []struct {
		name               string
		mode               reviewpromptreduction.ReviewPromptReductionMode
		wantPromptCompact  bool
		wantReplacementCnt int
	}{
		{
			name:               "apply compacts provider prompt only",
			mode:               reviewpromptreduction.ReviewPromptReductionModeApply,
			wantPromptCompact:  true,
			wantReplacementCnt: 1,
		},
		{
			name:               "dry run records candidate without compacting prompt",
			mode:               reviewpromptreduction.ReviewPromptReductionModeDryRun,
			wantPromptCompact:  false,
			wantReplacementCnt: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			longSearchSnippet := strings.Repeat("RAW_DISCOVERY_SNIPPET_SHOULD_NOT_REACH_COMPACTED_PROVIDER_PROMPT ", 400)
			externalSnippet := "CITATION_CAPABLE_EXTERNAL_DOC_SNIPPET_MUST_STAY_FOR_FINDING_EVIDENCE"
			secondExternalSnippet := "SECOND_CITATION_CAPABLE_EXTERNAL_DOC_SNIPPET_MUST_STAY_FOR_SUPPORT"
			initialBundle := newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")
			initialBundle.WebSearchEvidence = externaldoc.WebSearchEvidence{
				Enabled:  true,
				Provider: "gemini",
				Queries: []externaldoc.WebSearchEvidenceQuery{
					{
						Query:  "OpenAI Responses API previous_response_id official docs",
						Reason: "pre-pass1 external contract signal",
						Results: []externaldoc.WebSearchEvidenceResult{
							{
								Title:        "OpenAI Responses API docs",
								URL:          "https://platform.openai.com/docs/responses",
								SourceDomain: "platform.openai.com",
								Snippet:      longSearchSnippet,
							},
						},
					},
				},
				Inconclusive: true,
			}
			mergedEvidence := initialBundle.WebSearchEvidence
			mergedEvidence.Inconclusive = false
			mergedEvidence.ExternalDocs = []externaldoc.Evidence{
				{
					DocID:                   "external-doc-openai",
					URL:                     "https://platform.openai.com/docs/responses",
					SourceDomain:            "platform.openai.com",
					SourceCredibility:       externaldoc.SourceCredibilityOfficialCandidate,
					SourceCredibilityReason: "official_candidate: trusted source domain and reference signal are present",
					FetchedAt:               time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
					ContentHash:             "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					Snippets: []externaldoc.SnippetEvidence{
						{
							SnippetID:   "external-doc-openai-snippet-1",
							Content:     externalSnippet,
							ContentHash: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
						},
					},
				},
				{
					DocID:                   "external-doc-openai-reference",
					URL:                     "https://platform.openai.com/docs/api-reference/responses",
					SourceDomain:            "platform.openai.com",
					SourceCredibility:       externaldoc.SourceCredibilityOfficialCandidate,
					SourceCredibilityReason: "official_candidate: trusted source domain and reference signal are present",
					FetchedAt:               time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
					ContentHash:             "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
					Snippets: []externaldoc.SnippetEvidence{
						{
							SnippetID:   "external-doc-openai-reference-snippet-1",
							Content:     secondExternalSnippet,
							ContentHash: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
						},
					},
				},
			}
			evidence := &runnerPostPass1WebSearchEvidenceBuilder{
				runnerFakeEvidenceBuilder: runnerFakeEvidenceBuilder{bundle: initialBundle},
				postEvidence:              mergedEvidence,
			}
			report := newRunnerCleanReportForTest(nil)
			report.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "surface-1 checked after reviewing official-confirmed external evidence external-doc-openai and external-doc-openai-reference."
			report.ScopeCoverage.ReviewedCandidateRisks[0].Summary = "risk-1 dismissed after reviewing official-confirmed external evidence external-doc-openai and external-doc-openai-reference."
			model := &runnerFakeModel{
				responses: []runnerFakeModelResponse{
					{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerNoProbePlanForTest()))},
					{content: string(mustMarshalReviewReportForRunnerTest(t, report))},
					saturatedRunnerModelResponseForTest(t),
				},
			}
			artifactDir := t.TempDir()
			artifactWriter, err := reviewartifact.NewReviewRunDirectoryArtifactWriter(artifactDir)
			if err != nil {
				t.Fatalf("reviewartifact.NewReviewRunDirectoryArtifactWriter() error = %v, want nil", err)
			}
			runner, err := NewReviewRunner(ReviewRunnerOptions{
				EvidenceBuilder:     evidence,
				ProbeRunner:         &runnerFakeProbeRunner{},
				Model:               model,
				ArtifactWriter:      artifactWriter,
				PromptReductionMode: tt.mode,
			})
			if err != nil {
				t.Fatalf("NewReviewRunner() error = %v, want nil", err)
			}

			if _, err := runner.Run(context.Background(), NewCurrentChangesRequest("")); err != nil {
				t.Fatalf("Run() error = %v, want nil", err)
			}

			if !strings.Contains(model.requests[0].Prompt, longSearchSnippet) {
				t.Fatalf("Pass1 prompt missing raw discovery snippet:\n%s", model.requests[0].Prompt)
			}
			if strings.Contains(model.requests[0].Prompt, externalSnippet) {
				t.Fatalf("Pass1 prompt contains post-pass1 external_doc snippet:\n%s", model.requests[0].Prompt)
			}
			reportPrompt := model.requests[1].Prompt
			if !strings.Contains(reportPrompt, externalSnippet) ||
				!strings.Contains(reportPrompt, secondExternalSnippet) ||
				!strings.Contains(reportPrompt, `"source_credibility": "official_candidate"`) ||
				!strings.Contains(reportPrompt, `"content_hash": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"`) {
				t.Fatalf("Pass2 prompt lost citation-capable external_doc evidence:\n%s", reportPrompt)
			}
			if tt.wantPromptCompact {
				if strings.Contains(reportPrompt, longSearchSnippet) {
					t.Fatalf("apply Pass2 prompt leaked raw discovery snippet:\n%s", reportPrompt)
				}
				if !strings.Contains(reportPrompt, "[compacted discovery-only web_search snippet") ||
					!strings.Contains(reportPrompt, "raw_result_preserved=review_artifact") {
					t.Fatalf("apply Pass2 prompt missing discovery compact placeholder:\n%s", reportPrompt)
				}
				if strings.Contains(model.requests[2].Prompt, longSearchSnippet) {
					t.Fatalf("apply saturation prompt leaked raw discovery snippet:\n%s", model.requests[2].Prompt)
				}
			} else {
				if !strings.Contains(reportPrompt, longSearchSnippet) {
					t.Fatalf("dry-run Pass2 prompt missing raw discovery snippet:\n%s", reportPrompt)
				}
				if strings.Contains(reportPrompt, "[compacted discovery-only web_search snippet") {
					t.Fatalf("dry-run Pass2 prompt changed provider payload:\n%s", reportPrompt)
				}
			}

			evidenceArtifact := readReviewRunArtifactForTest(t, artifactDir, "evidence_post_pass1.md")
			if !strings.Contains(evidenceArtifact, longSearchSnippet) ||
				!strings.Contains(evidenceArtifact, externalSnippet) ||
				!strings.Contains(evidenceArtifact, secondExternalSnippet) {
				t.Fatalf("raw post-pass1 evidence artifact was compacted:\n%s", evidenceArtifact)
			}
			reportPromptArtifact := readReviewRunArtifactForTest(t, artifactDir, "report_prompt.md")
			if tt.wantPromptCompact && strings.Contains(reportPromptArtifact, longSearchSnippet) {
				t.Fatalf("apply report prompt artifact does not match compacted provider prompt:\n%s", reportPromptArtifact)
			}
			if !tt.wantPromptCompact && !strings.Contains(reportPromptArtifact, longSearchSnippet) {
				t.Fatalf("dry-run report prompt artifact was compacted:\n%s", reportPromptArtifact)
			}

			reductionReport := runner.PromptReductionReport()
			if reductionReport.ClassifierCounts["review_web_search_discovery"] != 1 ||
				reductionReport.ReplacedCount != tt.wantReplacementCnt {
				t.Fatalf("PromptReductionReport() = %#v, want web discovery candidate and %d replacement", reductionReport, tt.wantReplacementCnt)
			}
		})
	}
}

func TestCompactReviewWebSearchDiscoveryEvidenceKeepsUnsafeExternalSupport(t *testing.T) {
	longSearchSnippet := strings.Repeat("RAW_DISCOVERY_SNIPPET_MUST_STAY_WITH_UNSAFE_EXTERNAL_SUPPORT ", 400)
	tests := []struct {
		name     string
		evidence externaldoc.WebSearchEvidence
	}{
		{
			name: "unknown source credibility",
			evidence: reviewWebSearchDiscoveryCompactEvidenceForTest(longSearchSnippet, []externaldoc.Evidence{
				reviewExternalDocEvidenceForDiscoveryCompactTest("external-doc-unknown", externaldoc.SourceCredibilityUnknown, false, "unknown source snippet"),
				reviewExternalDocEvidenceForDiscoveryCompactTest("external-doc-official", externaldoc.SourceCredibilityOfficialCandidate, false, "official source snippet"),
			}),
		},
		{
			name: "truncated external doc",
			evidence: reviewWebSearchDiscoveryCompactEvidenceForTest(longSearchSnippet, []externaldoc.Evidence{
				reviewExternalDocEvidenceForDiscoveryCompactTest("external-doc-truncated", externaldoc.SourceCredibilityOfficialCandidate, true, "truncated source snippet"),
				reviewExternalDocEvidenceForDiscoveryCompactTest("external-doc-official", externaldoc.SourceCredibilityOfficialCandidate, false, "official source snippet"),
			}),
		},
		{
			name: "official confirmation false",
			evidence: reviewWebSearchDiscoveryCompactEvidenceForTest(longSearchSnippet, []externaldoc.Evidence{
				reviewExternalDocEvidenceForDiscoveryCompactTest("external-doc-single", externaldoc.SourceCredibilityOfficialCandidate, false, "single official snippet"),
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bundle := newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")
			bundle.WebSearchEvidence = tt.evidence
			compacted, savedBytes, savedTokens, ok := reviewpromptreduction.CompactReviewWebSearchDiscoveryEvidence(bundle)
			if ok || savedBytes != 0 || savedTokens != 0 || compacted.WebSearchEvidence.Enabled {
				t.Fatalf("reviewpromptreduction.CompactReviewWebSearchDiscoveryEvidence() = (%#v, %d, %d, %t), want conservative keep", compacted.WebSearchEvidence, savedBytes, savedTokens, ok)
			}
		})
	}
}

type runnerPostPass1WebSearchEvidenceBuilder struct {
	runnerFakeEvidenceBuilder
	postEvidence externaldoc.WebSearchEvidence
	postCalls    int
	seenPlan     reviewprobeplan.ReviewProbePlan
}

func (b *runnerPostPass1WebSearchEvidenceBuilder) CollectPostPass1WebSearchEvidence(_ context.Context, _ reviewevidence.ReviewEvidenceBundle, plan reviewprobeplan.ReviewProbePlan) externaldoc.WebSearchEvidence {
	b.postCalls++
	b.seenPlan = plan
	if b.events != nil {
		*b.events = append(*b.events, "post_search")
	}
	return b.postEvidence
}

func readReviewRunArtifactForTest(t *testing.T, dir, name string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil", name, err)
	}
	return string(content)
}

func reviewWebSearchDiscoveryCompactEvidenceForTest(snippet string, docs []externaldoc.Evidence) externaldoc.WebSearchEvidence {
	return externaldoc.WebSearchEvidence{
		Enabled:  true,
		Provider: "gemini",
		Queries: []externaldoc.WebSearchEvidenceQuery{
			{
				Query:  "OpenAI Responses API previous_response_id official docs",
				Reason: "test",
				Results: []externaldoc.WebSearchEvidenceResult{
					{
						Title:        "OpenAI Responses API docs",
						URL:          "https://platform.openai.com/docs/responses",
						SourceDomain: "platform.openai.com",
						Snippet:      snippet,
					},
				},
			},
		},
		ExternalDocs: docs,
	}
}

func reviewExternalDocEvidenceForDiscoveryCompactTest(docID string, credibility externaldoc.SourceCredibility, truncated bool, content string) externaldoc.Evidence {
	return externaldoc.Evidence{
		DocID:                   docID,
		URL:                     "https://platform.openai.com/docs/" + docID,
		SourceDomain:            "platform.openai.com",
		SourceCredibility:       credibility,
		SourceCredibilityReason: "test credibility reason",
		FetchedAt:               time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		ContentHash:             "sha256:" + strings.Repeat("a", 64-len(docID)) + docID,
		Truncated:               truncated,
		Snippets: []externaldoc.SnippetEvidence{
			{
				SnippetID:   docID + "-snippet-1",
				Content:     content,
				ContentHash: "sha256:" + strings.Repeat("b", 64-len(docID)) + docID,
				Truncated:   truncated,
			},
		},
	}
}
