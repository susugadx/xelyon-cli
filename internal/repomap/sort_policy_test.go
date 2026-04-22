package repomap

import "testing"

func TestSortSymbolsByLocation(t *testing.T) {
	results := map[string][]Symbol{
		"pkg/service.go": {
			{Name: "Run", Signature: "func Run()", Line: 20},
			{Name: "Build", Signature: "func Build()", Line: 10},
			{Name: "BuildAlias", Signature: "func BuildAlias()", Line: 10, EndLine: 12},
		},
	}

	sortSymbolsByLocation(results)
	got := results["pkg/service.go"]
	if got[0].Name != "Build" || got[1].Name != "BuildAlias" || got[2].Name != "Run" {
		t.Fatalf("sortSymbolsByLocation() order = %+v", got)
	}
}

func TestDirectoryDepth(t *testing.T) {
	if got := directoryDepth("main.go"); got != 0 {
		t.Fatalf("directoryDepth(main.go) = %d, want 0", got)
	}
	if got := directoryDepth("internal/agent/runner.go"); got != 2 {
		t.Fatalf("directoryDepth(internal/agent/runner.go) = %d, want 2", got)
	}
}
