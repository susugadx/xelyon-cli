package repomap

import "fmt"

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
