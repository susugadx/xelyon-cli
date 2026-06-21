package search

func cloneSymbolBundleImpact(impact *SymbolBundleImpact) *SymbolBundleImpact {
	if impact == nil {
		return nil
	}
	cloned := *impact
	if impact.RecommendedReads != nil {
		cloned.RecommendedReads = append([]SymbolBundleItem(nil), impact.RecommendedReads...)
	}
	return &cloned
}
