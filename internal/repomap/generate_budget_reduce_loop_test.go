package repomap

import "testing"

func TestRunBudgetReduction_ShrinksUntilBudgetFit(t *testing.T) {
	phase := 0
	got := runBudgetReduction(budgetReductionEngine{
		render: func() string {
			if phase == 0 {
				return "over-budget"
			}
			return "within-budget"
		},
		fits: func(text string) bool {
			return text == "within-budget"
		},
		shrink: func() bool {
			phase++
			return true
		},
	})

	if got != "within-budget" {
		t.Fatalf("result = %q, want within-budget", got)
	}
}

func TestRunBudgetReduction_ReturnsLastRenderedWhenExhaustedWithoutFallback(t *testing.T) {
	got := runBudgetReduction(budgetReductionEngine{
		render: func() string {
			return "over-budget-final"
		},
		fits: func(string) bool {
			return false
		},
		shrink: func() bool {
			return false
		},
	})

	if got != "over-budget-final" {
		t.Fatalf("result = %q, want over-budget-final", got)
	}
}

func TestRunBudgetReduction_UsesFallbackWhenExhausted(t *testing.T) {
	got := runBudgetReduction(budgetReductionEngine{
		render: func() string {
			return "over-budget-final"
		},
		fits: func(string) bool {
			return false
		},
		shrink: func() bool {
			return false
		},
		onExhausted: func(rendered string) string {
			if rendered != "over-budget-final" {
				t.Fatalf("rendered = %q, want over-budget-final", rendered)
			}
			return "fallback"
		},
	})

	if got != "fallback" {
		t.Fatalf("result = %q, want fallback", got)
	}
}
