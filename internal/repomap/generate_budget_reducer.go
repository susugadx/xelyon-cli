package repomap

import "sort"

type projectMapBudgetReducer struct {
	pm                    *ProjectMap
	options               []renderOption
	omitted               int
	testSymbolsSuppressed bool
	omitOrder             []int
	omitIndex             int
}

func newProjectMapBudgetReducer(pm *ProjectMap) *projectMapBudgetReducer {
	options := make([]renderOption, len(pm.Files))
	for i := range pm.Files {
		options[i] = renderOption{include: true, showSymbols: true}
	}
	return &projectMapBudgetReducer{
		pm:      pm,
		options: options,
	}
}

func (r *projectMapBudgetReducer) reduce() string {
	return runBudgetReduction(budgetReductionEngine{
		render: func() string {
			return r.pm.render(r.options, r.omitted)
		},
		fits: r.pm.fitsBudget,
		shrink: func() bool {
			return r.shrink()
		},
	})
}

func (r *projectMapBudgetReducer) shrink() bool {
	if !r.testSymbolsSuppressed {
		r.suppressTestFileSymbols()
		r.testSymbolsSuppressed = true
		return true
	}

	if len(r.omitOrder) == 0 {
		r.omitOrder = r.orderedIncludedFileIndexes()
	}
	if r.omitIndex >= len(r.omitOrder) {
		return false
	}

	idx := r.omitOrder[r.omitIndex]
	r.omitIndex++
	r.options[idx].include = false
	r.omitted++
	return true
}

func (r *projectMapBudgetReducer) suppressTestFileSymbols() {
	for i, file := range r.pm.Files {
		if file == nil || !isTestFile(file.Path) || len(file.Symbols) == 0 {
			continue
		}
		r.options[i].showSymbols = false
	}
}

func (r *projectMapBudgetReducer) orderedIncludedFileIndexes() []int {
	var order []int
	for i, file := range r.pm.Files {
		if !r.options[i].include {
			continue
		}
		order = append(order, i)
		if file == nil {
			continue
		}
	}

	sort.Slice(order, func(i, j int) bool {
		left := r.pm.Files[order[i]]
		right := r.pm.Files[order[j]]
		leftDepth := directoryDepth(left.Path)
		rightDepth := directoryDepth(right.Path)
		if leftDepth != rightDepth {
			return leftDepth > rightDepth
		}
		return left.Path > right.Path
	})
	return order
}
