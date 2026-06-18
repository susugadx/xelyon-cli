package search

func resolverForLanguage(lang string) symbolResolver {
	factory, ok := symbolResolverRegistry[lang]
	if !ok {
		return nil
	}
	return factory()
}

type symbolResolverFactory func() symbolResolver

var symbolResolverRegistry = map[string]symbolResolverFactory{
	"go": func() symbolResolver {
		return goSymbolResolver{}
	},
	"js":     genericLanguageResolverFactory(genericLanguageResolverSpec{language: "js", resolve: resolveJSFamilySymbol}),
	"python": genericLanguageResolverFactory(genericLanguageResolverSpec{language: "python", resolve: resolvePythonSymbol}),
	"rust":   genericLanguageResolverFactory(genericLanguageResolverSpec{language: "rust", resolve: resolveRustSymbol}),
	"java":   genericLanguageResolverFactory(genericLanguageResolverSpec{language: "java", resolve: resolveJavaSymbol}),
	"csharp": genericLanguageResolverFactory(genericLanguageResolverSpec{language: "csharp", resolve: resolveCSharpSymbol}),
	"php":    genericLanguageResolverFactory(genericLanguageResolverSpec{language: "php", resolve: resolvePHPSymbol}),
	"ruby":   genericLanguageResolverFactory(genericLanguageResolverSpec{language: "ruby", resolve: resolveRubySymbol}),
	"swift":  genericLanguageResolverFactory(genericLanguageResolverSpec{language: "swift", resolve: resolveSwiftSymbol}),
	"scala":  genericLanguageResolverFactory(genericLanguageResolverSpec{language: "scala", resolve: resolveScalaSymbol}),
	"elixir": genericLanguageResolverFactory(genericLanguageResolverSpec{language: "elixir", resolve: resolveElixirSymbol}),
	"lua":    genericLanguageResolverFactory(genericLanguageResolverSpec{language: "lua", resolve: resolveLuaSymbol}),
	"cpp":    genericLanguageResolverFactory(genericLanguageResolverSpec{language: "cpp", resolve: resolveCppSymbol}),
	"":       genericLanguageResolverFactory(genericLanguageResolverSpec{language: "", resolve: resolveGenericSymbol}),
}

func genericLanguageResolverFactory(spec genericLanguageResolverSpec) symbolResolverFactory {
	return func() symbolResolver {
		return genericLanguageResolver{spec: spec}
	}
}
