package repomap

type stubPatternScanner struct {
	calls   []patternScanCall
	results []map[string][]Symbol
	err     error
}

type stubASTScanner struct {
	supportedPaths map[string]bool
	results        map[string][]Symbol
	errByPath      map[string]error
	calls          []string
}

type patternScanCall struct {
	def     languagePattern
	targets []string
}

func (s *stubASTScanner) supports(path string) bool {
	if s.supportedPaths == nil {
		return false
	}
	return s.supportedPaths[path]
}

func (s *stubASTScanner) scan(absPath string) ([]Symbol, error) {
	s.calls = append(s.calls, absPath)
	if err := s.errByPath[absPath]; err != nil {
		return nil, err
	}
	return append([]Symbol(nil), s.results[absPath]...), nil
}

func (s *stubPatternScanner) scan(def languagePattern, targets []string, _ map[string]map[int]struct{}) (map[string][]Symbol, error) {
	s.calls = append(s.calls, patternScanCall{
		def:     def,
		targets: append([]string(nil), targets...),
	})
	if s.err != nil {
		return nil, s.err
	}
	if len(s.results) == 0 {
		return map[string][]Symbol{}, nil
	}
	next := s.results[0]
	s.results = s.results[1:]
	return next, nil
}
