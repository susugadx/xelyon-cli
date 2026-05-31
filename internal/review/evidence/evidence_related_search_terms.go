package evidence

import (
	"go/ast"
	"go/parser"
	"go/token"
	pathpkg "path"
	"strings"
)

type reviewRelatedSearchTerm struct {
	term     string
	reason   string
	priority int
}

type reviewRelatedSearchTermSet struct {
	items     []reviewRelatedSearchTerm
	truncated bool
}

const (
	reviewRelatedSearchPrioritySymbol = iota
	reviewRelatedSearchPriorityFileStem
	reviewRelatedSearchPriorityPackage
	reviewRelatedSearchPriorityCount
)

func buildReviewRelatedSearchTerms(changedFileContext []ReviewContextFileEvidence, limits ReviewEvidenceLimits) reviewRelatedSearchTermSet {
	limits = normalizeReviewEvidenceLimits(limits)
	terms := reviewRelatedSearchTermSet{
		items: make([]reviewRelatedSearchTerm, 0, limits.MaxRelatedSearchTerms),
	}
	seen := make(map[string]struct{})

	addTerm := func(term, reason string, priority int) bool {
		term = strings.TrimSpace(term)
		if term == "" {
			return false
		}
		if _, ok := seen[term]; ok {
			return false
		}
		seen[term] = struct{}{}
		if len(terms.items) >= limits.MaxRelatedSearchTerms {
			terms.truncated = true
			return true
		}
		terms.items = append(terms.items, reviewRelatedSearchTerm{term: term, reason: reason, priority: priority})
		return false
	}

	for _, file := range changedFileContext {
		if file.Skipped || pathpkg.Ext(file.Path) != ".go" {
			continue
		}
		parsed, _ := parser.ParseFile(token.NewFileSet(), file.Path, file.Content, parser.SkipObjectResolution)
		if parsed != nil {
			for _, decl := range parsed.Decls {
				for _, term := range reviewRelatedSearchTermsFromDecl(decl) {
					if addTerm(term, "symbol:"+term, reviewRelatedSearchPrioritySymbol) {
						return terms
					}
				}
			}
		}
		stem := strings.TrimSuffix(pathpkg.Base(file.Path), pathpkg.Ext(file.Path))
		if addTerm(stem, "file_stem:"+stem, reviewRelatedSearchPriorityFileStem) {
			return terms
		}
		if parsed != nil && parsed.Name != nil {
			if addTerm(parsed.Name.Name, "package:"+parsed.Name.Name, reviewRelatedSearchPriorityPackage) {
				return terms
			}
		}
	}

	return terms
}

func reviewRelatedSearchTermsFromDecl(decl ast.Decl) []string {
	switch typed := decl.(type) {
	case *ast.FuncDecl:
		return reviewRelatedSearchTermsFromFuncDecl(typed)
	case *ast.GenDecl:
		return reviewRelatedSearchTermsFromGenDecl(typed)
	default:
		return nil
	}
}

func reviewRelatedSearchTermsFromFuncDecl(decl *ast.FuncDecl) []string {
	if decl == nil {
		return nil
	}
	if isReviewRelatedSearchPackageInitFunc(decl) {
		return nil
	}
	term, ok := reviewRelatedSearchTermFromIdent(decl.Name)
	if !ok {
		return nil
	}
	return []string{term}
}

func isReviewRelatedSearchPackageInitFunc(decl *ast.FuncDecl) bool {
	return decl != nil && decl.Recv == nil && decl.Name != nil && decl.Name.Name == "init"
}

func reviewRelatedSearchTermsFromGenDecl(decl *ast.GenDecl) []string {
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
			if term, ok := reviewRelatedSearchTermFromIdent(typeSpec.Name); ok {
				terms = append(terms, term)
			}
		case token.CONST, token.VAR:
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			terms = append(terms, reviewRelatedSearchTermsFromValueSpec(valueSpec)...)
		}
	}
	return terms
}

func reviewRelatedSearchTermsFromValueSpec(spec *ast.ValueSpec) []string {
	if spec == nil {
		return nil
	}

	terms := make([]string, 0, len(spec.Names))
	for _, name := range spec.Names {
		if term, ok := reviewRelatedSearchTermFromIdent(name); ok {
			terms = append(terms, term)
		}
	}
	return terms
}

func reviewRelatedSearchTermFromIdent(ident *ast.Ident) (string, bool) {
	if ident == nil || !isReviewRelatedSearchNamedIdentifier(ident.Name) {
		return "", false
	}
	return ident.Name, true
}

func isReviewRelatedSearchNamedIdentifier(name string) bool {
	return name != "" && name != "_"
}
