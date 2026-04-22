package tools

import "strings"

var defaultJSONToolCallStartPatterns = []string{
	"{\"id\"",     // {"id" (Function Calling)
	"{ \"id\"",    // { "id" (Function Calling)
	"{\"tool\"",   // {"tool"
	"{ \"tool\"",  // { "tool"
	"{\"tool\":",  // {"tool":
	"{ \"tool\":", // { "tool":
}

type jsonToolCallStartFinder interface {
	Find(response string, searchFrom int) int
}

type patternJSONToolCallStartFinder struct {
	patterns []string
}

func newDefaultJSONToolCallStartFinder() jsonToolCallStartFinder {
	return newPatternJSONToolCallStartFinder(defaultJSONToolCallStartPatterns)
}

func newPatternJSONToolCallStartFinder(patterns []string) jsonToolCallStartFinder {
	copied := make([]string, len(patterns))
	copy(copied, patterns)
	return patternJSONToolCallStartFinder{patterns: copied}
}

func (f patternJSONToolCallStartFinder) Find(response string, searchFrom int) int {
	start := -1
	for _, pattern := range f.patterns {
		idx := strings.Index(response[searchFrom:], pattern)
		if idx == -1 {
			continue
		}
		absIdx := searchFrom + idx
		if start == -1 || absIdx < start {
			start = absIdx
		}
	}
	return start
}
