package probe

import "strings"

type probeShortOptionToken struct {
	name             byte
	hasAttachedValue bool
	attachedValue    string
	consumesNext     bool
}

func parseProbeShortOptions(arg string, valueOptions map[byte]struct{}) ([]probeShortOptionToken, bool) {
	if arg == "-" || !strings.HasPrefix(arg, "-") || strings.HasPrefix(arg, "--") {
		return nil, false
	}

	body := strings.TrimPrefix(arg, "-")
	if body == "" {
		return nil, false
	}

	tokens := make([]probeShortOptionToken, 0, len(body))
	for idx := 0; idx < len(body); idx++ {
		token := probeShortOptionToken{name: body[idx]}
		if _, consumesValue := valueOptions[token.name]; consumesValue {
			if idx+1 < len(body) {
				token.hasAttachedValue = true
				token.attachedValue = body[idx+1:]
			} else {
				token.consumesNext = true
			}
			tokens = append(tokens, token)
			return tokens, true
		}
		tokens = append(tokens, token)
	}
	return tokens, true
}

func probeShortOptionsContain(tokens []probeShortOptionToken, name byte) bool {
	for _, token := range tokens {
		if token.name == name {
			return true
		}
	}
	return false
}

func probeShortOptionValue(tokens []probeShortOptionToken, name byte) (value string, attached bool, consumesNext bool) {
	for _, token := range tokens {
		if token.name == name {
			return token.attachedValue, token.hasAttachedValue, token.consumesNext
		}
	}
	return "", false, false
}
