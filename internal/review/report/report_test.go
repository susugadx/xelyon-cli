package report

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

func TestReviewReportJSONRoundTrip(t *testing.T) {
	generatedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	const (
		primaryFindingID    = "finding-1"
		reportPassFindingID = "finding-2"
	)
	original := ReviewReport{
		SchemaVersion:             ReviewReportSchemaVersionV2,
		TargetKind:                domain.TargetCurrentChanges,
		CustomInstructions:        "correctness を優先",
		GeneratedAt:               generatedAt,
		OverallVerificationStatus: ReviewVerificationPartiallyVerified,
		Verdict:                   ReviewVerdictHasFindings,
		Summary:                   "review 全体の要約",
		RootCauseGroups: []ReviewRootCauseGroup{
			{
				ID:                 "rc-1",
				Title:              "入力検証漏れ",
				Summary:            "未検証入力が state 更新へ到達する",
				Severity:           ReviewGroupSeverityHigh,
				VerificationStatus: ReviewVerificationVerified,
				FixStrategy:        "path policy の評価を command 実行前へ統一する",
				DoNotFixBy: []string{
					"call site ごとの sanitize 分岐追加",
					"post-hoc な denylist のみの拡張",
				},
				VerificationPlan: []string{
					"probe_command_exec の regression test 追加",
					"symlink 経由 path traversal の再現ケース固定",
				},
				Findings: []ReviewFinding{
					{
						ID:      primaryFindingID,
						Title:   "path traversal の可能性",
						Summary: "relative path が sanitize されない",
						EvidenceRefs: []ReviewEvidenceRef{
							{
								Kind:         "probe_command",
								Summary:      "cat で想定外ファイルへ到達",
								ProbeID:      "probe-1",
								CommandIndex: ReviewCommandIndex(0),
								Path:         "internal/review/probe_command_exec.go",
								Line:         20,
								Snippet:      "exec.CommandContext(...)",
							},
						},
						CheckedSurfaces: []ReviewSurfaceCoverage{
							{
								SurfaceID: "command_execution",
								Summary:   "host_readonly policy",
							},
						},
						UnverifiedSurfaces: []ReviewSurfaceCoverage{
							{
								SurfaceID: "provider_specific_policy",
								Summary:   "provider ごとの差分",
							},
						},
						ResidualRisks: []ReviewResidualRisk{
							{
								ID:                  "risk-1",
								Summary:             "symlink 経由の迂回余地",
								SuggestedMitigation: "path policy を inode ベースへ強化",
							},
						},
					},
					{
						ID:      reportPassFindingID,
						Title:   "Pass2 追加 finding",
						Summary: "report pass で追加確認した finding",
						EvidenceRefs: []ReviewEvidenceRef{
							{
								Kind:    ReviewEvidenceKindFile,
								Path:    "internal/review/report_validation.go",
								Line:    1,
								Summary: "Pass2 追加 finding の証跡",
							},
						},
					},
				},
				CheckedSurfaces: []ReviewSurfaceCoverage{
					{
						SurfaceID: "policy_read",
						Summary:   "read-only policy の許可範囲",
					},
				},
				UnverifiedSurfaces: []ReviewSurfaceCoverage{
					{
						SurfaceID: "policy_search",
						Summary:   "search 系 args の境界条件",
					},
				},
				ResidualRisks: []ReviewResidualRisk{
					{
						ID:                  "risk-2",
						Summary:             "OS 差分由来の path normalize ずれ",
						SuggestedMitigation: "windows path test を追加",
					},
				},
			},
		},
		ProbeSummaries: []ReviewProbeSummary{
			{
				ProbeID:         "probe-1",
				Mode:            domain.ReviewProbeScratchOnly,
				Status:          domain.ReviewProbeMutatedWorktree,
				MutatedWorktree: true,
				MutatedFiles:    []string{"keep.txt"},
				OutputTruncated: true,
				Error:           "probe command changed the working tree",
				Commands: []ReviewProbeCommandSummary{
					{
						Command:         "sh",
						Args:            []string{"-c", "echo hello"},
						WorkDir:         "/tmp/review",
						Status:          domain.ReviewProbePassed,
						ExitCode:        0,
						OutputTruncated: true,
						DurationMs:      120,
					},
				},
			},
		},
		CheckedSurfaces: []ReviewSurfaceCoverage{
			{
				SurfaceID: "overall_contract",
				Summary:   "request/response 契約",
			},
		},
		UnverifiedSurfaces: []ReviewSurfaceCoverage{
			{
				SurfaceID: "windows_path",
				Summary:   "windows 実機検証",
			},
		},
		ResidualRisks: []ReviewResidualRisk{
			{
				ID:                  "risk-3",
				Summary:             "cross-platform の path separator 差",
				SuggestedMitigation: "OS matrix test を追加",
			},
		},
		ScopeCoverage: &ReviewReportScopeCoverage{
			ReviewedImpactSurfaces: []ReviewReportImpactSurfaceCoverage{
				{
					SurfaceID:    "surface-1",
					Status:       ReviewReportImpactSurfaceFinding,
					Summary:      "surface-1 は finding-1 に接続",
					EvidenceRefs: []ReviewEvidenceRef{{Kind: ReviewEvidenceKindDiff, Path: "internal/review/report_types.go"}},
					FindingIDs:   []string{primaryFindingID},
				},
			},
			ReviewedCandidateRisks: []ReviewReportCandidateRiskCoverage{
				{
					RiskID:       "risk-1",
					Status:       ReviewReportCandidateRiskFinding,
					Summary:      "risk-1 は finding-1 として確認",
					EvidenceRefs: []ReviewEvidenceRef{{Kind: ReviewEvidenceKindProbe, ProbeID: "probe-1"}},
					FindingIDs:   []string{primaryFindingID},
				},
			},
			NewFindingsFromReportPass: []ReviewReportPassFindingCoverage{
				{
					FindingIDs:   []string{reportPassFindingID},
					Summary:      "Pass2 で追加確認した finding",
					EvidenceRefs: []ReviewEvidenceRef{{Kind: ReviewEvidenceKindFile, Path: "internal/review/report_validation.go"}},
				},
			},
		},
		ComputedSummary: &ReviewReportComputedSummary{
			RootCauseGroupCount:       1,
			FindingCount:              2,
			CheckedSurfaceCount:       0,
			FindingSurfaceCount:       1,
			UnverifiedSurfaceCount:    0,
			ResidualSurfaceCount:      0,
			CandidateRiskCount:        1,
			DismissedRiskCount:        0,
			FindingRiskCount:          1,
			UnverifiedRiskCount:       0,
			ResidualRiskCount:         0,
			NewReportPassFindingCount: 1,
			ProbeCount:                1,
			PassedProbeCount:          0,
			FailedProbeCount:          0,
			TimedOutProbeCount:        0,
			BlockedProbeCount:         0,
			MutatedWorktreeProbeCount: 1,
		},
	}

	if err := ValidateReviewReport(original); err != nil {
		t.Fatalf("ValidateReviewReport(original) error = %v, want nil", err)
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var got ReviewReport
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if err := ValidateReviewReport(got); err != nil {
		t.Fatalf("ValidateReviewReport(got) error = %v, want nil", err)
	}

	if !reflect.DeepEqual(got, original) {
		t.Fatalf("round-trip mismatch:\n got  = %#v\n want = %#v", got, original)
	}
}

func TestReviewEvidenceRefJSONCommandIndexZero(t *testing.T) {
	ref := ReviewEvidenceRef{
		Kind:         "probe_command",
		CommandIndex: ReviewCommandIndex(0),
	}

	data, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	raw, exists := parsed["command_index"]
	if !exists {
		t.Fatal("command_index should be present when CommandIndex is ReviewCommandIndex(0)")
	}
	index, ok := raw.(float64)
	if !ok {
		t.Fatalf("command_index type = %T, want float64", raw)
	}
	if index != 0 {
		t.Fatalf("command_index = %v, want 0", index)
	}
}

func TestReviewEvidenceRefJSONCommandIndexOmittedWhenNil(t *testing.T) {
	ref := ReviewEvidenceRef{Kind: "probe_command"}

	data, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if _, exists := parsed["command_index"]; exists {
		t.Fatal("command_index should be omitted when CommandIndex is nil")
	}
}

func TestReviewReportJSONOmitempty(t *testing.T) {
	report := ReviewReport{
		SchemaVersion:             ReviewReportSchemaVersionV2,
		TargetKind:                domain.TargetCurrentChanges,
		GeneratedAt:               time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
		OverallVerificationStatus: ReviewVerificationPartiallyVerified,
		Verdict:                   ReviewVerdictHasFindings,
		RootCauseGroups: []ReviewRootCauseGroup{
			{
				ID:                 "rc-omitempty",
				Title:              "optional fields の確認",
				Severity:           ReviewGroupSeverityLow,
				VerificationStatus: ReviewVerificationVerified,
				Findings: []ReviewFinding{
					{
						Title:              "finding-omitempty",
						EvidenceRefs:       []ReviewEvidenceRef{},
						CheckedSurfaces:    []ReviewSurfaceCoverage{},
						UnverifiedSurfaces: []ReviewSurfaceCoverage{},
						ResidualRisks:      []ReviewResidualRisk{},
					},
				},
				CheckedSurfaces:    []ReviewSurfaceCoverage{},
				UnverifiedSurfaces: []ReviewSurfaceCoverage{},
				ResidualRisks:      []ReviewResidualRisk{},
			},
		},
		ProbeSummaries:     []ReviewProbeSummary{},
		CheckedSurfaces:    []ReviewSurfaceCoverage{},
		UnverifiedSurfaces: []ReviewSurfaceCoverage{},
		ResidualRisks:      []ReviewResidualRisk{},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if _, exists := parsed["probe_summaries"]; exists {
		t.Fatal("probe_summaries should be omitted when empty")
	}
	if _, exists := parsed["checked_surfaces"]; exists {
		t.Fatal("checked_surfaces should be omitted when empty")
	}
	if _, exists := parsed["unverified_surfaces"]; exists {
		t.Fatal("unverified_surfaces should be omitted when empty")
	}
	if _, exists := parsed["residual_risks"]; exists {
		t.Fatal("residual_risks should be omitted when empty")
	}
	if _, exists := parsed["summary"]; exists {
		t.Fatal("summary should be omitted when empty")
	}
	if _, exists := parsed["scope_coverage"]; exists {
		t.Fatal("scope_coverage should be omitted when nil")
	}
	if _, exists := parsed["computed_summary"]; exists {
		t.Fatal("computed_summary should be omitted when nil")
	}
	if rawVerdict, exists := parsed["verdict"]; !exists {
		t.Fatal("verdict should be present")
	} else if rawVerdict != string(ReviewVerdictHasFindings) {
		t.Fatalf("verdict = %#v, want %q", rawVerdict, ReviewVerdictHasFindings)
	}

	rootGroupsRaw, ok := parsed["root_cause_groups"].([]any)
	if !ok || len(rootGroupsRaw) != 1 {
		t.Fatalf("root_cause_groups = %#v, want length 1", parsed["root_cause_groups"])
	}
	groupMap, ok := rootGroupsRaw[0].(map[string]any)
	if !ok {
		t.Fatalf("root_cause_groups[0] type = %T, want map[string]any", rootGroupsRaw[0])
	}
	if _, exists := groupMap["checked_surfaces"]; exists {
		t.Fatal("group.checked_surfaces should be omitted when empty")
	}
	if _, exists := groupMap["fix_strategy"]; exists {
		t.Fatal("group.fix_strategy should be omitted when empty")
	}
	if _, exists := groupMap["do_not_fix_by"]; exists {
		t.Fatal("group.do_not_fix_by should be omitted when empty")
	}
	if _, exists := groupMap["verification_plan"]; exists {
		t.Fatal("group.verification_plan should be omitted when empty")
	}
	if _, exists := groupMap["unverified_surfaces"]; exists {
		t.Fatal("group.unverified_surfaces should be omitted when empty")
	}
	if _, exists := groupMap["residual_risks"]; exists {
		t.Fatal("group.residual_risks should be omitted when empty")
	}

	findingsRaw, ok := groupMap["findings"].([]any)
	if !ok || len(findingsRaw) != 1 {
		t.Fatalf("group.findings = %#v, want length 1", groupMap["findings"])
	}
	findingMap, ok := findingsRaw[0].(map[string]any)
	if !ok {
		t.Fatalf("group.findings[0] type = %T, want map[string]any", findingsRaw[0])
	}
	if _, exists := findingMap["evidence_refs"]; exists {
		t.Fatal("finding.evidence_refs should be omitted when empty")
	}
	if _, exists := findingMap["checked_surfaces"]; exists {
		t.Fatal("finding.checked_surfaces should be omitted when empty")
	}
	if _, exists := findingMap["unverified_surfaces"]; exists {
		t.Fatal("finding.unverified_surfaces should be omitted when empty")
	}
	if _, exists := findingMap["residual_risks"]; exists {
		t.Fatal("finding.residual_risks should be omitted when empty")
	}
}
