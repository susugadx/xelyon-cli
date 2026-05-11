package review

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	pathpkg "path"
	"strings"
)

type reviewRelatedSearchCollectionResult struct {
	hits      []ReviewRelatedSearchHit
	truncated bool
}

type reviewRelatedSearchCollector struct {
	repoRoot string
	ctx      context.Context
	limits   ReviewEvidenceLimits

	searchFileCount int
	totalSearchRead int64
	truncated       bool

	changedPaths          map[string]struct{}
	relatedCandidatePaths []string
}

type reviewRelatedSearchTerm struct {
	term     string
	reason   string
	priority int
}

type reviewRelatedSearchTermSet struct {
	items     []reviewRelatedSearchTerm
	truncated bool
}

type reviewRelatedSearchScanResult struct {
	stop bool
}

const (
	reviewRelatedSearchPrioritySymbol = iota
	reviewRelatedSearchPriorityFileStem
	reviewRelatedSearchPriorityPackage
	reviewRelatedSearchPriorityCount
)

type reviewRelatedSearchHitBuckets struct {
	limit int
	hits  [reviewRelatedSearchPriorityCount][]ReviewRelatedSearchHit
}

func newReviewRelatedSearchCollector(ctx context.Context, repoRoot string, limits ReviewEvidenceLimits, changedPaths map[string]struct{}, relatedCandidatePaths []string) *reviewRelatedSearchCollector {
	if ctx == nil {
		ctx = context.Background()
	}
	return &reviewRelatedSearchCollector{
		repoRoot:              repoRoot,
		ctx:                   ctx,
		limits:                normalizeReviewEvidenceLimits(limits),
		changedPaths:          changedPaths,
		relatedCandidatePaths: relatedCandidatePaths,
	}
}

func (c *reviewRelatedSearchCollector) collect(changedFileContext []ReviewContextFileEvidence) (reviewRelatedSearchCollectionResult, error) {
	termSet := buildReviewRelatedSearchTerms(changedFileContext, c.limits)
	if termSet.truncated {
		c.markTruncated()
	}
	if len(termSet.items) == 0 {
		return reviewRelatedSearchCollectionResult{
			hits:      []ReviewRelatedSearchHit{},
			truncated: c.truncated,
		}, nil
	}

	buckets := newReviewRelatedSearchHitBuckets(c.limits.MaxRelatedSearchHits)
	for i, relPath := range c.relatedCandidatePaths {
		if buckets.highestPriorityFull() {
			if i < len(c.relatedCandidatePaths) {
				c.markTruncated()
			}
			break
		}
		if c.searchFileCount >= c.limits.MaxRelatedSearchFiles || c.totalSearchRead >= c.limits.MaxTotalRelatedSearchBytes {
			if i < len(c.relatedCandidatePaths) {
				c.markTruncated()
			}
			break
		}
		if err := c.contextErr(); err != nil {
			return reviewRelatedSearchCollectionResult{}, err
		}
		data, ok := c.readCandidate(relPath)
		if !ok {
			continue
		}
		scan := c.scanRelatedSearchContent(&buckets, relPath, string(data), termSet.items)
		if scan.stop {
			if i+1 < len(c.relatedCandidatePaths) {
				c.markTruncated()
			}
			break
		}
	}
	if buckets.outputTruncated() {
		c.markTruncated()
	}
	return reviewRelatedSearchCollectionResult{
		hits:      buckets.flatten(),
		truncated: c.truncated,
	}, nil
}

func (c *reviewRelatedSearchCollector) contextErr() error {
	if c.ctx == nil {
		return nil
	}
	return c.ctx.Err()
}

func (c *reviewRelatedSearchCollector) markTruncated() {
	c.truncated = true
}

