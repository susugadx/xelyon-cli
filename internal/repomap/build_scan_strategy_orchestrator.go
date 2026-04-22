package repomap

func (s *symbolScanStrategy) scan(states []fileState) (map[string][]Symbol, error) {
	results := s.scanWithAST(states)
	targetsByExt := s.collectPatternFallbackTargets(states, results)
	if len(targetsByExt) == 0 {
		sortSymbolsByLocation(results)
		return results, nil
	}

	if err := s.scanWithPatternFallback(results, targetsByExt); err != nil {
		return nil, err
	}

	sortSymbolsByLocation(results)
	return results, nil
}
