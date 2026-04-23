package repomap

type languagePattern struct {
	Extensions []string
	Patterns   []string
}

var defaultPatterns = collectDefaultLanguagePatterns()

func collectDefaultLanguagePatterns() []languagePattern {
	return []languagePattern{
		defaultLanguagePatternGo(),
		defaultLanguagePatternJavaScript(),
		defaultLanguagePatternPython(),
		defaultLanguagePatternRust(),
		defaultLanguagePatternJVM(),
		defaultLanguagePatternRuby(),
		defaultLanguagePatternPHP(),
		defaultLanguagePatternC(),
		defaultLanguagePatternSwift(),
		defaultLanguagePatternScala(),
		defaultLanguagePatternShell(),
	}
}
