package externaldoc

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"mime"
	"regexp"
	"strings"
)

var (
	reviewExternalDocScriptRE = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>`)
	reviewExternalDocStyleRE  = regexp.MustCompile(`(?is)<style\b[^>]*>.*?</style>`)
	reviewExternalDocNavRE    = regexp.MustCompile(`(?is)<nav\b[^>]*>.*?</nav>`)
	reviewExternalDocTagRE    = regexp.MustCompile(`(?s)<[^>]+>`)
	reviewExternalDocSpaceRE  = regexp.MustCompile(`\s+`)
)

func reviewExternalDocAllowedContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	}
	mediaType = strings.ToLower(mediaType)
	return strings.HasPrefix(mediaType, "text/") ||
		mediaType == "application/xhtml+xml" ||
		mediaType == "application/xml" ||
		mediaType == "application/json"
}

func sanitizeReviewExternalDocText(body []byte, contentType string) string {
	text := string(body)
	mediaType, _, _ := mime.ParseMediaType(contentType)
	mediaType = strings.ToLower(mediaType)
	if strings.Contains(mediaType, "html") || strings.Contains(text, "<html") || strings.Contains(text, "<!doctype html") {
		text = reviewExternalDocScriptRE.ReplaceAllString(text, " ")
		text = reviewExternalDocStyleRE.ReplaceAllString(text, " ")
		text = reviewExternalDocNavRE.ReplaceAllString(text, " ")
		text = reviewExternalDocTagRE.ReplaceAllString(text, " ")
	}
	text = strings.ReplaceAll(text, "\x00", "")
	return strings.TrimSpace(reviewExternalDocSpaceRE.ReplaceAllString(text, " "))
}

func buildReviewExternalDocSnippets(docID, content string, sourceTruncated bool, focusTerms []FocusTerm) []SnippetEvidence {
	if snippets := buildReviewExternalDocFocusedSnippets(docID, content, sourceTruncated, focusTerms); len(snippets) > 0 {
		return snippets
	}
	return buildReviewExternalDocPrefixSnippets(docID, content, sourceTruncated)
}

func buildReviewExternalDocFocusedSnippets(docID, content string, sourceTruncated bool, focusTerms []FocusTerm) []SnippetEvidence {
	snippets := make([]SnippetEvidence, 0, reviewExternalDocMaxSnippets)
	seenRanges := make(map[reviewExternalDocSnippetRange]struct{})
	seenHashes := make(map[string]struct{})
	for _, focusTerm := range sanitizeFocusTerms(focusTerms) {
		matchStart := reviewExternalDocIndexFoldASCII(content, focusTerm.Term)
		if matchStart < 0 {
			continue
		}
		snippetRange := reviewExternalDocSnippetRangeAround(content, matchStart, matchStart+len(focusTerm.Term), reviewExternalDocMaxSnippetBytes)
		if snippetRange.empty() {
			continue
		}
		if _, exists := seenRanges[snippetRange]; exists {
			continue
		}
		chunk := content[snippetRange.start:snippetRange.end]
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		contentHash := reviewExternalDocContentHash(chunk)
		if _, exists := seenHashes[contentHash]; exists {
			continue
		}
		seenRanges[snippetRange] = struct{}{}
		seenHashes[contentHash] = struct{}{}
		snippets = append(snippets, SnippetEvidence{
			SnippetID:   fmt.Sprintf("%s-snippet-%d", docID, len(snippets)+1),
			Content:     chunk,
			ContentHash: contentHash,
			Truncated:   len(chunk) < len(content) || sourceTruncated,
			FocusTerm:   focusTerm.Term,
			FocusReason: focusTerm.Reason,
		})
		if len(snippets) >= reviewExternalDocMaxSnippets {
			break
		}
	}
	return snippets
}

type reviewExternalDocSnippetRange struct {
	start int
	end   int
}

func (r reviewExternalDocSnippetRange) empty() bool {
	return r.start >= r.end
}

func buildReviewExternalDocPrefixSnippets(docID, content string, sourceTruncated bool) []SnippetEvidence {
	var snippets []SnippetEvidence
	remaining := content
	for i := 1; i <= reviewExternalDocMaxSnippets && strings.TrimSpace(remaining) != ""; i++ {
		chunk := reviewExternalDocBoundedString(remaining, reviewExternalDocMaxSnippetBytes)
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			break
		}
		truncated := len(chunk) < len(remaining) || sourceTruncated
		snippets = append(snippets, SnippetEvidence{
			SnippetID:   fmt.Sprintf("%s-snippet-%d", docID, i),
			Content:     chunk,
			ContentHash: reviewExternalDocContentHash(chunk),
			Truncated:   truncated,
		})
		if len(chunk) >= len(remaining) {
			break
		}
		remaining = strings.TrimSpace(remaining[len(chunk):])
	}
	return snippets
}

func reviewExternalDocSnippetRangeAround(value string, matchStart, matchEnd, maxBytes int) reviewExternalDocSnippetRange {
	if len(value) <= maxBytes {
		return reviewExternalDocSnippetRange{start: 0, end: len(value)}
	}
	matchLen := matchEnd - matchStart
	if matchLen < 0 {
		matchLen = 0
	}
	before := (maxBytes - matchLen) / 2
	if before < 0 {
		before = 0
	}
	start := matchStart - before
	if start < 0 {
		start = 0
	}
	end := start + maxBytes
	if end > len(value) {
		end = len(value)
		start = end - maxBytes
		if start < 0 {
			start = 0
		}
	}
	start = reviewExternalDocForwardRuneBoundary(value, start)
	end = reviewExternalDocBackwardRuneBoundary(value, end)
	if start >= end {
		end = matchStart + len(reviewExternalDocBoundedString(value[matchStart:], maxBytes))
		return reviewExternalDocSnippetRange{start: matchStart, end: end}
	}
	return reviewExternalDocSnippetRange{start: start, end: end}
}

func reviewExternalDocForwardRuneBoundary(value string, index int) int {
	for index > 0 && index < len(value) && (value[index]&0xc0) == 0x80 {
		index++
	}
	return index
}

func reviewExternalDocBackwardRuneBoundary(value string, index int) int {
	for index > 0 && index < len(value) && (value[index]&0xc0) == 0x80 {
		index--
	}
	return index
}

func reviewExternalDocIndexFoldASCII(value, term string) int {
	if term == "" || len(term) > len(value) {
		return -1
	}
	for i := 0; i <= len(value)-len(term); i++ {
		if reviewExternalDocEqualFoldASCII(value[i:i+len(term)], term) {
			return i
		}
	}
	return -1
}

func reviewExternalDocEqualFoldASCII(value, term string) bool {
	if len(value) != len(term) {
		return false
	}
	for i := 0; i < len(term); i++ {
		if reviewExternalDocFoldASCIIByte(value[i]) != reviewExternalDocFoldASCIIByte(term[i]) {
			return false
		}
	}
	return true
}

func reviewExternalDocFoldASCIIByte(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

func reviewExternalDocBoundedString(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	cut := maxBytes
	for cut > 0 && (value[cut]&0xc0) == 0x80 {
		cut--
	}
	if cut <= 0 {
		return value[:maxBytes]
	}
	return value[:cut]
}

func reviewExternalDocContentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
}
