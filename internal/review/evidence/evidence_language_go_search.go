package evidence

import (
	"go/ast"
	"go/parser"
	"go/token"
	pathpkg "path"
	"strings"
)

func extractReviewGoRelatedSearchTerms(_ reviewEvidenceLanguageSpec, file ReviewContextFileEvidence, addTerm reviewRelatedSearchTermAdder) bool {
	parsed, _ := parser.ParseFile(token.NewFileSet(), file.Path, file.Content, parser.SkipObjectResolution)
	if parsed != nil {
		for _, decl := range parsed.Decls {
			for _, term := range reviewGoRelatedSearchTermsFromDecl(decl) {
				if addTerm(term, "symbol:"+term, reviewRelatedSearchPrioritySymbol) {
					return true
				}
			}
		}
	}

	stem := reviewGoRelatedSearchFileStem(file.Path)
	if addTerm(stem, "file_stem:"+stem, reviewRelatedSearchPriorityFileStem) {
		return true
	}
	if parsed != nil && parsed.Name != nil {
		if addTerm(parsed.Name.Name, "package:"+parsed.Name.Name, reviewRelatedSearchPriorityPackage) {
			return true
		}
	}
	return false
}

func reviewGoRelatedSearchFileStem(relPath string) string {
	base := pathpkg.Base(relPath)
	return strings.TrimSuffix(base, pathpkg.Ext(base))
}

func reviewGoRelatedSearchTermsFromDecl(decl ast.Decl) []string {
	switch typed := decl.(type) {
	case *ast.FuncDecl:
		return reviewGoRelatedSearchTermsFromFuncDecl(typed)
	case *ast.GenDecl:
		return reviewGoRelatedSearchTermsFromGenDecl(typed)
	default:
		return nil
	}
}

func reviewGoRelatedSearchTermsFromFuncDecl(decl *ast.FuncDecl) []string {
	if decl == nil {
		return nil
	}
	if isReviewGoRelatedSearchPackageInitFunc(decl) {
		return nil
	}
	term, ok := reviewGoRelatedSearchTermFromIdent(decl.Name)
	if !ok {
		return nil
	}
	return []string{term}
}

func isReviewGoRelatedSearchPackageInitFunc(decl *ast.FuncDecl) bool {
	return decl != nil && decl.Recv == nil && decl.Name != nil && decl.Name.Name == "init"
}

func reviewGoRelatedSearchTermsFromGenDecl(decl *ast.GenDecl) []string {
	if decl == nil {
		return nil
	}

	terms := make([]string, 0, len(decl.Specs))
	for _, spec := range decl.Specs {
		switch decl.Tok {
		case token.TYPE:
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if term, ok := reviewGoRelatedSearchTermFromIdent(typeSpec.Name); ok {
				terms = append(terms, term)
			}
		case token.CONST, token.VAR:
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			terms = append(terms, reviewGoRelatedSearchTermsFromValueSpec(valueSpec)...)
		}
	}
	return terms
}

func reviewGoRelatedSearchTermsFromValueSpec(spec *ast.ValueSpec) []string {
	if spec == nil {
		return nil
	}

	terms := make([]string, 0, len(spec.Names))
	for _, name := range spec.Names {
		if term, ok := reviewGoRelatedSearchTermFromIdent(name); ok {
			terms = append(terms, term)
		}
	}
	return terms
}

func reviewGoRelatedSearchTermFromIdent(ident *ast.Ident) (string, bool) {
	if ident == nil || !isReviewGoRelatedSearchNamedIdentifier(ident.Name) {
		return "", false
	}
	return ident.Name, true
}

func isReviewGoRelatedSearchNamedIdentifier(name string) bool {
	return name != "" && name != "_"
}
