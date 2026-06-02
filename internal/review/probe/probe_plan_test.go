package probe

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDecodeReviewProbePlanJSONValidPlan(t *testing.T) {
	data := mustMarshalReviewProbePlanForTest(t, newValidReviewProbePlanForTest())

	plan, err := DecodeReviewProbePlanJSON(data)
	if err != nil {
		t.Fatalf("DecodeReviewProbePlanJSON() error = %v, want nil", err)
	}
	if err := ValidateReviewProbePlan(plan); err != nil {
		t.Fatalf("ValidateReviewProbePlan() error = %v, want nil", err)
	}
	if got, want := plan.SchemaVersion, ReviewProbePlanSchemaVersionV2; got != want {
		t.Fatalf("SchemaVersion = %q, want %q", got, want)
	}
	if got, want := len(plan.Probes), 1; got != want {
		t.Fatalf("len(Probes) = %d, want %d", got, want)
	}
}

func TestDecodeReviewProbePlanJSONRejectsUnknownFieldsAndTrailingToken(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "unknown top-level field",
			json: mustMarshalMutatedReviewProbePlanJSONForTest(t, func(plan map[string]any) {
				plan["unexpected"] = true
			}),
		},
		{
			name: "unknown nested probe field",
			json: mustMarshalMutatedReviewProbePlanJSONForTest(t, func(plan map[string]any) {
				mustFirstReviewProbePlanProbeJSONForTest(t, plan)["unexpected"] = true
			}),
		},
		{
			name: "unknown nested command field",
			json: mustMarshalMutatedReviewProbePlanJSONForTest(t, func(plan map[string]any) {
				mustFirstReviewProbePlanCommandJSONForTest(t, plan)["unexpected"] = true
			}),
		},
		{
			name: "trailing JSON token",
			json: string(mustMarshalReviewProbePlanForTest(t, newValidReviewProbePlanForTest())) + ` {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeReviewProbePlanJSON([]byte(tt.json))
			if err == nil {
				t.Fatal("DecodeReviewProbePlanJSON() error = nil, want error")
			}
		})
	}
}

func TestValidateReviewProbePlanBasicContract(t *testing.T) {
	tests := []struct {
		name        string
		plan        func() ReviewProbePlan
		errContains string
	}{
		{
			name: "valid plan",
			plan: newValidReviewProbePlanForTest,
		},
		{
			name: "invalid schema_version",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.SchemaVersion = ReviewProbePlanSchemaVersionV1
				return plan
			},
			errContains: "schema_version",
		},
		{
			name: "invalid target_kind",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.TargetKind = TargetKind("workspace_snapshot")
				return plan
			},
			errContains: "target_kind",
		},
		{
			name: "duplicate probe id",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes = append(plan.Probes, plan.Probes[0])
				return plan
			},
			errContains: "probes[1].id",
		},
		{
			name: "probe id with leading whitespace",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes[0].ID = " probe-1"
				return plan
			},
			errContains: "probes[0].id",
		},
		{
			name: "probe id with internal whitespace",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes[0].ID = "probe 1"
				return plan
			},
			errContains: "probes[0].id",
		},
		{
			name: "invalid mode",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes[0].Mode = ReviewProbeMode("unknown")
				return plan
			},
			errContains: "probes[0].mode",
		},
		{
			name: "probes empty requires no_probe_reason",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes = nil
				return plan
			},
			errContains: "no_probe_reason",
		},
		{
			name: "probes non-empty rejects no_probe_reason",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.NoProbeReason = "All relevant surfaces were already checked."
				return plan
			},
			errContains: "no_probe_reason",
		},
		{
			name: "too many probes",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes = make([]ReviewPlannedProbe, 0, MaxReviewProbePlanProbes+1)
				for i := 0; i < MaxReviewProbePlanProbes+1; i++ {
					probe := newValidReviewProbePlanForTest().Probes[0]
					probe.ID = "probe-" + string(rune('a'+i))
					plan.Probes = append(plan.Probes, probe)
				}
				return plan
			},
			errContains: "probes",
		},
		{
			name: "negative timeout_seconds",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes[0].TimeoutSeconds = -1
				return plan
			},
			errContains: "timeout_seconds",
		},
		{
			name: "too large timeout_seconds",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes[0].TimeoutSeconds = MaxReviewProbePlanTimeoutSeconds + 1
				return plan
			},
			errContains: "timeout_seconds",
		},
		{
			name: "negative max_output_bytes",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes[0].MaxOutputBytes = -1
				return plan
			},
			errContains: "max_output_bytes",
		},
		{
			name: "too large max_output_bytes",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.Probes[0].MaxOutputBytes = MaxReviewProbePlanMaxOutputBytes + 1
				return plan
			},
			errContains: "max_output_bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReviewProbePlan(tt.plan())
			if tt.errContains == "" {
				if err != nil {
					t.Fatalf("ValidateReviewProbePlan() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("ValidateReviewProbePlan() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("ValidateReviewProbePlan() error = %q, want substring %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestValidateReviewProbePlanScopeAnalysisContract(t *testing.T) {
	tests := []struct {
		name        string
		plan        func() ReviewProbePlan
		errContains string
	}{
		{
			name: "missing impact_surfaces",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces = nil
				return plan
			},
			errContains: "impact_surfaces",
		},
		{
			name: "duplicate surface id",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces = append(plan.ImpactSurfaces, plan.ImpactSurfaces[0])
				return plan
			},
			errContains: "impact_surfaces[1].id",
		},
		{
			name: "duplicate risk id",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.CandidateRisks = append(plan.CandidateRisks, plan.CandidateRisks[0])
				return plan
			},
			errContains: "candidate_risks[1].id",
		},
		{
			name: "risk references unknown surface id",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.CandidateRisks[0].SurfaceIDs = []string{"missing-surface"}
				return plan
			},
			errContains: "candidate_risks[0].surface_ids[0]",
		},
		{
			name: "risk requires at least one surface id",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.CandidateRisks[0].SurfaceIDs = nil
				return plan
			},
			errContains: "candidate_risks[0].surface_ids",
		},
		{
			name: "unknown surface category",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces[0].Category = ReviewProbeImpactSurfaceCategory("unknown")
				return plan
			},
			errContains: "impact_surfaces[0].category",
		},
		{
			name: "unknown surface status",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces[0].Status = ReviewProbeImpactSurfaceStatus("unknown")
				return plan
			},
			errContains: "impact_surfaces[0].status",
		},
		{
			name: "unknown risk severity",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.CandidateRisks[0].Severity = ReviewGroupSeverity("unknown")
				return plan
			},
			errContains: "candidate_risks[0].severity",
		},
		{
			name: "unknown risk status",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.CandidateRisks[0].Status = ReviewProbeCandidateRiskStatus("unknown")
				return plan
			},
			errContains: "candidate_risks[0].status",
		},
		{
			name: "surface requires evidence summary or refs",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces[0].EvidenceSummary = ""
				plan.ImpactSurfaces[0].EvidenceRefs = nil
				return plan
			},
			errContains: "impact_surfaces[0] requires evidence_summary or evidence_refs",
		},
		{
			name: "risk requires evidence summary or refs",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.CandidateRisks[0].EvidenceSummary = ""
				plan.CandidateRisks[0].EvidenceRefs = nil
				return plan
			},
			errContains: "candidate_risks[0] requires evidence_summary or evidence_refs",
		},
		{
			name: "scope evidence rejects probe kind",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces[0].EvidenceSummary = ""
				plan.ImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{{Kind: ReviewEvidenceKindProbe}}
				return plan
			},
			errContains: "impact_surfaces[0].evidence_refs[0].kind",
		},
		{
			name: "scope evidence rejects probe_id",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces[0].EvidenceSummary = ""
				plan.ImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{{Kind: ReviewEvidenceKindGitStatus, ProbeID: "probe-1"}}
				return plan
			},
			errContains: "impact_surfaces[0].evidence_refs[0].probe_id",
		},
		{
			name: "scope evidence rejects command_index",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces[0].EvidenceSummary = ""
				plan.ImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{{Kind: ReviewEvidenceKindGitStatus, CommandIndex: ReviewCommandIndex(0)}}
				return plan
			},
			errContains: "impact_surfaces[0].evidence_refs[0].command_index",
		},
		{
			name: "scope evidence accepts file ref",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.ImpactSurfaces[0].EvidenceSummary = ""
				plan.ImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{{
					Kind: ReviewEvidenceKindFile,
					Path: "internal/review/probe_plan.go",
					Line: 12,
				}}
				return plan
			},
		},
		{
			name: "candidate risks may be empty",
			plan: func() ReviewProbePlan {
				return markReviewProbePlanRisklessForTest(newValidReviewProbePlanForTest())
			},
		},
		{
			name: "candidate risks empty requires reason",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.CandidateRisks = nil
				plan.Probes[0].RiskIDs = nil
				return plan
			},
			errContains: "no_candidate_risk_reason",
		},
		{
			name: "candidate risks empty rejects reason missing surface id",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.CandidateRisks = nil
				plan.Probes[0].RiskIDs = nil
				plan.NoCandidateRiskReason = "The diff evidence leaves no material candidate risk."
				return plan
			},
			errContains: "no_candidate_risk_reason",
		},
		{
			name: "candidate risks non-empty rejects no_candidate_risk_reason",
			plan: func() ReviewProbePlan {
				plan := newValidReviewProbePlanForTest()
				plan.NoCandidateRiskReason = "surface-1 has no material candidate risk."
				return plan
			},
			errContains: "no_candidate_risk_reason",
		},
		{
			name: "no-probe rejects needs_probe surface",
			plan: func() ReviewProbePlan {
				plan := newNoProbeReviewProbePlanForTest()
				plan.ImpactSurfaces[0].Status = ReviewProbeImpactSurfaceNeedsProbe
				return plan
			},
			errContains: "impact_surfaces[0].status",
		},
		{
			name: "no-probe rejects unverified risk",
			plan: func() ReviewProbePlan {
				plan := newNoProbeReviewProbePlanForTest()
				plan.CandidateRisks[0].Status = ReviewProbeCandidateRiskUnverified
				return plan
			},
			errContains: "candidate_risks[0].status",
		},
		{
			name: "no-probe rejects reason missing surface id",
			plan: func() ReviewProbePlan {
				plan := newNoProbeReviewProbePlanForTest()
				plan.NoProbeReason = "risk-1 is checked by existing evidence."
				return plan
			},
			errContains: "no_probe_reason",
		},
		{
			name: "no-probe rejects reason missing risk id",
			plan: func() ReviewProbePlan {
				plan := newNoProbeReviewProbePlanForTest()
				plan.NoProbeReason = "surface-1 is checked by existing evidence."
				return plan
			},
			errContains: "no_probe_reason",
		},
		{
			name: "no-probe accepts fully checked scope",
			plan: newNoProbeReviewProbePlanForTest,
		},
		{
			name: "no-probe accepts riskless fully checked scope",
			plan: newNoProbeRisklessReviewProbePlanForTest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReviewProbePlan(tt.plan())
			if tt.errContains == "" {
				if err != nil {
					t.Fatalf("ValidateReviewProbePlan() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("ValidateReviewProbePlan() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("ValidateReviewProbePlan() error = %q, want substring %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestValidateReviewProbePlanRejectsInvalidFilePath(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "absolute", path: "/tmp/check.go"},
		{name: "parent escape", path: "../check.go"},
		{name: "empty", path: ""},
		{name: "whitespace-only", path: " "},
		{name: "leading whitespace", path: " checks/check.go"},
		{name: "null byte", path: "checks/\x00.go"},
		{name: "windows absolute", path: `C:\repo\check.go`},
		{name: "backslash", path: `checks\check.go`},
		{name: "current segment", path: "./check.go"},
		{name: "non canonical parent segment", path: "checks/../check.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := newValidReviewProbePlanForTest()
			plan.Probes[0].Files[0].Path = tt.path

			err := ValidateReviewProbePlan(plan)
			if err == nil {
				t.Fatal("ValidateReviewProbePlan() error = nil, want error")
			}
			if !strings.Contains(err.Error(), "probes[0].files[0].path") {
				t.Fatalf("ValidateReviewProbePlan() error = %q, want file path field", err.Error())
			}
		})
	}
}

func TestValidateReviewProbePlanAllowsInternalWhitespaceInFilePath(t *testing.T) {
	plan := newValidReviewProbePlanForTest()
	plan.Probes[0].Files[0].Path = "checks/check with space.go"

	if err := ValidateReviewProbePlan(plan); err != nil {
		t.Fatalf("ValidateReviewProbePlan() error = %v, want nil for internal whitespace path", err)
	}
}

func TestValidateReviewProbePlanModeFileContract(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(*ReviewProbePlan)
		errContains string
	}{
		{
			name: "host_readonly rejects generated files",
			mutate: func(plan *ReviewProbePlan) {
				plan.Probes[0].Mode = ReviewProbeHostReadOnly
			},
			errContains: "probes[0].files",
		},
		{
			name: "host_readonly allows empty files",
			mutate: func(plan *ReviewProbePlan) {
				plan.Probes[0].Mode = ReviewProbeHostReadOnly
				plan.Probes[0].Files = nil
			},
		},
		{
			name: "scratch_only allows generated files",
			mutate: func(plan *ReviewProbePlan) {
				plan.Probes[0].Mode = ReviewProbeScratchOnly
			},
		},
		{
			name: "repo_sandbox allows generated files",
			mutate: func(plan *ReviewProbePlan) {
				plan.Probes[0].Mode = ReviewProbeRepoSandbox
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := newValidReviewProbePlanForTest()
			tt.mutate(&plan)

			err := ValidateReviewProbePlan(plan)
			if tt.errContains == "" {
				if err != nil {
					t.Fatalf("ValidateReviewProbePlan() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("ValidateReviewProbePlan() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("ValidateReviewProbePlan() error = %q, want substring %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestValidateReviewProbePlanRejectsInvalidCommandContract(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(*ReviewProbePlan)
		errContains string
	}{
		{
			name: "empty command",
			mutate: func(plan *ReviewProbePlan) {
				plan.Probes[0].Commands[0].Command = ""
			},
			errContains: "probes[0].commands[0].command",
		},
		{
			name: "command with leading whitespace",
			mutate: func(plan *ReviewProbePlan) {
				plan.Probes[0].Commands[0].Command = " go"
			},
			errContains: "probes[0].commands[0].command",
		},
		{
			name: "command with internal whitespace",
			mutate: func(plan *ReviewProbePlan) {
				plan.Probes[0].Commands[0].Command = "go test"
			},
			errContains: "probes[0].commands[0].command",
		},
		{
			name: "npm command with internal whitespace",
			mutate: func(plan *ReviewProbePlan) {
				plan.Probes[0].Commands[0].Command = "npm run"
			},
			errContains: "probes[0].commands[0].command",
		},
		{
			name: "command with null byte",
			mutate: func(plan *ReviewProbePlan) {
				plan.Probes[0].Commands[0].Command = "go\x00"
			},
			errContains: "probes[0].commands[0].command",
		},
		{
			name: "command with slash",
			mutate: func(plan *ReviewProbePlan) {
				plan.Probes[0].Commands[0].Command = "./go"
			},
			errContains: "probes[0].commands[0].command",
		},
		{
			name: "command with backslash",
			mutate: func(plan *ReviewProbePlan) {
				plan.Probes[0].Commands[0].Command = `bin\go`
			},
			errContains: "probes[0].commands[0].command",
		},
		{
			name: "arg with null byte",
			mutate: func(plan *ReviewProbePlan) {
				plan.Probes[0].Commands[0].Args = []string{"test", "\x00"}
			},
			errContains: "probes[0].commands[0].args[1]",
		},
		{
			name: "work_dir parent escape",
			mutate: func(plan *ReviewProbePlan) {
				plan.Probes[0].Commands[0].WorkDir = "../internal"
			},
			errContains: "probes[0].commands[0].work_dir",
		},
		{
			name: "work_dir backslash",
			mutate: func(plan *ReviewProbePlan) {
				plan.Probes[0].Commands[0].WorkDir = `internal\review`
			},
			errContains: "probes[0].commands[0].work_dir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := newValidReviewProbePlanForTest()
			tt.mutate(&plan)

			err := ValidateReviewProbePlan(plan)
			if err == nil {
				t.Fatal("ValidateReviewProbePlan() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("ValidateReviewProbePlan() error = %q, want substring %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestValidateReviewProbePlanAllowsCommandNamesAndPathArgs(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    []string
	}{
		{
			name:    "go command with slash arg",
			command: "go",
			args:    []string{"test", "./internal/review"},
		},
		{
			name:    "go command with split test args",
			command: "go",
			args:    []string{"run", "test"},
		},
		{
			name:    "python command with backslash arg",
			command: "python3",
			args:    []string{`path\like`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := newValidReviewProbePlanForTest()
			plan.Probes[0].Commands[0].Command = tt.command
			plan.Probes[0].Commands[0].Args = tt.args

			if err := ValidateReviewProbePlan(plan); err != nil {
				t.Fatalf("ValidateReviewProbePlan() error = %v, want nil", err)
			}
		})
	}
}

func TestValidateReviewProbePlanGeneratedFileContract(t *testing.T) {
	tests := []struct {
		name        string
		files       []ReviewPlannedProbeFile
		errContains string
	}{
		{
			name: "duplicate path",
			files: []ReviewPlannedProbeFile{
				{Path: "checks/check.txt", Content: "a"},
				{Path: "checks/check.txt", Content: "b"},
			},
			errContains: "probes[0].files[1].path",
		},
		{
			name: "distinct paths",
			files: []ReviewPlannedProbeFile{
				{Path: "checks/a.txt", Content: "a"},
				{Path: "checks/b.txt", Content: "b"},
			},
		},
		{
			name: "empty content",
			files: []ReviewPlannedProbeFile{
				{Path: "checks/empty.txt", Content: ""},
			},
		},
		{
			name: "per file content limit exceeded",
			files: []ReviewPlannedProbeFile{
				{Path: "checks/huge.txt", Content: strings.Repeat("x", MaxReviewProbePlanFileContentBytes+1)},
			},
			errContains: "probes[0].files[0].content",
		},
		{
			name:        "total content limit exceeded",
			files:       newReviewPlannedProbeFilesForTest(5, MaxReviewProbePlanFileContentBytes),
			errContains: "probes[0].files content",
		},
		{
			name:  "content limits exactly",
			files: newReviewPlannedProbeFilesForTest(4, MaxReviewProbePlanFileContentBytes),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := newValidReviewProbePlanForTest()
			plan.Probes[0].Files = tt.files

			err := ValidateReviewProbePlan(plan)
			if tt.errContains == "" {
				if err != nil {
					t.Fatalf("ValidateReviewProbePlan() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("ValidateReviewProbePlan() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("ValidateReviewProbePlan() error = %q, want substring %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestBuildReviewProbeRequestsFromPlanConvertsValidPlan(t *testing.T) {
	plan := newValidReviewProbePlanForTest()

	requests, err := BuildReviewProbeRequestsFromPlan(plan)
	if err != nil {
		t.Fatalf("BuildReviewProbeRequestsFromPlan() error = %v, want nil", err)
	}
	if got, want := len(requests), 1; got != want {
		t.Fatalf("len(requests) = %d, want %d", got, want)
	}

	req := requests[0]
	if got, want := req.ID, "probe-1"; got != want {
		t.Fatalf("ID = %q, want %q", got, want)
	}
	if got, want := req.Mode, ReviewProbeRepoSandbox; got != want {
		t.Fatalf("Mode = %q, want %q", got, want)
	}
	if got, want := req.Timeout, 30*time.Second; got != want {
		t.Fatalf("Timeout = %v, want %v", got, want)
	}
	if got, want := req.MaxOutputBytes, int64(4096); got != want {
		t.Fatalf("MaxOutputBytes = %d, want %d", got, want)
	}
	if got, want := len(req.Commands), 2; got != want {
		t.Fatalf("len(Commands) = %d, want %d", got, want)
	}
	if got := req.Commands[0].WorkDir; got != "" {
		t.Fatalf("Commands[0].WorkDir = %q, want default empty workdir", got)
	}
	if got, want := req.Commands[1].WorkDir, "internal/review"; got != want {
		t.Fatalf("Commands[1].WorkDir = %q, want %q", got, want)
	}
	if got, want := req.Commands[1].Args[0], `path\like`; got != want {
		t.Fatalf("Commands[1].Args[0] = %q, want %q", got, want)
	}
	if got, want := req.Files[0].Path, "checks/check.txt"; got != want {
		t.Fatalf("Files[0].Path = %q, want %q", got, want)
	}
}

func TestBuildReviewProbeRequestsFromPlanNoProbePlanReturnsEmptySlice(t *testing.T) {
	plan := newNoProbeReviewProbePlanForTest()

	requests, err := BuildReviewProbeRequestsFromPlan(plan)
	if err != nil {
		t.Fatalf("BuildReviewProbeRequestsFromPlan() error = %v, want nil", err)
	}
	if requests == nil {
		t.Fatal("BuildReviewProbeRequestsFromPlan() requests = nil, want non-nil empty slice")
	}
	if got := len(requests); got != 0 {
		t.Fatalf("len(requests) = %d, want 0", got)
	}
}

func TestBuildReviewProbeRequestsFromPlanCopiesSlices(t *testing.T) {
	plan := newValidReviewProbePlanForTest()

	requests, err := BuildReviewProbeRequestsFromPlan(plan)
	if err != nil {
		t.Fatalf("BuildReviewProbeRequestsFromPlan() error = %v, want nil", err)
	}

	plan.Probes[0].Commands[0].Args[0] = "changed"
	plan.Probes[0].Files[0].Path = "changed.txt"

	if got, want := requests[0].Commands[0].Args[0], "test"; got != want {
		t.Fatalf("request command args share DTO slice: got %q, want %q", got, want)
	}
	if got, want := requests[0].Files[0].Path, "checks/check.txt"; got != want {
		t.Fatalf("request files share DTO slice: got %q, want %q", got, want)
	}
}

func TestReviewProbePlanDecodeValidateConvertIsDeterministic(t *testing.T) {
	data := mustMarshalReviewProbePlanForTest(t, newValidReviewProbePlanForTest())

	firstPlan, err := DecodeReviewProbePlanJSON(data)
	if err != nil {
		t.Fatalf("first DecodeReviewProbePlanJSON() error = %v, want nil", err)
	}
	if err := ValidateReviewProbePlan(firstPlan); err != nil {
		t.Fatalf("first ValidateReviewProbePlan() error = %v, want nil", err)
	}
	firstRequests, err := BuildReviewProbeRequestsFromPlan(firstPlan)
	if err != nil {
		t.Fatalf("first BuildReviewProbeRequestsFromPlan() error = %v, want nil", err)
	}

	secondPlan, err := DecodeReviewProbePlanJSON(data)
	if err != nil {
		t.Fatalf("second DecodeReviewProbePlanJSON() error = %v, want nil", err)
	}
	if err := ValidateReviewProbePlan(secondPlan); err != nil {
		t.Fatalf("second ValidateReviewProbePlan() error = %v, want nil", err)
	}
	secondRequests, err := BuildReviewProbeRequestsFromPlan(secondPlan)
	if err != nil {
		t.Fatalf("second BuildReviewProbeRequestsFromPlan() error = %v, want nil", err)
	}

	if !reflect.DeepEqual(firstPlan, secondPlan) {
		t.Fatalf("decoded plans differ:\nfirst: %#v\nsecond: %#v", firstPlan, secondPlan)
	}
	if !reflect.DeepEqual(firstRequests, secondRequests) {
		t.Fatalf("converted requests differ:\nfirst: %#v\nsecond: %#v", firstRequests, secondRequests)
	}
}

func newValidReviewProbePlanForTest() ReviewProbePlan {
	return ReviewProbePlan{
		SchemaVersion: ReviewProbePlanSchemaVersionV2,
		TargetKind:    TargetCurrentChanges,
		Summary:       "Probe current changes.",
		ImpactSurfaces: []ReviewProbeImpactSurface{
			{
				ID:              "surface-1",
				Summary:         "Probe plan validation changed.",
				Category:        ReviewProbeImpactSurfaceValidator,
				EvidenceSummary: "Diff touches internal/review/probe_plan_validate.go.",
				Status:          ReviewProbeImpactSurfaceNeedsProbe,
				Reason:          "Focused tests should verify the contract.",
			},
		},
		CandidateRisks: []ReviewProbeCandidateRisk{
			{
				ID:                   "risk-1",
				Summary:              "Validation could accept an invalid probe plan.",
				Severity:             ReviewGroupSeverityMedium,
				SurfaceIDs:           []string{"surface-1"},
				EvidenceSummary:      "Validation code owns the probe plan contract.",
				VerificationStrategy: "Run focused review tests.",
				Status:               ReviewProbeCandidateRiskNeedsProbe,
			},
		},
		Probes: []ReviewPlannedProbe{
			{
				ID:             "probe-1",
				SurfaceIDs:     []string{"surface-1"},
				RiskIDs:        []string{"risk-1"},
				Purpose:        "Confirm or falsify risk-1 for surface-1 by running focused review tests.",
				Mode:           ReviewProbeRepoSandbox,
				TimeoutSeconds: 30,
				MaxOutputBytes: 4096,
				Commands: []ReviewPlannedProbeCommand{
					{
						Command: "go",
						Args:    []string{"test", "./internal/review"},
						WorkDir: ".",
					},
					{
						Command: "rg",
						Args:    []string{`path\like`, "ReviewProbePlan"},
						WorkDir: "internal/review",
					},
				},
				Files: []ReviewPlannedProbeFile{
					{
						Path:    "checks/check.txt",
						Content: "ok\n",
					},
				},
			},
		},
	}
}

func newNoProbeReviewProbePlanForTest() ReviewProbePlan {
	return markReviewProbePlanCheckedWithoutProbesForTest(newValidReviewProbePlanForTest())
}

func newNoProbeRisklessReviewProbePlanForTest() ReviewProbePlan {
	plan := markReviewProbePlanCheckedWithoutProbesForTest(newValidReviewProbePlanForTest())
	plan = markReviewProbePlanRisklessForTest(plan)
	plan.NoProbeReason = "surface-1 is checked by existing evidence."
	return plan
}

func markReviewProbePlanRisklessForTest(plan ReviewProbePlan) ReviewProbePlan {
	plan.CandidateRisks = nil
	for i := range plan.Probes {
		plan.Probes[i].RiskIDs = nil
	}
	plan.NoCandidateRiskReason = noCandidateRiskReasonForReviewProbePlanForTest(plan)
	return plan
}

func noCandidateRiskReasonForReviewProbePlanForTest(plan ReviewProbePlan) string {
	surfaceIDs := make([]string, 0, len(plan.ImpactSurfaces))
	for _, surface := range plan.ImpactSurfaces {
		surfaceIDs = append(surfaceIDs, surface.ID)
	}
	return "impact surfaces " + strings.Join(surfaceIDs, ", ") + " were reviewed from available evidence and leave no material candidate risk."
}

func markReviewProbePlanCheckedWithoutProbesForTest(plan ReviewProbePlan) ReviewProbePlan {
	plan.ImpactSurfaces[0].Status = ReviewProbeImpactSurfaceChecked
	plan.ImpactSurfaces[0].Reason = "Existing evidence covers surface-1."
	plan.CandidateRisks[0].Status = ReviewProbeCandidateRiskCheckedByEvidence
	plan.CandidateRisks[0].VerificationStrategy = "No probe is needed."
	plan.Probes = []ReviewPlannedProbe{}
	plan.NoProbeReason = "surface-1 and risk-1 are checked by existing evidence."
	return plan
}

func mustMarshalReviewProbePlanForTest(t *testing.T, plan ReviewProbePlan) []byte {
	t.Helper()

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v, want nil", err)
	}
	return data
}

func mustMarshalMutatedReviewProbePlanJSONForTest(t *testing.T, mutate func(map[string]any)) string {
	t.Helper()

	var rawPlan map[string]any
	if err := json.Unmarshal(mustMarshalReviewProbePlanForTest(t, newValidReviewProbePlanForTest()), &rawPlan); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, want nil", err)
	}
	mutate(rawPlan)

	data, err := json.Marshal(rawPlan)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v, want nil", err)
	}
	return string(data)
}

func mustFirstReviewProbePlanProbeJSONForTest(t *testing.T, plan map[string]any) map[string]any {
	t.Helper()

	probes, ok := plan["probes"].([]any)
	if !ok {
		t.Fatalf("probes expected array, got %T", plan["probes"])
	}
	if len(probes) == 0 {
		t.Fatal("probes expected at least one entry")
	}
	probe, ok := probes[0].(map[string]any)
	if !ok {
		t.Fatalf("probes[0] expected object, got %T", probes[0])
	}
	return probe
}

func mustFirstReviewProbePlanCommandJSONForTest(t *testing.T, plan map[string]any) map[string]any {
	t.Helper()

	probe := mustFirstReviewProbePlanProbeJSONForTest(t, plan)
	commands, ok := probe["commands"].([]any)
	if !ok {
		t.Fatalf("probes[0].commands expected array, got %T", probe["commands"])
	}
	if len(commands) == 0 {
		t.Fatal("probes[0].commands expected at least one entry")
	}
	command, ok := commands[0].(map[string]any)
	if !ok {
		t.Fatalf("probes[0].commands[0] expected object, got %T", commands[0])
	}
	return command
}

func newReviewPlannedProbeFilesForTest(count, contentBytes int) []ReviewPlannedProbeFile {
	files := make([]ReviewPlannedProbeFile, 0, count)
	for i := 0; i < count; i++ {
		files = append(files, ReviewPlannedProbeFile{
			Path:    "checks/file-" + string(rune('a'+i)) + ".txt",
			Content: strings.Repeat("x", contentBytes),
		})
	}
	return files
}
