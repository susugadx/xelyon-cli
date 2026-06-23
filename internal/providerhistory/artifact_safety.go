package providerhistory

import "regexp"

const providerHistoryBareSecretKeyPattern = `password|passwd|secret|api[_-]?key|api\s+key|authorization|auth[_-]?token|access[_-]?token|refresh[_-]?token|id[_-]?token|session[_-]?token|client[_-]?secret|private[_-]?key|jwt`

var providerHistoryBareSecretPattern = regexp.MustCompile(`(?i)["']?\b(` + providerHistoryBareSecretKeyPattern + `)\b["']?\s*[:=]\s*["']?(bearer\s+)?[^\s"'\]\}),;]+`)

func providerHistoryLooksBareSecret(content string) bool {
	return providerHistoryBareSecretPattern.MatchString(content)
}
