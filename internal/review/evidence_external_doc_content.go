package review

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

func buildReviewExternalDocSnippets(docID, content string, sourceTruncated bool, focusTerms []ReviewExternalDocFocusTerm) []ReviewExternalDocSnippetEvidence {
	if snippet, ok := buildReviewExternalDocFocusedSnippet(docID, content, sourceTruncated, focusTerms); ok {
		return []ReviewExternalDocSnippetEvidence{snippet}
	}
	return buildReviewExternalDocPrefixSnippets(docID, content, sourceTruncated)
}

func buildReviewExternalDocFocusedSnippet(docID, content string, sourceTruncated bool, focusTerms []ReviewExternalDocFocusTerm) (ReviewExternalDocSnippetEvidence, bool) {
	for _, focusTerm := range sanitizeReviewExternalDocFocusTerms(focusTerms) {
		matchStart := reviewExternalDocIndexFoldASCII(content, focusTerm.Term)
		if matchStart < 0 {
			continue
		}
		chunk := reviewExternalDocSnippetAround(content, matchStart, matchStart+len(focusTerm.Term), reviewExternalDocMaxSnippetBytes)
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		return ReviewExternalDocSnippetEvidence{
			SnippetID:   fmt.Sprintf("%s-snippet-1", docID),
			Content:     chunk,
			ContentHash: reviewExternalDocContentHash(chunk),
			Truncated:   len(chunk) < len(content) || sourceTruncated,
			FocusTerm:   focusTerm.Term,
			FocusReason: focusTerm.Reason,
		}, true
	}
	return ReviewExternalDocSnippetEvidence{}, false
}

func buildReviewExternalDocPrefixSnippets(docID, content string, sourceTruncated bool) []ReviewExternalDocSnippetEvidence {
	var snippets []ReviewExternalDocSnippetEvidence
	remaining := content
	for i := 1; i <= reviewExternalDocMaxSnippets && strings.TrimSpace(remaining) != ""; i++ {
		chunk := reviewExternalDocBoundedString(remaining, reviewExternalDocMaxSnippetBytes)
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			break
		}
		truncated := len(chunk) < len(remaining) || sourceTruncated
		snippets = append(snippets, ReviewExternalDocSnippetEvidence{
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

func reviewExternalDocSnippetAround(value string, matchStart, matchEnd, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
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
		return reviewExternalDocBoundedString(value[matchStart:], maxBytes)
	}
	return value[start:end]
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
