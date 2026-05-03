package review

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestReviewReportJSONRoundTrip(t *testing.T) {
	generatedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	original := ReviewReport{
		SchemaVersion:             ReviewReportSchemaVersionV1,
		TargetKind:                TargetCurrentChanges,
		CustomInstructions:        "correctness を優先",
		GeneratedAt:               generatedAt,
		OverallVerificationStatus: ReviewVerificationPartiallyVerified,
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
						ID:      "finding-1",
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
				Mode:            ReviewProbeScratchOnly,
				Status:          ReviewProbeMutatedWorktree,
				MutatedWorktree: true,
				MutatedFiles:    []string{"keep.txt"},
				OutputTruncated: true,
				Error:           "probe command changed the working tree",
				Commands: []ReviewProbeCommandSummary{
					{
						Command:         "sh",
						Args:            []string{"-c", "echo hello"},
						WorkDir:         "/tmp/review",
						Status:          ReviewProbePassed,
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
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var got ReviewReport
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
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

func TestBuildReviewProbeSummaries(t *testing.T) {
	results := []ReviewProbeResult{
		{
			ID:     "passed",
			Mode:   ReviewProbeHostReadOnly,
			Status: ReviewProbePassed,
			CommandResults: []ReviewProbeCommandResult{
				{
					Command:  "cat",
					Args:     []string{"keep.txt"},
					WorkDir:  "/repo",
					Status:   ReviewProbePassed,
					ExitCode: 0,
					Duration: 42 * time.Millisecond,
				},
			},
		},
		{
			ID:     "failed",
			Mode:   ReviewProbeScratchOnly,
			Status: ReviewProbeFailed,
			Error:  "probe command failed: go test ./...",
			CommandResults: []ReviewProbeCommandResult{
				{
					Command:         "go",
					Args:            []string{"test", "./..."},
					WorkDir:         "/scratch",
					Status:          ReviewProbeFailed,
					ExitCode:        1,
					OutputTruncated: true,
					Error:           "exit status 1",
					Duration:        2300 * time.Millisecond,
				},
			},
		},
		{
			ID:     "blocked",
			Mode:   ReviewProbeHostReadOnly,
			Status: ReviewProbeBlocked,
			Error:  "probe command blocked: rm -rf .",
		},
		{
			ID:     "timed_out",
			Mode:   ReviewProbeRepoSandbox,
			Status: ReviewProbeTimedOut,
			Error:  "probe command timed out: sleep 10",
		},
		{
			ID:              "mutated",
			Mode:            ReviewProbeRepoSandbox,
			Status:          ReviewProbeMutatedWorktree,
			MutatedWorktree: true,
			MutatedFiles:    []string{"keep.txt", "tmp/new.txt"},
			OutputTruncated: true,
			Error:           "probe command changed the working tree",
		},
	}

	got := BuildReviewProbeSummaries(results)
	if len(got) != len(results) {
		t.Fatalf("len(BuildReviewProbeSummaries()) = %d, want %d", len(got), len(results))
	}

	byID := make(map[string]ReviewProbeSummary, len(got))
	for _, summary := range got {
		byID[summary.ProbeID] = summary
	}

	if byID["passed"].Status != ReviewProbePassed {
		t.Fatalf("passed.Status = %q, want %q", byID["passed"].Status, ReviewProbePassed)
	}
	if byID["failed"].Status != ReviewProbeFailed {
		t.Fatalf("failed.Status = %q, want %q", byID["failed"].Status, ReviewProbeFailed)
	}
	if byID["blocked"].Status != ReviewProbeBlocked {
		t.Fatalf("blocked.Status = %q, want %q", byID["blocked"].Status, ReviewProbeBlocked)
	}
	if byID["timed_out"].Status != ReviewProbeTimedOut {
		t.Fatalf("timed_out.Status = %q, want %q", byID["timed_out"].Status, ReviewProbeTimedOut)
	}
	if byID["mutated"].Status != ReviewProbeMutatedWorktree {
		t.Fatalf("mutated.Status = %q, want %q", byID["mutated"].Status, ReviewProbeMutatedWorktree)
	}

	failed := byID["failed"]
	if len(failed.Commands) != 1 {
		t.Fatalf("len(failed.Commands) = %d, want 1", len(failed.Commands))
	}
	cmd := failed.Commands[0]
	if cmd.Command != "go" {
		t.Fatalf("failed.Commands[0].Command = %q, want go", cmd.Command)
	}
	if cmd.Status != ReviewProbeFailed {
		t.Fatalf("failed.Commands[0].Status = %q, want %q", cmd.Status, ReviewProbeFailed)
	}
	if cmd.ExitCode != 1 {
		t.Fatalf("failed.Commands[0].ExitCode = %d, want 1", cmd.ExitCode)
	}
	if !cmd.OutputTruncated {
		t.Fatal("failed.Commands[0].OutputTruncated = false, want true")
	}
	if cmd.DurationMs != 2300 {
		t.Fatalf("failed.Commands[0].DurationMs = %d, want 2300", cmd.DurationMs)
	}

	mutated := byID["mutated"]
	if !mutated.MutatedWorktree {
		t.Fatal("mutated.MutatedWorktree = false, want true")
	}
	if !reflect.DeepEqual(mutated.MutatedFiles, []string{"keep.txt", "tmp/new.txt"}) {
		t.Fatalf("mutated.MutatedFiles = %#v", mutated.MutatedFiles)
	}
	if !mutated.OutputTruncated {
		t.Fatal("mutated.OutputTruncated = false, want true")
	}
}

func TestReviewReportJSONOmitempty(t *testing.T) {
	report := ReviewReport{
		SchemaVersion:             ReviewReportSchemaVersionV1,
		TargetKind:                TargetCurrentChanges,
		GeneratedAt:               time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
		OverallVerificationStatus: ReviewVerificationUnverified,
		RootCauseGroups: []ReviewRootCauseGroup{
			{
				ID:                 "rc-omitempty",
				Title:              "optional fields の確認",
				Severity:           ReviewGroupSeverityLow,
				VerificationStatus: ReviewVerificationNotApplicable,
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

func TestNewReviewReportSkeleton(t *testing.T) {
	req := NewCurrentChangesRequest("safety first")
	generatedAt := time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)

	got := NewReviewReportSkeleton(req, generatedAt)
	if got.SchemaVersion != ReviewReportSchemaVersionV1 {
		t.Fatalf("SchemaVersion = %q, want %q", got.SchemaVersion, ReviewReportSchemaVersionV1)
	}
	if got.TargetKind != TargetCurrentChanges {
		t.Fatalf("TargetKind = %q, want %q", got.TargetKind, TargetCurrentChanges)
	}
	if got.CustomInstructions != "safety first" {
		t.Fatalf("CustomInstructions = %q, want %q", got.CustomInstructions, "safety first")
	}
	if !got.GeneratedAt.Equal(generatedAt) {
		t.Fatalf("GeneratedAt = %s, want %s", got.GeneratedAt, generatedAt)
	}
	if got.OverallVerificationStatus != ReviewVerificationUnverified {
		t.Fatalf("OverallVerificationStatus = %q, want %q", got.OverallVerificationStatus, ReviewVerificationUnverified)
	}
	if got.RootCauseGroups == nil || len(got.RootCauseGroups) != 0 {
		t.Fatalf("RootCauseGroups = %#v, want non-nil empty slice", got.RootCauseGroups)
	}
	if got.ProbeSummaries == nil || len(got.ProbeSummaries) != 0 {
		t.Fatalf("ProbeSummaries = %#v, want non-nil empty slice", got.ProbeSummaries)
	}
	if got.CheckedSurfaces == nil || len(got.CheckedSurfaces) != 0 {
		t.Fatalf("CheckedSurfaces = %#v, want non-nil empty slice", got.CheckedSurfaces)
	}
	if got.UnverifiedSurfaces == nil || len(got.UnverifiedSurfaces) != 0 {
		t.Fatalf("UnverifiedSurfaces = %#v, want non-nil empty slice", got.UnverifiedSurfaces)
	}
	if got.ResidualRisks == nil || len(got.ResidualRisks) != 0 {
		t.Fatalf("ResidualRisks = %#v, want non-nil empty slice", got.ResidualRisks)
	}
}
