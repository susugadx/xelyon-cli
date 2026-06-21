package directquery

import "github.com/susugadx/xelyon-cli/internal/filequery"

type directQuerySyntaxKind = filequery.SyntaxKind

const (
	directQuerySyntaxNone                   = filequery.SyntaxNone
	directQuerySyntaxExplicitPath           = filequery.SyntaxExplicitPath
	directQuerySyntaxPathCandidate          = filequery.SyntaxPathCandidate
	directQuerySyntaxBareExtFileCandidate   = filequery.SyntaxBareExtFileCandidate
	directQuerySyntaxBareNamedFileCandidate = filequery.SyntaxBareNamedFileCandidate
)

type directQueryInput = filequery.Input
type directQueryEntryInput = filequery.Entry

func parseDirectQueryInput(query string) (directQueryInput, bool) {
	return filequery.ParseInput(query)
}

func parseDirectQueryEntryInput(entry string) (directQueryEntryInput, bool) {
	return filequery.ParseEntry(entry)
}

func looksLikeExplicitDirectQuery(query string) bool {
	return filequery.LooksLikeExplicitQuery(query)
}

func hasWindowsPathPrefix(path string) bool {
	return filequery.HasWindowsPathPrefix(path)
}

func inputHasOnlyExplicitPathSyntax(input directQueryInput) bool {
	return filequery.InputHasOnlyExplicitPathSyntax(input)
}

func inputHasOnlyPathCandidateSyntax(input directQueryInput) bool {
	return filequery.InputHasOnlyPathCandidateSyntax(input)
}

func inputHasOnlyStrongDirectIntent(input directQueryInput) bool {
	return filequery.InputHasOnlyStrongDirectIntent(input)
}

func inputHasStrictScopedDirectIntent(input directQueryInput) bool {
	return filequery.InputHasStrictScopedDirectIntent(input)
}

func inputContainsPathCandidateSyntax(input directQueryInput) bool {
	return filequery.InputContainsPathCandidateSyntax(input)
}

func inputHasOnlyNamedBareFileCandidates(input directQueryInput) bool {
	return filequery.InputHasOnlyNamedBareFileCandidates(input)
}

func inputHasOnlyCandidateDirectSyntax(input directQueryInput, allowNamedBareFiles bool) bool {
	return filequery.InputHasOnlyCandidateDirectSyntax(input, allowNamedBareFiles)
}

func inputHasOnlyDirectReadCandidates(input directQueryInput, allowNamedBareFiles bool) bool {
	return filequery.InputHasOnlyDirectReadCandidates(input, allowNamedBareFiles)
}
