package repomap

func defaultLanguagePatternGo() languagePattern {
	return languagePattern{
		Extensions: []string{".go"},
		Patterns: []string{
			`^func `,
			`^type [A-Za-z0-9_]+ (struct|interface)`,
			`^var [A-Za-z0-9_]+`,
			`^const [A-Za-z0-9_]+`,
		},
	}
}

const jsDeclareLanguageKindPattern = `(async\s+function|function|abstract\s+class|class|const|interface|type|enum)`

func jsDeclareLanguagePatterns(prefix string) []string {
	return []string{
		`^` + prefix + `\s+` + jsDeclareLanguageKindPattern + `\s+`,
	}
}

func defaultLanguagePatternJavaScript() languagePattern {
	patterns := jsDeclareLanguagePatterns(`export\s+declare`)
	patterns = append(patterns, jsDeclareLanguagePatterns(`declare`)...)
	patterns = append(patterns, []string{
		`^export\s+(function|class|const|interface|type|enum|abstract\s+class)\s+`,
		`^export\s+default\s+(function|class)\s+`,
		`^(async\s+)?function\s+[A-Za-z0-9_]+`,
		`^class\s+[A-Za-z0-9_]+`,
		`^(const|let)\s+[A-Za-z0-9_]+\s*=`,
		`^interface\s+[A-Za-z0-9_]+`,
	}...)

	return languagePattern{
		Extensions: []string{".ts", ".tsx", ".js", ".jsx", ".mjs"},
		Patterns:   patterns,
	}
}

func defaultLanguagePatternPython() languagePattern {
	return languagePattern{
		Extensions: []string{".py"},
		Patterns: []string{
			`^(async )?def [A-Za-z0-9_]+`,
			`^class [A-Za-z0-9_]+`,
		},
	}
}

func defaultLanguagePatternRust() languagePattern {
	return languagePattern{
		Extensions: []string{".rs"},
		Patterns: []string{
			`^(pub )?(async )?fn [A-Za-z0-9_]+`,
			`^(pub )?struct [A-Za-z0-9_]+`,
			`^(pub )?enum [A-Za-z0-9_]+`,
			`^(pub )?trait [A-Za-z0-9_]+`,
			`^impl(<[^>]*>)? [A-Za-z0-9_]+`,
		},
	}
}

func defaultLanguagePatternJVM() languagePattern {
	return languagePattern{
		Extensions: []string{".java", ".kt", ".kts"},
		Patterns: []string{
			`^(public |private |protected )?(static )?(abstract )?(class|interface|enum|record) [A-Za-z0-9_]+`,
			`^(public |private |protected )?(static )?(abstract )?[A-Za-z0-9_<>,\[\]?]+\s+[A-Za-z0-9_]+\(`,
			`^(fun|suspend fun) [A-Za-z0-9_]+`,
			`^(data |sealed |value )?class [A-Za-z0-9_]+`,
			`^object [A-Za-z0-9_]+`,
		},
	}
}

func defaultLanguagePatternRuby() languagePattern {
	return languagePattern{
		Extensions: []string{".rb"},
		Patterns: []string{
			`^\s*def [A-Za-z0-9_]+`,
			`^\s*class [A-Za-z0-9_]+`,
			`^\s*module [A-Za-z0-9_]+`,
		},
	}
}

func defaultLanguagePatternPHP() languagePattern {
	return languagePattern{
		Extensions: []string{".php"},
		Patterns: []string{
			`^\s*(public |private |protected )?(static )?function [A-Za-z0-9_]+`,
			`^\s*(abstract )?(class|interface|trait|enum) [A-Za-z0-9_]+`,
		},
	}
}

func defaultLanguagePatternC() languagePattern {
	return languagePattern{
		Extensions: []string{".c", ".h", ".cpp", ".hpp", ".cc"},
		Patterns: []string{
			`^(typedef )?(struct|class|enum|union) [A-Za-z0-9_]+`,
			`^#define [A-Za-z0-9_]+`,
			`^namespace [A-Za-z0-9_]+`,
		},
	}
}

func defaultLanguagePatternSwift() languagePattern {
	return languagePattern{
		Extensions: []string{".swift"},
		Patterns: []string{
			`^\s*(public |private |internal |open )?(func|class|struct|enum|protocol|extension) [A-Za-z0-9_]+`,
		},
	}
}

func defaultLanguagePatternScala() languagePattern {
	return languagePattern{
		Extensions: []string{".scala"},
		Patterns: []string{
			`^\s*(def|class|object|trait|case class|case object|sealed trait) [A-Za-z0-9_]+`,
		},
	}
}

func defaultLanguagePatternShell() languagePattern {
	return languagePattern{
		Extensions: []string{".sh", ".bash", ".zsh"},
		Patterns: []string{
			`^([A-Za-z0-9_]+)\s*\(\)`,
			`^function [A-Za-z0-9_]+`,
		},
	}
}
