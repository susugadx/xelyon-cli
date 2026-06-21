package search

type symbolBundleSectionInput struct {
	Kind       string
	Title      string
	Items      []genericSymbolRef
	TotalItems []genericSymbolRef
	Limit      int
	IsTest     bool
}

func buildGenericSymbolBundle(lang, query string, def genericSymbolDef, body []string, inputs []symbolBundleSectionInput) *SymbolBundle {
	displayName := def.Name
	if displayName == "" {
		displayName = query
	}

	bundle := &SymbolBundle{
		Identity: SymbolBundleIdentity{
			Language:    lang,
			Query:       query,
			Canonical:   canonicalSymbolBundleKey(lang, def.File, def.Line, displayName),
			DisplayName: displayName,
			Kind:        def.Kind,
			File:        def.File,
			Line:        def.Line,
			EndLine:     def.Line,
		},
		Definition: SymbolBundleDefinition{
			File:      def.File,
			Line:      def.Line,
			EndLine:   def.Line,
			Signature: def.Signature,
			Body:      append([]string(nil), body...),
		},
		Debug: SymbolBundleDebug{Source: "generic-resolver"},
	}

	for _, input := range inputs {
		section := buildGenericBundleSection(def, input)
		if section != nil {
			bundle.Sections = append(bundle.Sections, *section)
		}
	}
	finalizeSymbolBundleDiagnostics(bundle)

	return bundle
}
