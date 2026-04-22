package repomap

import (
	"fmt"
)

type symbolASTScanner interface {
	supports(path string) bool
	scan(absPath string) ([]Symbol, error)
}

type symbolPatternScanner interface {
	scan(def languagePattern, targets []string, seen map[string]map[int]struct{}) (map[string][]Symbol, error)
}

type symbolScanStrategy struct {
	patternDefinitions []languagePattern
	astScanner         symbolASTScanner
	patternScanner     symbolPatternScanner
}

func newSymbolScanStrategy(pm *ProjectMap) *symbolScanStrategy {
	return newSymbolScanStrategyWithPatternDefinitions(pm, defaultPatterns)
}

func newSymbolScanStrategyWithPatternDefinitions(pm *ProjectMap, patternDefinitions []languagePattern) *symbolScanStrategy {
	return newSymbolScanStrategyWithScanners(
		patternDefinitions,
		newProjectMapASTScanner(),
		newProjectMapPatternScanner(pm),
	)
}

func newSymbolScanStrategyWithPatternScanner(patternDefinitions []languagePattern, patternScanner symbolPatternScanner) *symbolScanStrategy {
	return newSymbolScanStrategyWithScanners(patternDefinitions, newProjectMapASTScanner(), patternScanner)
}

func newSymbolScanStrategyWithScanners(
	patternDefinitions []languagePattern,
	astScanner symbolASTScanner,
	patternScanner symbolPatternScanner,
) *symbolScanStrategy {
	return &symbolScanStrategy{
		patternDefinitions: patternDefinitions,
		astScanner:         astScanner,
		patternScanner:     patternScanner,
	}
}

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

func (s *symbolScanStrategy) scanWithPatternFallback(results map[string][]Symbol, targetsByExt map[string][]string) error {
	if s.patternScanner == nil {
		return fmt.Errorf("pattern scanner is nil")
	}

	seen := make(map[string]map[int]struct{})

	for _, def := range s.patternDefinitions {
		var targets []string
		for _, ext := range def.Extensions {
			targets = append(targets, targetsByExt[ext]...)
		}
		if len(targets) == 0 {
			continue
		}

		symbols, err := s.patternScanner.scan(def, targets, seen)
		if err != nil {
			return err
		}
		for path, syms := range symbols {
			results[path] = append(results[path], syms...)
		}
	}

	return nil
}
