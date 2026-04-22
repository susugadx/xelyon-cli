package repomap

import "fmt"

type projectMapPatternScanner struct {
	pm *ProjectMap
}

func newProjectMapPatternScanner(pm *ProjectMap) *projectMapPatternScanner {
	return &projectMapPatternScanner{pm: pm}
}

func (s *projectMapPatternScanner) scan(def languagePattern, targets []string, seen map[string]map[int]struct{}) (map[string][]Symbol, error) {
	if s.pm == nil {
		return nil, fmt.Errorf("project map is nil")
	}

	output, err := s.pm.runSymbolScan(def, targets)
	if err != nil {
		return nil, err
	}
	return parseSymbolScanOutput(output, seen), nil
}
