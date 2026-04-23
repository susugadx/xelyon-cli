package repomap

type budgetReductionEngine struct {
	render      func() string
	fits        func(string) bool
	shrink      func() bool
	onExhausted func(string) string
}

func runBudgetReduction(engine budgetReductionEngine) string {
	for {
		rendered := engine.render()
		if rendered == "" || engine.fits(rendered) {
			return rendered
		}
		if !engine.shrink() {
			if engine.onExhausted != nil {
				return engine.onExhausted(rendered)
			}
			return rendered
		}
	}
}
