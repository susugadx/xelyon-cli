package repomap

func (s *symbolScanStrategy) scanWithAST(states []fileState) map[string][]Symbol {
	results := make(map[string][]Symbol)

	for _, state := range states {
		if state.cached != nil || !state.supportsSym || s.astScanner == nil || !s.astScanner.supports(state.path) {
			continue
		}

		astSymbols, err := s.astScanner.scan(state.absPath)
		if err != nil {
			continue
		}

		results[state.path] = astSymbols
	}

	return results
}

func (s *symbolScanStrategy) collectPatternFallbackTargets(states []fileState, resolvedByAST map[string][]Symbol) map[string][]string {
	targetsByExt := make(map[string][]string)

	for _, state := range states {
		if state.cached != nil || !state.supportsSym {
			continue
		}
		if _, done := resolvedByAST[state.path]; done {
			continue
		}
		ext := extensionForPath(state.path)
		if ext == "" {
			continue
		}
		targetsByExt[ext] = append(targetsByExt[ext], state.path)
	}

	return targetsByExt
}
