package navigation

import "testing"

func TestApplyInspectBudget_RelimitsCollectedEvidence(t *testing.T) {
	result := InspectResult{
		Body: []string{"1: func Run() {", "2: step()", "3: }"},
		Callers: []Reference{
			{File: "caller1.go", Line: 10},
			{File: "caller2.go", Line: 20},
			{File: "caller3.go", Line: 30},
		},
		TotalCallers: 5,
		MoreCallers:  true,
		Refs: []Reference{
			{File: "ref1.go", Line: 11},
			{File: "ref2.go", Line: 21},
		},
		TotalRefs: 2,
		Tests: []TestRef{
			{File: "run_test.go", Line: 12, Name: "TestRun"},
			{File: "run_test.go", Line: 24, Name: "TestRunOther"},
		},
		TotalTests: 4,
		MoreTests:  true,
		Implementations: []ImplementationRef{
			{File: "impl.go", Line: 8, Name: "runner"},
		},
	}

	got := ApplyInspectBudget(result, Budget{
		BodyLines:   2,
		CallerLimit: 1,
		RefLimit:    1,
		TestLimit:   1,
	})

	if len(got.Body) != 2 {
		t.Fatalf("body len = %d, want 2", len(got.Body))
	}
	if len(got.Callers) != 1 || got.TotalCallers != 5 || !got.MoreCallers {
		t.Fatalf("callers = len %d total %d more %v, want len 1 total 5 more true", len(got.Callers), got.TotalCallers, got.MoreCallers)
	}
	if len(got.Refs) != 1 || got.TotalRefs != 2 || !got.MoreRefs {
		t.Fatalf("refs = len %d total %d more %v, want len 1 total 2 more true", len(got.Refs), got.TotalRefs, got.MoreRefs)
	}
	if len(got.Tests) != 1 || got.TotalTests != 4 || !got.MoreTests {
		t.Fatalf("tests = len %d total %d more %v, want len 1 total 4 more true", len(got.Tests), got.TotalTests, got.MoreTests)
	}
	if len(got.Implementations) != 1 {
		t.Fatalf("implementations len = %d, want unchanged 1", len(got.Implementations))
	}
	if len(result.Callers) != 3 || len(result.Body) != 3 {
		t.Fatalf("ApplyInspectBudget mutated original result")
	}
}

func TestApplyInspectBudget_UsesCollectedLengthWhenTotalIsUnset(t *testing.T) {
	got := ApplyInspectBudget(InspectResult{
		Callers: []Reference{
			{File: "caller1.go", Line: 10},
			{File: "caller2.go", Line: 20},
		},
	}, Budget{BodyLines: 1, CallerLimit: 1})

	if got.TotalCallers != 2 || !got.MoreCallers {
		t.Fatalf("callers total/more = %d/%v, want 2/true", got.TotalCallers, got.MoreCallers)
	}
}
