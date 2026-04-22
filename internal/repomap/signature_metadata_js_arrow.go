package repomap

import "strings"

func extractJSArrowFunctionMetadata(sig string) (string, string, bool) {
	trimmed := strings.TrimSpace(sig)
	for _, prefix := range []string{"export const ", "const ", "let "} {
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}

		rest := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		nameEnd := 0
		for i, r := range rest {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9' && i > 0) || r == '_' {
				nameEnd = i + 1
				continue
			}
			break
		}
		if nameEnd == 0 {
			return "", "", false
		}

		name := rest[:nameEnd]
		rest = strings.TrimSpace(rest[nameEnd:])
		if !strings.HasPrefix(rest, "=") {
			return "", "", false
		}

		rest = strings.TrimSpace(strings.TrimPrefix(rest, "="))
		if strings.HasPrefix(rest, "async ") {
			rest = strings.TrimSpace(strings.TrimPrefix(rest, "async "))
		}
		if !strings.HasPrefix(rest, "(") {
			return "", "", false
		}

		closeIdx := strings.Index(rest, ")")
		if closeIdx < 0 {
			return "", "", false
		}
		if !strings.Contains(rest[closeIdx+1:], "=>") {
			return "", "", false
		}

		return name, "function", true
	}

	return "", "", false
}
