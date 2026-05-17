package goimpact

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/impactplan"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestClassifyRisk_Policy(t *testing.T) {
	tests := []struct {
		name   string
		result navigation.InspectResult
		want   string
	}{
		{
			name:   "nil symbol is low",
			result: navigation.InspectResult{},
			want:   impactplan.RiskLow,
		},
		{
			name: "exported symbol is high",
			result: navigation.InspectResult{
				Symbol: &navigation.SymbolCandidate{Name: "Run", Exported: true},
			},
			want: impactplan.RiskHigh,
		},
		{
			name: "interface symbol is high",
			result: navigation.InspectResult{
				Symbol: &navigation.SymbolCandidate{Name: "runner", Kind: "interface"},
			},
			want: impactplan.RiskHigh,
		},
		{
			name: "implementations make symbol high",
			result: navigation.InspectResult{
				Symbol:          &navigation.SymbolCandidate{Name: "runner"},
				Implementations: []navigation.ImplementationRef{{File: "runner.go", Line: 10}},
			},
			want: impactplan.RiskHigh,
		},
		{
			name: "references across directories are high",
			result: navigation.InspectResult{
				Symbol: &navigation.SymbolCandidate{Name: "runner"},
				Callers: []navigation.Reference{
					{File: "pkg/a.go"},
					{File: "other/b.go"},
				},
			},
			want: impactplan.RiskHigh,
		},
		{
			name: "references across files in one directory are medium",
			result: navigation.InspectResult{
				Symbol: &navigation.SymbolCandidate{Name: "runner"},
				Callers: []navigation.Reference{
					{File: "pkg/a.go"},
					{File: "pkg/b.go"},
				},
			},
			want: impactplan.RiskMedium,
		},
		{
			name: "shared package symbol is medium",
			result: navigation.InspectResult{
				Symbol: &navigation.SymbolCandidate{Name: "runner", PackageDir: "internal/run"},
			},
			want: impactplan.RiskMedium,
		},
		{
			name: "widening signal is medium",
			result: navigation.InspectResult{
				Symbol:       &navigation.SymbolCandidate{Name: "runner"},
				MoreCallers:  true,
				TotalCallers: 2,
			},
			want: impactplan.RiskMedium,
		},
		{
			name: "local unexported symbol without spread is low",
			result: navigation.InspectResult{
				Symbol: &navigation.SymbolCandidate{Name: "runner", PackageDir: "."},
				Callers: []navigation.Reference{
					{File: "runner.go"},
				},
			},
			want: impactplan.RiskLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyRisk(tt.result); got != tt.want {
				t.Fatalf("ClassifyRisk() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNeedsWidening(t *testing.T) {
	base := navigation.InspectResult{
		Callers:      []navigation.Reference{{File: "caller.go"}},
		Refs:         []navigation.Reference{{File: "ref.go"}},
		TotalCallers: 1,
		TotalRefs:    1,
	}
	tests := []struct {
		name   string
		mutate func(*navigation.InspectResult)
		want   bool
	}{
		{name: "no signal", want: false},
		{name: "more callers", mutate: func(r *navigation.InspectResult) { r.MoreCallers = true }, want: true},
		{name: "more refs", mutate: func(r *navigation.InspectResult) { r.MoreRefs = true }, want: true},
		{name: "upstream truncated", mutate: func(r *navigation.InspectResult) { r.UpstreamTruncated = true }, want: true},
		{name: "upstream incomplete", mutate: func(r *navigation.InspectResult) { r.UpstreamIncomplete = true }, want: true},
		{name: "caller count exceeds stored callers", mutate: func(r *navigation.InspectResult) { r.TotalCallers = 2 }, want: true},
		{name: "ref count exceeds stored refs", mutate: func(r *navigation.InspectResult) { r.TotalRefs = 2 }, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := base
			if tt.mutate != nil {
				tt.mutate(&result)
			}
			if got := NeedsWidening(result); got != tt.want {
				t.Fatalf("NeedsWidening() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReferenceSpread(t *testing.T) {
	fileCount, dirCount := ReferenceSpread(navigation.InspectResult{
		Callers: []navigation.Reference{
			{File: "pkg/a.go"},
			{File: "pkg/./a.go"},
			{File: "pkg/b.go"},
			{File: " "},
		},
		Refs: []navigation.Reference{
			{File: "other/c.go"},
			{File: ""},
		},
	})

	if fileCount != 3 || dirCount != 2 {
		t.Fatalf("ReferenceSpread() = (%d, %d), want (3, 2)", fileCount, dirCount)
	}
}

func TestIsSharedPackageSymbol(t *testing.T) {
	tests := []struct {
		packageDir string
		want       bool
	}{
		{packageDir: "", want: false},
		{packageDir: ".", want: false},
		{packageDir: "cmd", want: false},
		{packageDir: "cmd/server", want: false},
		{packageDir: "internal/run", want: true},
		{packageDir: "pkg/api", want: true},
		{packageDir: " pkg/api ", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.packageDir, func(t *testing.T) {
			got := IsSharedPackageSymbol(navigation.SymbolCandidate{PackageDir: tt.packageDir})
			if got != tt.want {
				t.Fatalf("IsSharedPackageSymbol(%q) = %v, want %v", tt.packageDir, got, tt.want)
			}
		})
	}
}
