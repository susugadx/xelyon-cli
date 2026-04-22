package repomap

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
