package review

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestGroupByFile(t *testing.T) {
	tests := []struct {
		name      string
		proposals []FixProposal
		wantLen   int
	}{
		{
			name:      "empty proposals",
			proposals: nil,
			wantLen:   0,
		},
		{
			name: "single file",
			proposals: []FixProposal{
				{IssueID: "1", FilePath: "file1.go"},
				{IssueID: "2", FilePath: "file1.go"},
			},
			wantLen: 1,
		},
		{
			name: "multiple files",
			proposals: []FixProposal{
				{IssueID: "1", FilePath: "file1.go"},
				{IssueID: "2", FilePath: "file2.go"},
				{IssueID: "3", FilePath: "file1.go"},
			},
			wantLen: 2,
		},
		{
			name: "three files",
			proposals: []FixProposal{
				{IssueID: "1", FilePath: "a.go"},
				{IssueID: "2", FilePath: "b.go"},
				{IssueID: "3", FilePath: "c.go"},
			},
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := GroupByFile(tt.proposals)
			if len(groups) != tt.wantLen {
				t.Errorf("GroupByFile() len = %d, want %d", len(groups), tt.wantLen)
			}
		})
	}
}

func TestGroupByFile_Ordering(t *testing.T) {
	proposals := []FixProposal{
		{IssueID: "1", FilePath: "c.go"},
		{IssueID: "2", FilePath: "a.go"},
		{IssueID: "3", FilePath: "b.go"},
	}

	groups := GroupByFile(proposals)

	// Should be sorted alphabetically
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}
	if groups[0].FilePath != "a.go" {
		t.Errorf("first group should be a.go, got %s", groups[0].FilePath)
	}
	if groups[1].FilePath != "b.go" {
		t.Errorf("second group should be b.go, got %s", groups[1].FilePath)
	}
	if groups[2].FilePath != "c.go" {
		t.Errorf("third group should be c.go, got %s", groups[2].FilePath)
	}
}

func TestGroupIndependent(t *testing.T) {
	groups := []FixGroup{
		{FilePath: "a.go", Proposals: []FixProposal{{IssueID: "1"}}},
		{FilePath: "b.go", Proposals: []FixProposal{{IssueID: "2"}}},
	}

	independent := GroupIndependent(groups)

	if len(independent) != 1 {
		t.Errorf("expected 1 independent group, got %d", len(independent))
	}
	if len(independent[0].Groups) != 2 {
		t.Errorf("expected 2 file groups in independent group, got %d", len(independent[0].Groups))
	}
}

func TestGroupIndependent_Empty(t *testing.T) {
	independent := GroupIndependent(nil)
	if independent != nil {
		t.Errorf("expected nil for empty input, got %v", independent)
	}
}

