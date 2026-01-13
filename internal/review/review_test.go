package review

import (
	"context"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestNewOrchestrator(t *testing.T) {
	o := NewOrchestrator()

	if o.Scanner == nil {
		t.Error("Scanner should not be nil")
	}
	if o.Analyzer == nil {
		t.Error("Analyzer should not be nil")
	}
	if o.Fixer == nil {
		t.Error("Fixer should not be nil")
	}
	if o.Reporter == nil {
		t.Error("Reporter should not be nil")
	}
	if o.Now == nil {
		t.Error("Now should not be nil")
	}
}

func TestOrchestrator_Run_EmptyChanges(t *testing.T) {
	o := NewOrchestrator()
	ctx := context.Background()

	report, _, err := o.Run(ctx, []tools.FileChange{}, Options{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(report.Targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(report.Targets))
	}
	if len(report.Issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(report.Issues))
	}
}

func TestOrchestrator_Run_NilComponents(t *testing.T) {
	o := &Orchestrator{
		Scanner:  nil,
		Analyzer: nil,
		Reporter: nil,
	}
	ctx := context.Background()

	_, _, err := o.Run(ctx, []tools.FileChange{}, Options{})
	if err == nil {
		t.Error("Run() should return error when components are nil")
	}
}

func TestOrchestrator_Run_ModeChanges(t *testing.T) {
	o := NewOrchestrator()
	o.Now = func() time.Time {
		return time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	}
	ctx := context.Background()

	changes := []tools.FileChange{
		{FilePath: "test.go", Tool: "write_file"},
	}

	opt := Options{
		All:      false,
		Provider: "test",
		Model:    "test-model",
	}

	report, _, err := o.Run(ctx, changes, opt)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if report.Mode != "changes" {
		t.Errorf("Mode = %q, want %q", report.Mode, "changes")
	}
	if report.Provider != "test" {
		t.Errorf("Provider = %q, want %q", report.Provider, "test")
	}
	if report.Model != "test-model" {
		t.Errorf("Model = %q, want %q", report.Model, "test-model")
	}
}

func TestOrchestrator_Run_WithFix(t *testing.T) {
	o := NewOrchestrator()
	ctx := context.Background()

	// Create a file with TODO to trigger an issue
	changes := []tools.FileChange{
		{FilePath: "todo.go", Tool: "write_file"},
	}

	opt := Options{
		All: false,
		Fix: true,
	}

	report, _, err := o.Run(ctx, changes, opt)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Fixes should be generated (may be empty if no matching issues)
	if report.Fixes == nil {
		t.Error("Fixes should not be nil when --fix is enabled")
	}
}

func TestOrchestrator_Run_FocusSecurity(t *testing.T) {
	o := NewOrchestrator()
	ctx := context.Background()

	changes := []tools.FileChange{
		{FilePath: "test.go", Tool: "write_file"},
	}

	opt := Options{
		All:   false,
		Focus: ReviewFocus{Security: true},
	}

	report, _, err := o.Run(ctx, changes, opt)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should run without error
	_ = report
}

func TestOrchestrator_Run_FocusTest(t *testing.T) {
	o := NewOrchestrator()
	ctx := context.Background()

	changes := []tools.FileChange{
		{FilePath: "test.go", Tool: "write_file"},
	}

	opt := Options{
		All:   false,
		Focus: ReviewFocus{Test: true},
	}

	report, _, err := o.Run(ctx, changes, opt)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should run without error
	_ = report
}

func TestOptions_Defaults(t *testing.T) {
	opt := Options{}

	if opt.All != false {
		t.Error("All should default to false")
	}
	if opt.Fix != false {
		t.Error("Fix should default to false")
	}
	if opt.Focus.Security != false {
		t.Error("Focus.Security should default to false")
	}
	if opt.Focus.Test != false {
		t.Error("Focus.Test should default to false")
	}
}

func TestReviewFocus(t *testing.T) {
	tests := []struct {
		name     string
		focus    ReviewFocus
		security bool
		test     bool
	}{
		{
			name:     "default",
			focus:    ReviewFocus{},
			security: false,
			test:     false,
		},
		{
			name:     "security only",
			focus:    ReviewFocus{Security: true},
			security: true,
			test:     false,
		},
		{
			name:     "test only",
			focus:    ReviewFocus{Test: true},
			security: false,
			test:     true,
		},
		{
			name:     "both",
			focus:    ReviewFocus{Security: true, Test: true},
			security: true,
			test:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.focus.Security != tt.security {
				t.Errorf("Security = %v, want %v", tt.focus.Security, tt.security)
			}
			if tt.focus.Test != tt.test {
				t.Errorf("Test = %v, want %v", tt.focus.Test, tt.test)
			}
		})
	}
}
