package repomap

import (
	"errors"
	"testing"
)

func TestSymbolScanStrategy_ScanWithPatternFallback_UsesPatternScanner(t *testing.T) {
	scanner := &stubPatternScanner{
		results: []map[string][]Symbol{
			{"pkg/fallback.go": {{Name: "Build", Line: 3}}},
			{"pkg/task.py": {{Name: "run_task", Line: 5}}},
		},
	}
	strategy := newSymbolScanStrategyWithPatternScanner([]languagePattern{
		{
			Extensions: []string{".go"},
			Patterns:   []string{`^func `},
		},
		{
			Extensions: []string{".py"},
			Patterns:   []string{`^def `},
		},
	}, scanner)

	results := map[string][]Symbol{}
	targetsByExt := map[string][]string{
		".go": {"pkg/fallback.go"},
		".py": {"pkg/task.py"},
	}
	if err := strategy.scanWithPatternFallback(results, targetsByExt); err != nil {
		t.Fatalf("scanWithPatternFallback() error = %v", err)
	}

	if len(scanner.calls) != 2 {
		t.Fatalf("pattern scanner calls = %d, want 2", len(scanner.calls))
	}
	if len(scanner.calls[0].targets) != 1 || scanner.calls[0].targets[0] != "pkg/fallback.go" {
		t.Fatalf("first call targets = %#v, want [pkg/fallback.go]", scanner.calls[0].targets)
	}
	if len(scanner.calls[1].targets) != 1 || scanner.calls[1].targets[0] != "pkg/task.py" {
		t.Fatalf("second call targets = %#v, want [pkg/task.py]", scanner.calls[1].targets)
	}
	if len(results["pkg/fallback.go"]) != 1 || results["pkg/fallback.go"][0].Name != "Build" {
		t.Fatalf("go fallback symbols = %#v, want Build", results["pkg/fallback.go"])
	}
	if len(results["pkg/task.py"]) != 1 || results["pkg/task.py"][0].Name != "run_task" {
		t.Fatalf("python fallback symbols = %#v, want run_task", results["pkg/task.py"])
	}
}

func TestSymbolScanStrategy_ScanWithPatternFallback_PropagatesScannerError(t *testing.T) {
	wantErr := errors.New("scan failed")
	strategy := newSymbolScanStrategyWithPatternScanner([]languagePattern{
		{
			Extensions: []string{".go"},
			Patterns:   []string{`^func `},
		},
	}, &stubPatternScanner{err: wantErr})

	err := strategy.scanWithPatternFallback(map[string][]Symbol{}, map[string][]string{
		".go": {"pkg/fallback.go"},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("scanWithPatternFallback() error = %v, want %v", err, wantErr)
	}
}
