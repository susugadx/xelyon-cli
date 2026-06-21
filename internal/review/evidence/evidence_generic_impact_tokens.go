package evidence

import (
	"regexp"
	"strings"
)

var (
	reviewGenericImpactCommandTokenRE    = regexp.MustCompile(`/[A-Za-z][A-Za-z0-9_-]*`)
	reviewGenericImpactFlagTokenRE       = regexp.MustCompile(`--[A-Za-z][A-Za-z0-9-]*`)
	reviewGenericImpactKeyTokenRE        = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_-]*)\s*[:=]`)
	reviewGenericImpactIdentifierTokenRE = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)
)

type reviewGenericImpactToken struct {
	value    string
	fromDiff bool
	fromStem bool
}

type reviewGenericImpactTokenExtraction struct {
	commands    []string
	flags       []string
	keys        []string
	identifiers []string
}

type reviewGenericImpactTokenExtractor struct {
	extraction reviewGenericImpactTokenExtraction
	seen       map[string]struct{}
}

func extractReviewGenericImpactDiffTokens(diffs []ReviewDiffEvidence) reviewGenericImpactTokenExtraction {
	extractor := newReviewGenericImpactTokenExtractor()
	for _, diff := range diffs {
		for _, line := range strings.Split(diff.Diff, "\n") {
			line, ok := reviewGenericImpactChangedDiffContentLine(line)
			if !ok {
				continue
			}
			extractor.addLine(line)
		}
	}
	return extractor.extraction
}

func extractReviewGenericImpactUntrackedTokens(files []ReviewUntrackedFile) reviewGenericImpactTokenExtraction {
	extractor := newReviewGenericImpactTokenExtractor()
	for _, file := range files {
		if file.Symlink || file.Binary || strings.TrimSpace(file.Snapshot) == "" ||
			isReviewGenericImpactExcludedPath(file.Path) || !isReviewGenericImpactSearchableTextPath(file.Path) {
			continue
		}
		for _, line := range strings.Split(file.Snapshot, "\n") {
			extractor.addLine(line)
		}
	}
	return extractor.extraction
}

func newReviewGenericImpactTokenExtractor() reviewGenericImpactTokenExtractor {
	return reviewGenericImpactTokenExtractor{
		seen: make(map[string]struct{}),
	}
}

func (e *reviewGenericImpactTokenExtractor) addLine(line string) {
	for _, token := range reviewGenericImpactCommandTokenRE.FindAllString(line, -1) {
		e.addToken(&e.extraction.commands, token)
	}
	for _, token := range reviewGenericImpactFlagTokenRE.FindAllString(line, -1) {
		e.addToken(&e.extraction.flags, token)
	}
	for _, match := range reviewGenericImpactKeyTokenRE.FindAllStringSubmatch(line, -1) {
		if len(match) > 1 {
			e.addToken(&e.extraction.keys, match[1])
		}
	}
	for _, token := range reviewGenericImpactIdentifierTokenRE.FindAllString(line, -1) {
		if isReviewGenericImpactIdentifierLikeToken(token) {
			e.addToken(&e.extraction.identifiers, token)
		}
	}
}

func reviewGenericImpactUntrackedSnapshotsHaveTruncation(files []ReviewUntrackedFile) bool {
	for _, file := range files {
		if file.Truncated {
			return true
		}
	}
	return false
}

func reviewGenericImpactChangedDiffContentLine(line string) (string, bool) {
	if len(line) == 0 {
		return "", false
	}
	switch line[0] {
	case '+':
		if isReviewGenericImpactDiffFileHeader(line, '+') {
			return "", false
		}
	case '-':
		if isReviewGenericImpactDiffFileHeader(line, '-') {
			return "", false
		}
	default:
		return "", false
	}
	return line[1:], true
}

func isReviewGenericImpactDiffFileHeader(line string, marker byte) bool {
	if len(line) < 4 || line[0] != marker || line[1] != marker || line[2] != marker {
		return false
	}
	switch line[3] {
	case ' ', '\t':
		return true
	default:
		return false
	}
}

func reviewGenericImpactLineContainsToken(line, token string) bool {
	if strings.HasPrefix(token, "--") || strings.HasPrefix(token, "/") || strings.Contains(token, "-") {
		return strings.Contains(line, token)
	}
	start := 0
	for {
		index := strings.Index(line[start:], token)
		if index < 0 {
			return false
		}
		index += start
		beforeOK := index == 0 || !isReviewGenericImpactIdentifierByte(line[index-1])
		afterIndex := index + len(token)
		afterOK := afterIndex >= len(line) || !isReviewGenericImpactIdentifierByte(line[afterIndex])
		if beforeOK && afterOK {
			return true
		}
		start = afterIndex
	}
}

func isReviewGenericImpactUsefulToken(token string) bool {
	token = strings.TrimSpace(token)
	if len(token) < 3 || len(token) > 80 {
		return false
	}
	if _, ok := reviewGenericImpactStopWords[strings.ToLower(token)]; ok {
		return false
	}
	return true
}

func isReviewGenericImpactIdentifierLikeToken(token string) bool {
	if !isReviewGenericImpactUsefulToken(token) {
		return false
	}
	if strings.Contains(token, "_") {
		return true
	}
	hasLower := false
	hasUpper := false
	for i := 0; i < len(token); i++ {
		if token[i] >= 'a' && token[i] <= 'z' {
			hasLower = true
		}
		if token[i] >= 'A' && token[i] <= 'Z' {
			hasUpper = true
		}
	}
	return hasLower && hasUpper
}

func isReviewGenericImpactIdentifierByte(ch byte) bool {
	return ch == '_' ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9')
}
