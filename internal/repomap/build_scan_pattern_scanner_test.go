package repomap

import "testing"

func TestProjectMapPatternScanner_Scan_NilProjectMap(t *testing.T) {
	scanner := newProjectMapPatternScanner(nil)
	_, err := scanner.scan(languagePattern{}, nil, map[string]map[int]struct{}{})
	if err == nil {
		t.Fatal("scanner.scan() error = nil, want non-nil")
	}
}
