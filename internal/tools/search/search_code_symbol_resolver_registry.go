package search

func resolverForLanguage(lang string) symbolResolver {
	switch lang {
	case "go":
		return goSymbolResolver{}
	case "js", "python", "rust", "java", "csharp", "php", "ruby", "swift", "scala", "elixir", "lua", "cpp":
		return genericLanguageResolver{lang: lang}
	case "":
		return genericLanguageResolver{lang: ""}
	default:
		return nil
	}
}
