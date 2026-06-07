package rawoutputs

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	displayURLPattern          = regexp.MustCompile(`https?://[^\s<>"'\]\)}]+`)
	privateKeyBlockPattern     = regexp.MustCompile(`(?is)-----BEGIN [^-]*PRIVATE KEY-----.*?-----END [^-]*PRIVATE KEY-----`)
	secretAssignmentPattern    = regexp.MustCompile(`(?i)\b(access_token|refresh_token|id_token|session_token|auth_token|api_key|apikey|client_secret|private_key|password|passwd|secret|token|jwt|signature|sig)=([^\s&;]+)`)
	secretJSONFieldPattern     = regexp.MustCompile(`(?i)["']?(access_token|refresh_token|id_token|session_token|auth_token|api_key|apikey|client_secret|private_key|password|passwd|secret|token|jwt|signature|sig)["']?\s*:\s*["'][^"']+["']`)
	authHeaderDisplayPattern   = regexp.MustCompile("(?i)(\\bauthorization\\s*[:=]\\s*)(?:([A-Za-z][A-Za-z0-9._-]*)\\s+)?([^\\s'\";]+)")
	secretHeaderPattern        = regexp.MustCompile("(?i)\\b(x-api-key|api-key|apikey|access-token|refresh-token|id-token|session-token|auth-token|client-secret)\\s*[:=]\\s*([^\\s'\";]+)")
	cookieHeaderPattern        = regexp.MustCompile(`(?i)\b(set-cookie|cookie)\s*[:=]\s*[^\r\n]+`)
	secretQueryFallbackPattern = regexp.MustCompile(`(?i)([?&](?:access_token|refresh_token|id_token|session_token|auth_token|api_key|apikey|key|secret|password|passwd|token|client_secret|jwt|signature|sig)=)[^&#\s]+`)
)

func sanitizeSourceMetadata(source SourceMetadata) SourceMetadata {
	source.Provider = trimDisplay(source.Provider, maxPreviewRunes)
	source.Model = trimDisplay(source.Model, maxPreviewRunes)
	source.CommandHash = trimHashOrEmpty(source.CommandHash)
	source.CommandPreview = trimDisplay(source.CommandPreview, maxPreviewRunes)
	source.ToolName = trimDisplay(source.ToolName, maxPreviewRunes)
	source.ToolCallID = trimDisplay(source.ToolCallID, maxPreviewRunes)
	source.EventID = trimDisplay(source.EventID, maxPreviewRunes)
	return source
}

func sanitizeClassificationMetadata(classification ClassificationMetadata) ClassificationMetadata {
	classification.SemanticRole = trimDisplay(classification.SemanticRole, maxPreviewRunes)
	classification.Family = trimDisplay(classification.Family, maxPreviewRunes)
	classification.Subfamily = trimDisplay(classification.Subfamily, maxPreviewRunes)
	classification.Classifier = trimDisplay(classification.Classifier, maxPreviewRunes)
	return classification
}

func trimHashOrEmpty(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "sha256:") {
		hash, err := parseSHA256ID(value)
		if err == nil {
			return "sha256:" + hash
		}
	}
	return trimDisplay(value, maxPreviewRunes)
}

func trimDisplay(value string, limit int) string {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(value)
	value = redactDisplaySecrets(value)
	value = strings.Join(strings.Fields(value), " ")
	if limit <= 0 || len([]rune(value)) <= limit {
		return value
	}
	runes := []rune(value)
	if limit <= 3 {
		return string(runes[:limit])
	}
	return string(runes[:limit-3]) + "..."
}

// SanitizeDisplayPreview は provider-facing metadata に出せる短い表示文字列へ redaction / trimming する。
func SanitizeDisplayPreview(value string, limit int) string {
	return trimDisplay(value, limit)
}

// RedactDisplaySecrets は prompt / report / ledger に出る外部由来 text から secret-like 値を伏せる。
func RedactDisplaySecrets(value string) string {
	return redactDisplaySecrets(value)
}

// LooksSensitiveContent は raw artifact として永続化してはいけない secret-like body を検出する。
func LooksSensitiveContent(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if secretAssignmentPattern.MatchString(value) ||
		secretJSONFieldPattern.MatchString(value) ||
		authHeaderDisplayPattern.MatchString(value) ||
		secretHeaderPattern.MatchString(value) ||
		cookieHeaderPattern.MatchString(value) ||
		privateKeyBlockPattern.MatchString(value) ||
		secretQueryFallbackPattern.MatchString(value) {
		return true
	}
	return displayURLPattern.MatchString(value) && displayURLHasSensitiveParts(value)
}

func redactDisplaySecrets(value string) string {
	value = displayURLPattern.ReplaceAllStringFunc(value, redactDisplayURL)
	value = privateKeyBlockPattern.ReplaceAllString(value, "[redacted private key]")
	value = secretAssignmentPattern.ReplaceAllString(value, "$1=[redacted]")
	value = secretJSONFieldPattern.ReplaceAllString(value, "$1: [redacted]")
	value = authHeaderDisplayPattern.ReplaceAllStringFunc(value, redactAuthorizationHeader)
	value = secretHeaderPattern.ReplaceAllString(value, "$1: [redacted]")
	value = cookieHeaderPattern.ReplaceAllString(value, "$1: [redacted]")
	value = secretQueryFallbackPattern.ReplaceAllString(value, "${1}[redacted]")
	return value
}

func redactAuthorizationHeader(value string) string {
	match := authHeaderDisplayPattern.FindStringSubmatch(value)
	if len(match) < 4 {
		return "[redacted authorization]"
	}
	prefix := match[1]
	scheme := match[2]
	if scheme != "" && looksLikeAuthorizationScheme(scheme) {
		return prefix + scheme + " [redacted]"
	}
	return prefix + "[redacted]"
}

func looksLikeAuthorizationScheme(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r == '-' || r == '_' || r == '.' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
			continue
		}
		return false
	}
	return true
}

func displayURLHasSensitiveParts(value string) bool {
	found := false
	displayURLPattern.ReplaceAllStringFunc(value, func(raw string) string {
		if rawURLHasSensitiveParts(raw) {
			found = true
		}
		return raw
	})
	return found
}

func rawURLHasSensitiveParts(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return secretQueryFallbackPattern.MatchString(raw)
	}
	if parsed.User != nil {
		return true
	}
	if secretQueryFallbackPattern.MatchString("?" + parsed.RawQuery) {
		return true
	}
	if secretQueryFallbackPattern.MatchString("?" + parsed.Fragment) {
		return true
	}
	return false
}

func redactDisplayURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return secretQueryFallbackPattern.ReplaceAllString(raw, "${1}[redacted]")
	}
	parsed.User = nil
	if parsed.RawQuery != "" {
		parsed.RawQuery = "redacted"
	}
	if parsed.Fragment != "" {
		parsed.Fragment = "redacted"
	}
	return parsed.String()
}
