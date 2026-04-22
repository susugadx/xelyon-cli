package repomap

import "testing"

func TestSymbolScanStrategy_CollectPatternFallbackTargets(t *testing.T) {
	strategy := newSymbolScanStrategy(&ProjectMap{})
	states := []fileState{
		{path: "pkg/build.go", supportsSym: true},
		{path: "pkg/build_test.go", supportsSym: true},
		{path: "pkg/task.py", supportsSym: true},
		{path: "pkg/cached.py", supportsSym: true, cached: &CacheFile{}},
		{path: "README.md", supportsSym: false},
		{path: "scripts/run", supportsSym: true},
	}

	resolvedByAST := map[string][]Symbol{
		"pkg/build.go": {{Name: "Build"}},
	}
	got := strategy.collectPatternFallbackTargets(states, resolvedByAST)

	goTargets := got[".go"]
	if len(goTargets) != 1 || goTargets[0] != "pkg/build_test.go" {
		t.Fatalf("go targets = %#v, want [pkg/build_test.go]", goTargets)
	}
	pyTargets := got[".py"]
	if len(pyTargets) != 1 || pyTargets[0] != "pkg/task.py" {
		t.Fatalf("py targets = %#v, want [pkg/task.py]", pyTargets)
	}
	if _, ok := got[".md"]; ok {
		t.Fatalf("markdown target should not be collected: %#v", got[".md"])
	}
}
