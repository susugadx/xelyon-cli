package probe

import (
	"strings"
	"testing"
)

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
