package agent

import (
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func providerHistoryRawOutputRehydrateHintsFromRaw(raw []api.Message) []string {
	for i := len(raw) - 1; i >= 0; i-- {
		if raw[i].Role == "user" {
			return providerHistoryRawOutputRehydrateHints(raw[i].Content)
		}
	}
	return nil
}

func providerHistoryRawOutputRehydrateHints(value string) []string {
	words := providerHistoryRawOutputHintWords(value)
	seen := make(map[string]struct{}, len(words))
	hints := make([]string, 0, len(words))
	for _, word := range words {
		word = strings.ToLower(strings.Trim(word, ".,;:!?()[]{}<>\"'"))
		if word == "" || providerHistoryRawOutputHintStopWords[word] {
			continue
		}
		if len([]rune(word)) < 4 && !providerHistoryRawOutputContainsDigit(word) {
			continue
		}
		if _, ok := seen[word]; ok {
			continue
		}
		seen[word] = struct{}{}
		hints = append(hints, word)
	}
	sort.SliceStable(hints, func(i, j int) bool {
		leftDigit := providerHistoryRawOutputContainsDigit(hints[i])
		rightDigit := providerHistoryRawOutputContainsDigit(hints[j])
		if leftDigit != rightDigit {
			return leftDigit
		}
		return len(hints[i]) > len(hints[j])
	})
	if len(hints) > 32 {
		return hints[:32]
	}
	return hints
}

func providerHistoryRawOutputHintWords(value string) []string {
	words := make([]string, 0)
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		words = append(words, b.String())
		b.Reset()
	}
	for _, r := range value {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_' || r == '-' || r == '.' || r == '/' || r == ':':
			b.WriteRune(r)
		default:
			flush()
		}
	}
	flush()
	return words
}

var providerHistoryRawOutputHintStopWords = map[string]bool{
	"about":   true,
	"again":   true,
	"check":   true,
	"current": true,
	"history": true,
	"inspect": true,
	"latest":  true,
	"next":    true,
	"please":  true,
	"request": true,
	"result":  true,
	"show":    true,
	"that":    true,
	"this":    true,
	"with":    true,
}

func providerHistoryRawOutputContainsDigit(value string) bool {
	for _, r := range value {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}
