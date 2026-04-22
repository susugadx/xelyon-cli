package navigation

import "strings"

func extractMethodReceiver(signature string) string {
	signature = strings.TrimSpace(signature)
	if !strings.HasPrefix(signature, "func") {
		return ""
	}

	rest := strings.TrimSpace(strings.TrimPrefix(signature, "func"))
	if !strings.HasPrefix(rest, "(") {
		return ""
	}

	closeIdx := strings.Index(rest, ")")
	if closeIdx <= 1 {
		return ""
	}

	receiverSpec := strings.TrimSpace(rest[1:closeIdx])
	if receiverSpec == "" {
		return ""
	}

	fields := strings.Fields(receiverSpec)
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}

func canonicalReceiver(receiver string) string {
	receiver = strings.TrimSpace(receiver)
	for strings.HasPrefix(receiver, "(") && strings.HasSuffix(receiver, ")") && len(receiver) > 1 {
		receiver = strings.TrimSpace(receiver[1 : len(receiver)-1])
	}
	if idx := strings.LastIndexAny(receiver, " \t"); idx >= 0 {
		receiver = strings.TrimSpace(receiver[idx+1:])
	}
	receiver = strings.TrimSpace(strings.TrimPrefix(receiver, "*"))
	return stripTypeArguments(receiver)
}

func stripTypeArguments(receiver string) string {
	if receiver == "" {
		return ""
	}

	var b strings.Builder
	depth := 0
	for _, r := range receiver {
		switch r {
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}
	return strings.TrimSpace(b.String())
}
