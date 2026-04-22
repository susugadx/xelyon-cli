package repomap

import "sort"

type projectMapBudgetReducer struct {
	pm      *ProjectMap
	options []renderOption
	omitted int
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
	if result, ok := r.renderIfWithinBudget(); ok {
		return result
	}

	r.suppressTestFileSymbols()
	if result, ok := r.renderIfWithinBudget(); ok {
		return result
	}

	for _, idx := range r.orderedIncludedFileIndexes() {
		r.options[idx].include = false
		r.omitted++

		if result, ok := r.renderIfWithinBudget(); ok {
			return result
		}
	}

	return r.pm.render(r.options, r.omitted)
}

func (r *projectMapBudgetReducer) renderIfWithinBudget() (string, bool) {
	result := r.pm.render(r.options, r.omitted)
	return result, r.pm.fitsBudget(result)
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