func TestCanRunParallel(t *testing.T) {
	tests := []struct {
		name string
		a    FixProposal
		b    FixProposal
		want bool
	}{
		{
			name: "same file - not parallel",
			a:    FixProposal{FilePath: "file.go"},
			b:    FixProposal{FilePath: "file.go"},
			want: false,
		},
		{
			name: "different files - parallel",
			a:    FixProposal{FilePath: "file1.go"},
			b:    FixProposal{FilePath: "file2.go"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanRunParallel(tt.a, tt.b); got != tt.want {
				t.Errorf("CanRunParallel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewParallelExecutor(t *testing.T) {
	tests := []struct {
		name       string
		maxWorkers int
		wantMax    int
	}{
		{"default workers", 0, 4},
		{"negative workers", -1, 4},
		{"custom workers", 8, 8},
		{"single worker", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pe := NewParallelExecutor(tt.maxWorkers)
			if pe.MaxWorkers != tt.wantMax {
				t.Errorf("MaxWorkers = %d, want %d", pe.MaxWorkers, tt.wantMax)
			}
		})
	}
}

func TestParallelExecutor_Execute(t *testing.T) {
	proposals := []FixProposal{
		{IssueID: "1", FilePath: "file1.go"},
		{IssueID: "2", FilePath: "file2.go"},
		{IssueID: "3", FilePath: "file3.go"},
	}

	groups := GroupByFile(proposals)
	pe := NewParallelExecutor(2)

	var counter int32
	applyFn := func(ctx context.Context, p FixProposal) error {
		atomic.AddInt32(&counter, 1)
		time.Sleep(10 * time.Millisecond) // Simulate work
		return nil
	}

	ctx := context.Background()
	results := pe.Execute(ctx, groups, applyFn)

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}
	if successCount != 3 {
		t.Errorf("expected 3 successful results, got %d", successCount)
	}

	if counter != 3 {
		t.Errorf("expected counter to be 3, got %d", counter)
	}
}

func TestParallelExecutor_Execute_WithErrors(t *testing.T) {
	proposals := []FixProposal{
		{IssueID: "1", FilePath: "file1.go"},
		{IssueID: "2", FilePath: "file2.go"},
	}

	groups := GroupByFile(proposals)
	pe := NewParallelExecutor(2)

	applyFn := func(ctx context.Context, p FixProposal) error {
		if p.FilePath == "file1.go" {
			return errors.New("simulated error")
		}
		return nil
	}

	ctx := context.Background()
	results := pe.Execute(ctx, groups, applyFn)

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	failCount := 0
	for _, r := range results {
		if !r.Success {
			failCount++
		}
	}
	if failCount != 1 {
		t.Errorf("expected 1 failed result, got %d", failCount)
	}
}

func TestParallelExecutor_Execute_ContextCancellation(t *testing.T) {
	proposals := []FixProposal{
		{IssueID: "1", FilePath: "file1.go"},
		{IssueID: "2", FilePath: "file2.go"},
	}

	groups := GroupByFile(proposals)
	pe := NewParallelExecutor(1)

	ctx, cancel := context.WithCancel(context.Background())

	applyFn := func(ctx context.Context, p FixProposal) error {
		if p.FilePath == "file1.go" {
			// Cancel after first file
			cancel()
		}
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	results := pe.Execute(ctx, groups, applyFn)

	// At least one result should exist
	if len(results) == 0 {
		t.Error("expected at least one result")
	}
}

func TestParallelExecutor_Summary(t *testing.T) {
	pe := NewParallelExecutor(4)
	pe.Results = []ParallelFixResult{
		{Success: true},
		{Success: true},
		{Success: false},
	}

	total, success, failed := pe.Summary()

	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if success != 2 {
		t.Errorf("success = %d, want 2", success)
	}
	if failed != 1 {
		t.Errorf("failed = %d, want 1", failed)
	}
}

func TestParallelExecutor_TotalDuration(t *testing.T) {
	pe := NewParallelExecutor(4)
	pe.Results = []ParallelFixResult{
		{Duration: 100 * time.Millisecond},
		{Duration: 200 * time.Millisecond},
		{Duration: 50 * time.Millisecond},
	}

	total := pe.TotalDuration()
	expected := 350 * time.Millisecond

	if total != expected {
		t.Errorf("TotalDuration() = %v, want %v", total, expected)
	}
}

func TestParallelExecutor_SameFileSequential(t *testing.T) {
	// Multiple fixes for the same file should be applied sequentially
	proposals := []FixProposal{
		{IssueID: "1", FilePath: "file.go"},
		{IssueID: "2", FilePath: "file.go"},
		{IssueID: "3", FilePath: "file.go"},
	}

	groups := GroupByFile(proposals)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	pe := NewParallelExecutor(4)

	var order []string
	applyFn := func(ctx context.Context, p FixProposal) error {
		order = append(order, p.IssueID)
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	ctx := context.Background()
	pe.Execute(ctx, groups, applyFn)

	// Order should be preserved for same file
	if len(order) != 3 {
		t.Errorf("expected 3 executions, got %d", len(order))
	}
	// First proposal should be executed first
	if order[0] != "1" {
		t.Errorf("expected first execution to be ID '1', got '%s'", order[0])
	}
}

func TestParallelExecutor_EmptyGroups(t *testing.T) {
	pe := NewParallelExecutor(4)
	ctx := context.Background()

	applyFn := func(ctx context.Context, p FixProposal) error {
		return nil
	}

	results := pe.Execute(ctx, nil, applyFn)
	if results != nil {
		t.Errorf("expected nil results for empty groups, got %v", results)
	}
}
