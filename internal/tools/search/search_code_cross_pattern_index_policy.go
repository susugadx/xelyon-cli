package search

type crossPatternIndexRenderPolicy struct {
	MinCategoryCount int
	MinUniqueFiles   int
}

var defaultCrossPatternIndexRenderPolicy = crossPatternIndexRenderPolicy{
	MinCategoryCount: 2,
	MinUniqueFiles:   3,
}

func shouldRenderCrossPatternIndexData(data crossPatternIndexData) bool {
	return shouldRenderCrossPatternIndex(data.order, data.sections, data.hasHotspot)
}

func (d crossPatternIndexData) shouldRender() bool {
	return shouldRenderCrossPatternIndexData(d)
}

func shouldRenderCrossPatternIndex(order []string, sections crossPatternIndexSections, hasHotspot bool) bool {
	return defaultCrossPatternIndexRenderPolicy.shouldRender(order, sections, hasHotspot)
}

func (sections crossPatternIndexSections) categoryCount() int {
	categoryCount := 0
	if len(sections.implKeys) > 0 {
		categoryCount++
	}
	if len(sections.testKeys) > 0 {
		categoryCount++
	}
	if len(sections.configKeys) > 0 {
		categoryCount++
	}
	return categoryCount
}

func (policy crossPatternIndexRenderPolicy) shouldRender(order []string, sections crossPatternIndexSections, hasHotspot bool) bool {
	if hasHotspot {
		return true
	}
	if sections.categoryCount() >= policy.MinCategoryCount {
		return true
	}
	return len(order) >= policy.MinUniqueFiles
}
