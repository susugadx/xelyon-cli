package repomap

import (
	"strings"
	"testing"
)

func TestParseSymbolScanOutput_DedupesAndRespectsSeen(t *testing.T) {
	seen := map[string]map[int]struct{}{
		"pkg/a.go": {
			5: {},
		},
	}
	output := strings.Join([]string{
		"pkg/a.go:3:func Build() error {",
		"pkg/a.go:3:func Build() error {",
		"pkg/a.go:5:type Builder struct {",
		"pkg/a.go:7:// func Fake() {}",
		"pkg/a.go:x:func Broken() {}",
		"invalid line",
	}, "\n")

	got := parseSymbolScanOutput(output, seen)
	symbols := got["pkg/a.go"]
	if len(symbols) != 1 {
		t.Fatalf("symbols length = %d, want 1", len(symbols))
	}
	if symbols[0].Line != 3 {
		t.Fatalf("line = %d, want 3", symbols[0].Line)
	}
	if symbols[0].Name != "Build" {
		t.Fatalf("name = %q, want Build", symbols[0].Name)
	}
	if _, ok := seen["pkg/a.go"][3]; !ok {
		t.Fatal("expected seen to include parsed line 3")
	}
	if _, ok := seen["pkg/a.go"][5]; !ok {
		t.Fatal("expected existing seen line 5 to be preserved")
	}
}

func TestParseSymbolScanOutput_NormalizesPathAndFiltersComment(t *testing.T) {
	output := strings.Join([]string{
		"./pkg/../pkg/tasks.py:2:class Runner:",
		"./pkg/../pkg/tasks.py:3:# class Ignored:",
		"./pkg/../pkg/tasks.py:4:async def run_task():",
	}, "\n")

	got := parseSymbolScanOutput(output, map[string]map[int]struct{}{})
	symbols := got["pkg/tasks.py"]
	if len(symbols) != 2 {
		t.Fatalf("symbols length = %d, want 2", len(symbols))
	}
	if symbols[0].Name != "Runner" {
		t.Fatalf("first symbol name = %q, want Runner", symbols[0].Name)
	}
	if symbols[1].Name != "run_task" {
		t.Fatalf("second symbol name = %q, want run_task", symbols[1].Name)
	}
}
