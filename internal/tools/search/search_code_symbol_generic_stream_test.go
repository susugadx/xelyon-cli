package search

import (
	"strings"
	"testing"
)

func TestCollectGenericSymbolMatches_RequestsCancelAtLimit(t *testing.T) {
	result := collectGenericSymbolMatches(strings.NewReader(genericSymbolRGOutput(
		"pkg/service.go:10:func Run(){}",
		"pkg/runner.go:12:Run()",
	)), SearchOptions{FileType: "go"}, 1)
	if !result.cancelRequested {
		t.Fatal("cancelRequested = false, want true when limit is reached")
	}
	if len(result.matches) != 1 {
		t.Fatalf("matches len = %d, want 1", len(result.matches))
	}
}

func TestCollectGenericSymbolMatches_RequestsCancelAtLargeLimit(t *testing.T) {
	const largeLimit = 500
	lines := make([]string, 0, largeLimit+1)
	for i := 0; i <= largeLimit; i++ {
		lines = append(lines, "src/file.go:10:Run()")
	}

	result := collectGenericSymbolMatches(strings.NewReader(genericSymbolRGOutput(lines...)), SearchOptions{FileType: "go"}, largeLimit)

	if !result.cancelRequested {
		t.Fatal("cancelRequested = false, want true at large limit")
	}
	if len(result.matches) != largeLimit {
		t.Fatalf("matches len = %d, want %d", len(result.matches), largeLimit)
	}
}

func TestCollectGenericSymbolMatches_DoesNotRequestCancelForUnlimitedSearch(t *testing.T) {
	result := collectGenericSymbolMatches(strings.NewReader(genericSymbolRGOutput(
		"pkg/service.go:10:func Run(){}",
		"pkg/runner.go:12:Run()",
	)), SearchOptions{FileType: "go"}, 0)
	if result.cancelRequested {
		t.Fatal("cancelRequested = true, want false for unlimited search")
	}
	if len(result.matches) != 2 {
		t.Fatalf("matches len = %d, want 2", len(result.matches))
	}
}

func TestCollectGenericSymbolMatches_NilReader(t *testing.T) {
	result := collectGenericSymbolMatches(nil, SearchOptions{FileType: "go"}, 1)
	if result.cancelRequested {
		t.Fatal("cancelRequested = true, want false for nil reader")
	}
	if len(result.matches) != 0 {
		t.Fatalf("matches len = %d, want 0", len(result.matches))
	}
}

func TestCollectGenericSymbolMatches_RequestsCancelOnScannerError(t *testing.T) {
	longLine := "pkg/service.go:10:" + strings.Repeat("x", 70*1024)

	result := collectGenericSymbolMatches(strings.NewReader(longLine), SearchOptions{FileType: "go"}, 0)
	if !result.cancelRequested {
		t.Fatal("cancelRequested = false, want true when scanner stops with an error")
	}
}

func genericSymbolRGOutput(lines ...string) string {
	return strings.Join(lines, "\n")
}
