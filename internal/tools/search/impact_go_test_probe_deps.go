package search

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type structuredGoImpactProbeDeps struct {
	paths map[string]struct{}
}

func newStructuredGoImpactProbeDeps() *structuredGoImpactProbeDeps {
	return &structuredGoImpactProbeDeps{paths: make(map[string]struct{})}
}

func (d *structuredGoImpactProbeDeps) add(path string) {
	if d == nil {
		return
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	d.paths[filepath.Clean(path)] = struct{}{}
}

func (d *structuredGoImpactProbeDeps) addDirGoFiles(dir string) {
	if d == nil {
		return
	}
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		d.add(filepath.Join(dir, entry.Name()))
	}
}

func (d *structuredGoImpactProbeDeps) list() []string {
	if d == nil || len(d.paths) == 0 {
		return nil
	}
	out := make([]string, 0, len(d.paths))
	for path := range d.paths {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}
