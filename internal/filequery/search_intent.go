package filequery

import (
	"path/filepath"
	"strings"
)

// LooksLikeNaturalLanguageSearchIntent は query が自然文の検索意図を含むか返す。
func LooksLikeNaturalLanguageSearchIntent(query string) bool {
	rawEntries := SplitEntries(query)
	if len(rawEntries) == 0 {
		return false
	}
	for _, rawEntry := range rawEntries {
		entry, ok := ParseEntry(rawEntry)
		if !ok {
			continue
		}
		if entryLooksLikeNaturalLanguageSearchIntent(entry) {
			return true
		}
	}
	return false
}

func entryLooksLikeNaturalLanguageSearchIntent(entry Entry) bool {
	rawEntry := strings.TrimSpace(entry.RawEntry)
	if rawEntry == "" || !strings.ContainsAny(rawEntry, " \t\r\n") {
		return false
	}
	if entryHasExplicitPathOverride(entry) {
		return false
	}

	return ContainsNaturalLanguageSearchIntentMarker(rawEntry) ||
		hasNaturalLanguageScopePhrase(rawEntry)
}

// ContainsNaturalLanguageSearchIntentMarker は query が自然文検索を示す語彙を含むか返す。
func ContainsNaturalLanguageSearchIntentMarker(query string) bool {
	tokens := searchIntentTokens(query)
	for _, token := range tokens {
		if isNaturalLanguageSearchIntentToken(cleanSearchIntentToken(token.text)) {
			return true
		}
	}
	for i := 0; i < len(tokens)-1; i++ {
		if isNaturalLanguageSearchIntentPhrase(
			cleanSearchIntentToken(tokens[i].text),
			cleanSearchIntentToken(tokens[i+1].text),
		) {
			return true
		}
	}
	return false
}

// LeadingSearchIntentPayloadStart は先頭の検索命令語の直後にある検索語の開始位置を返す。
func LeadingSearchIntentPayloadStart(query string) (int, bool) {
	tokens := searchIntentTokens(query)
	if len(tokens) == 0 {
		return 0, false
	}

	switch cleanSearchIntentToken(tokens[0].text) {
	case "look", "looking":
		if len(tokens) > 2 && cleanSearchIntentToken(tokens[1].text) == "for" {
			return tokens[2].start, true
		}
	case "search", "searching", "find", "finding":
		if len(tokens) > 2 && cleanSearchIntentToken(tokens[1].text) == "for" {
			return tokens[2].start, true
		}
		if len(tokens) > 1 {
			return tokens[1].start, true
		}
	}
	return 0, false
}

func isNaturalLanguageSearchIntentToken(token string) bool {
	switch token {
	case "or", "and", "search", "searching", "find", "finding":
		return true
	default:
		return false
	}
}

func isNaturalLanguageSearchIntentPhrase(first, second string) bool {
	return (first == "look" || first == "looking") && second == "for"
}

type searchIntentToken struct {
	text  string
	start int
}

func searchIntentTokens(query string) []searchIntentToken {
	tokens := make([]searchIntentToken, 0)
	start := -1
	for i := 0; i < len(query); i++ {
		if query[i] == ' ' || query[i] == '\t' || query[i] == '\r' || query[i] == '\n' {
			if start >= 0 {
				tokens = append(tokens, searchIntentToken{text: query[start:i], start: start})
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		tokens = append(tokens, searchIntentToken{text: query[start:], start: start})
	}
	return tokens
}

func cleanSearchIntentToken(token string) string {
	return strings.ToLower(strings.Trim(token, `"'()[]{}:;,.`))
}

func entryHasExplicitPathOverride(entry Entry) bool {
	return filepath.IsAbs(entry.CleanedPath) ||
		HasWindowsPathPrefix(entry.RawPath) ||
		entry.ExplicitRelative ||
		HasExplicitDirectoryMarker(entry.RawPath) ||
		entry.StartLine > 0 ||
		entry.EndLine > 0
}

func hasNaturalLanguageScopePhrase(rawEntry string) bool {
	fields := strings.Fields(rawEntry)
	for i := 0; i < len(fields)-1; i++ {
		word := strings.ToLower(strings.Trim(fields[i], `"'()[]{}:;,.`))
		if word != "in" && word != "under" {
			continue
		}
		if strings.Trim(fields[i+1], `"'()[]{}:;,.`) != "" {
			return true
		}
	}
	return false
}