func (c *reviewRelatedSearchCollector) readCandidate(relPath string) ([]byte, bool) {
	if !isReviewContextRelatedGoPath(relPath) {
		return nil, false
	}
	if _, changed := c.changedPaths[relPath]; changed {
		return nil, false
	}

	absPath, relPath, err := resolveReviewEvidenceRepoPathLexically(c.repoRoot, relPath)
	if err != nil {
		return nil, false
	}

	remainingTotal := c.limits.MaxTotalRelatedSearchBytes - c.totalSearchRead
	if remainingTotal <= 0 {
		return nil, false
	}
	maxBytes := minReviewEvidenceInt64(c.limits.MaxRelatedSearchFileBytes, remainingTotal)
	file := readReviewEvidenceRegularFile(reviewEvidenceRegularFileReadInput{
		repoRoot: c.repoRoot,
		absPath:  absPath,
		relPath:  relPath,
		maxBytes: maxBytes,
	})
	if file.regular {
		c.searchFileCount++
	}
	if file.status != reviewEvidenceRegularFileReadOK {
		return nil, false
	}
	c.totalSearchRead += file.readBytes
	if file.truncated {
		c.markTruncated()
	}
	if file.binary {
		return nil, false
	}
	return file.data, true
}

func (c *reviewRelatedSearchCollector) scanRelatedSearchContent(hits *reviewRelatedSearchHitBuckets, relPath, content string, terms []reviewRelatedSearchTerm) reviewRelatedSearchScanResult {
	lines := strings.SplitAfter(content, "\n")
	for i, line := range lines {
		if hits.highestPriorityFull() {
			if hasReviewRelatedSearchRemainingContent(lines[i:]) {
				c.markTruncated()
			}
			return reviewRelatedSearchScanResult{stop: true}
		}
		lineText := strings.TrimRight(line, "\r\n")
		if lineText == "" {
			continue
		}
		for _, term := range terms {
			if !reviewRelatedSearchTermMatchesLine(term, lineText) {
				continue
			}
			snippet := strings.TrimSpace(lineText)
			if snippet == "" {
				snippet = lineText
			}
			snippet, snippetTruncated := truncateReviewEvidenceStringPrefix(snippet, c.limits.MaxSearchSnippetBytes)
			if snippetTruncated {
				c.markTruncated()
			}
			if hits.appendHit(term.priority, ReviewRelatedSearchHit{
				Path:    relPath,
				Line:    i + 1,
				Snippet: snippet,
				Reason:  term.reason,
			}) {
				c.markTruncated()
			}
			break
		}
	}
	return reviewRelatedSearchScanResult{}
}

func newReviewRelatedSearchHitBuckets(limit int) reviewRelatedSearchHitBuckets {
	return reviewRelatedSearchHitBuckets{
		limit: limit,
	}
}

func (b *reviewRelatedSearchHitBuckets) appendHit(priority int, hit ReviewRelatedSearchHit) bool {
	if priority < 0 || priority >= reviewRelatedSearchPriorityCount {
		return false
	}
	if b.limit <= 0 {
		return true
	}
	if len(b.hits[priority]) >= b.limit {
		return true
	}
	b.hits[priority] = append(b.hits[priority], hit)
	return false
}

func (b reviewRelatedSearchHitBuckets) highestPriorityFull() bool {
	return b.limit > 0 && len(b.hits[reviewRelatedSearchPrioritySymbol]) >= b.limit
}

func (b reviewRelatedSearchHitBuckets) flatten() []ReviewRelatedSearchHit {
	result := make([]ReviewRelatedSearchHit, 0, b.limit)
	for priority := 0; priority < reviewRelatedSearchPriorityCount; priority++ {
		for _, hit := range b.hits[priority] {
			if len(result) >= b.limit {
				return result
			}
			result = append(result, hit)
		}
	}
	return result
}

func (b reviewRelatedSearchHitBuckets) outputTruncated() bool {
	total := 0
	for priority := 0; priority < reviewRelatedSearchPriorityCount; priority++ {
		total += len(b.hits[priority])
	}
	if b.limit <= 0 {
		return total > 0
	}
	return total > b.limit
}

func hasReviewRelatedSearchRemainingContent(lines []string) bool {
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			return true
		}
	}
	return false
}

func reviewRelatedSearchTermMatchesLine(term reviewRelatedSearchTerm, lineText string) bool {
	if isReviewRelatedPackageDeclarationLine(lineText) {
		return false
	}
	if !strings.Contains(lineText, term.term) {
		return false
	}
	return true
}

func isReviewRelatedPackageDeclarationLine(lineText string) bool {
	return strings.HasPrefix(strings.TrimSpace(lineText), "package ")
}

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
