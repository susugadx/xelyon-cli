package repomap

import "testing"

func TestProjectMapBudgetReducer_SuppressTestFileSymbols(t *testing.T) {
	pm := &ProjectMap{
		Files: []*FileEntry{
			{Path: "pkg/service.go", Symbols: []Symbol{{Name: "Build"}}},
			{Path: "pkg/service_test.go", Symbols: []Symbol{{Name: "TestBuild"}}},
			{Path: "pkg/no_symbol_test.go"},
		},
	}

	reducer := newProjectMapBudgetReducer(pm)
	reducer.suppressTestFileSymbols()

	if !reducer.options[0].showSymbols {
		t.Fatal("non-test file symbols should stay visible")
	}
	if reducer.options[1].showSymbols {
		t.Fatal("test file symbols should be hidden")
	}
	if !reducer.options[2].showSymbols {
		t.Fatal("test file without symbols should keep default visibility")
	}
}

func TestProjectMapBudgetReducer_OrderedIncludedFileIndexes(t *testing.T) {
	pm := &ProjectMap{
		Files: []*FileEntry{
			{Path: "root.go"},
			{Path: "internal/agent/run.go"},
			{Path: "pkg/service.go"},
			{Path: "internal/agent/helper.go"},
		},
	}

	reducer := newProjectMapBudgetReducer(pm)
	order := reducer.orderedIncludedFileIndexes()

	want := []int{1, 3, 2, 0}
	if len(order) != len(want) {
		t.Fatalf("order length = %d, want %d (%v)", len(order), len(want), order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order[%d] = %d, want %d (full=%v)", i, order[i], want[i], order)
		}
	}
}
