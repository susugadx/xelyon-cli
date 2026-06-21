package probeplan

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

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
				plan.Probes[0].Mode = domain.ReviewProbeHostReadOnly
			},
			errContains: "probes[0].files",
		},
		{
			name: "host_readonly allows empty files",
			mutate: func(plan *ReviewProbePlan) {
				plan.Probes[0].Mode = domain.ReviewProbeHostReadOnly
				plan.Probes[0].Files = nil
			},
		},
		{
			name: "scratch_only allows generated files",
			mutate: func(plan *ReviewProbePlan) {
				plan.Probes[0].Mode = domain.ReviewProbeScratchOnly
			},
		},
		{
			name: "repo_sandbox allows generated files",
			mutate: func(plan *ReviewProbePlan) {
				plan.Probes[0].Mode = domain.ReviewProbeRepoSandbox
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
