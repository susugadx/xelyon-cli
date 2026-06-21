package evidence

import (
	"path/filepath"
	"strings"
)

type reviewGenericImpactCandidateBuilder struct {
	bundle ReviewEvidenceBundle
	limits ReviewEvidenceLimits

	repoRoot string

	repoPaths    []string
	changedPaths map[string]struct{}
	changedDirs  []string
	changedStems []string

	tokens []reviewGenericImpactToken

	candidates []ReviewGenericImpactCandidate
	seen       map[string]struct{}
	roleCounts map[string]int
	tokenHits  map[string]int
	truncated  bool

	searchFileCount int
	totalSearchRead int64
}

// BuildReviewGenericImpactCandidates は言語非依存 heuristic で review 用 impact 候補を作る。
// これは import/caller graph ではなく、Pass1 の impact surface 検討に使う bounded lead である。
func BuildReviewGenericImpactCandidates(bundle ReviewEvidenceBundle) ReviewGenericImpactCandidates {
	builder := newReviewGenericImpactCandidateBuilder(bundle)
	builder.collectRepoPaths()
	builder.collectPathCandidates()
	builder.collectSearchCandidates()
	return builder.build()
}

func newReviewGenericImpactCandidateBuilder(bundle ReviewEvidenceBundle) *reviewGenericImpactCandidateBuilder {
	limits := normalizeReviewEvidenceLimits(bundle.Limits)
	repoRoot := strings.TrimSpace(bundle.RepoRoot)
	if repoRoot != "" {
		repoRoot = filepath.Clean(repoRoot)
	}
	changedPaths := reviewGenericImpactChangedPathSet(bundle)
	changedDirs := reviewGenericImpactChangedDirs(changedPaths)
	changedStems := reviewGenericImpactChangedStems(changedPaths)
	builder := &reviewGenericImpactCandidateBuilder{
		bundle:       bundle,
		limits:       limits,
		repoRoot:     repoRoot,
		changedPaths: changedPaths,
		changedDirs:  changedDirs,
		changedStems: changedStems,
		seen:         make(map[string]struct{}),
		roleCounts:   make(map[string]int),
		tokenHits:    make(map[string]int),
		truncated:    bundle.GenericImpactCandidateListTruncated,
	}
	builder.collectTokens()
	return builder
}

func (b *reviewGenericImpactCandidateBuilder) collectTokens() {
	diffTokens := extractReviewGenericImpactDiffTokens(b.bundle.Diffs)
	for _, token := range diffTokens.commands {
		b.addToken(token, true, false)
	}
	for _, token := range diffTokens.flags {
		b.addToken(token, true, false)
	}
	for _, token := range diffTokens.keys {
		b.addToken(token, true, false)
	}
	untrackedTokens := extractReviewGenericImpactUntrackedTokens(b.bundle.UntrackedFiles)
	if b.bundle.UntrackedSnapshotsTruncated || reviewGenericImpactUntrackedSnapshotsHaveTruncation(b.bundle.UntrackedFiles) {
		b.truncated = true
	}
	for _, token := range untrackedTokens.commands {
		b.addToken(token, true, false)
	}
	for _, token := range untrackedTokens.flags {
		b.addToken(token, true, false)
	}
	for _, token := range untrackedTokens.keys {
		b.addToken(token, true, false)
	}
	for _, stem := range b.changedStems {
		b.addToken(stem, false, true)
	}
	for _, token := range diffTokens.identifiers {
		b.addToken(token, true, false)
	}
	for _, token := range untrackedTokens.identifiers {
		b.addToken(token, true, false)
	}
}

func (b *reviewGenericImpactCandidateBuilder) addToken(token string, fromDiff, fromStem bool) {
	token = strings.TrimSpace(token)
	if !isReviewGenericImpactUsefulToken(token) {
		return
	}
	for i := range b.tokens {
		if b.tokens[i].value == token {
			b.tokens[i].fromDiff = b.tokens[i].fromDiff || fromDiff
			b.tokens[i].fromStem = b.tokens[i].fromStem || fromStem
			return
		}
	}
	if len(b.tokens) >= reviewGenericImpactMaxTokens {
		b.truncated = true
		return
	}
	b.tokens = append(b.tokens, reviewGenericImpactToken{
		value:    token,
		fromDiff: fromDiff,
		fromStem: fromStem,
	})
}

func (b *reviewGenericImpactCandidateBuilder) addCandidate(candidate ReviewGenericImpactCandidate) bool {
	if strings.TrimSpace(candidate.Path) == "" || strings.TrimSpace(candidate.Role) == "" {
		return false
	}
	key := candidate.Role + "\x00" + candidate.Path + "\x00" + candidate.Token + "\x00" + strconvReviewGenericImpactLine(candidate.Line)
	if _, ok := b.seen[key]; ok {
		return true
	}
	if len(b.candidates) >= reviewGenericImpactMaxCandidatesTotal {
		b.truncated = true
		return false
	}
	if b.roleCounts[candidate.Role] >= reviewGenericImpactMaxCandidatesPerRole {
		b.truncated = true
		return false
	}
	if candidate.Line > 0 && candidate.Token != "" {
		hitKey := candidate.Role + "\x00" + candidate.Token
		if b.tokenHits[hitKey] >= reviewGenericImpactMaxHitsPerToken {
			b.truncated = true
			return false
		}
		b.tokenHits[hitKey]++
	}
	b.seen[key] = struct{}{}
	b.roleCounts[candidate.Role]++
	b.candidates = append(b.candidates, candidate)
	return true
}

func (b *reviewGenericImpactCandidateBuilder) build() ReviewGenericImpactCandidates {
	tokens := make([]string, 0, len(b.tokens))
	for _, token := range b.tokens {
		tokens = append(tokens, token.value)
	}
	if tokens == nil {
		tokens = []string{}
	}
	if b.candidates == nil {
		b.candidates = []ReviewGenericImpactCandidate{}
	}
	return ReviewGenericImpactCandidates{
		Tokens:     tokens,
		Candidates: b.candidates,
		Truncated:  b.truncated,
	}
}

func (e *reviewGenericImpactTokenExtractor) addToken(values *[]string, token string) {
	token = strings.TrimSpace(token)
	if !isReviewGenericImpactUsefulToken(token) {
		return
	}
	if _, ok := e.seen[token]; ok {
		return
	}
	e.seen[token] = struct{}{}
	*values = append(*values, token)
}
