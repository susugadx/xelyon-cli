package repomap

import "regexp"

type languagePatternEngine struct {
	patternsByExtension map[string][]*regexp.Regexp
}

var defaultPatternEngine = newLanguagePatternEngine(defaultPatterns)

func newLanguagePatternEngine(patternDefinitions []languagePattern) *languagePatternEngine {
	return &languagePatternEngine{
		patternsByExtension: compilePatternDefinitions(patternDefinitions),
	}
}

func compilePatternDefinitions(patternDefinitions []languagePattern) map[string][]*regexp.Regexp {
	compiled := make(map[string][]*regexp.Regexp)
	for _, def := range patternDefinitions {
		var regexps []*regexp.Regexp
		for _, pattern := range def.Patterns {
			regexps = append(regexps, regexp.MustCompile(pattern))
		}
		for _, ext := range def.Extensions {
			compiled[ext] = regexps
		}
	}
	return compiled
}

func (e *languagePatternEngine) supports(path string) bool {
	if e == nil {
		return false
	}
	_, ok := e.patternsByExtension[extensionForPath(path)]
	return ok
}

func (e *languagePatternEngine) matches(path, line string) bool {
	if e == nil {
		return false
	}
	regexps := e.patternsByExtension[extensionForPath(path)]
	if len(regexps) == 0 {
		return false
	}
	if isCommentLikeLine(path, line) {
		return false
	}
	for _, re := range regexps {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}
