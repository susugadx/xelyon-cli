package review

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
	if got, want := plan.SchemaVersion, ReviewProbePlanSchemaVersionV1; got != want {
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
			json: `{
				"schema_version": "review_probe_plan.v1",
				"target_kind": "current_changes",
				"probes": [{
					"id": "probe-1",
					"purpose": "Run focused tests",
					"mode": "host_readonly",
					"commands": [{"command": "go", "args": ["test", "./internal/review"]}]
				}],
				"unexpected": true
			}`,
		},
		{
			name: "unknown nested probe field",
			json: `{
				"schema_version": "review_probe_plan.v1",
				"target_kind": "current_changes",
				"probes": [{
					"id": "probe-1",
					"purpose": "Run focused tests",
					"mode": "host_readonly",
					"commands": [{"command": "go", "args": ["test", "./internal/review"]}],
					"unexpected": true
				}]
			}`,
		},
		{
			name: "unknown nested command field",
			json: `{
				"schema_version": "review_probe_plan.v1",
				"target_kind": "current_changes",
				"probes": [{
					"id": "probe-1",
					"purpose": "Run focused tests",
					"mode": "host_readonly",
					"commands": [{"command": "go", "unexpected": true}]
				}]
			}`,
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
				plan.SchemaVersion = "review_probe_plan.v2"
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
	plan := ReviewProbePlan{
		SchemaVersion: ReviewProbePlanSchemaVersionV1,
		TargetKind:    TargetCurrentChanges,
		NoProbeReason: "No additional probe is needed.",
	}

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
		SchemaVersion: ReviewProbePlanSchemaVersionV1,
		TargetKind:    TargetCurrentChanges,
		Summary:       "Probe current changes.",
		Probes: []ReviewPlannedProbe{
			{
				ID:             "probe-1",
				Purpose:        "Run focused review tests.",
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

func mustMarshalReviewProbePlanForTest(t *testing.T, plan ReviewProbePlan) []byte {
	t.Helper()

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v, want nil", err)
	}
	return data
}
