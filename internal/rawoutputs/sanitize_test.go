package rawoutputs

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRedactDisplaySecretsRedactsURLFragmentsHeadersAndEncodedValues(t *testing.T) {
	input := strings.Join([]string{
		"Authorization: Basic dXNlcjpwYXNz",
		"https://example.test/items?token=secret#access_token=fragment-secret",
		"api_key=abc%2Fdef%3D",
		`{"client_secret":"json-secret"}`,
	}, "\n")

	got := RedactDisplaySecrets(input)

	for _, want := range []string{
		"Authorization: Basic [redacted]",
		"https://example.test/items?redacted#redacted",
		"api_key=[redacted]",
		"client_secret: [redacted]",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("RedactDisplaySecrets() missing %q:\n%s", want, got)
		}
	}
	for _, reject := range []string{
		"dXNlcjpwYXNz",
		"token=secret",
		"access_token=fragment-secret",
		"abc%2Fdef%3D",
		"json-secret",
	} {
		if strings.Contains(got, reject) {
			t.Fatalf("RedactDisplaySecrets() leaked %q:\n%s", reject, got)
		}
	}
}

func TestSanitizeDisplayPreviewRedactsCollapsesAndTrims(t *testing.T) {
	input := "  Authorization: Bearer secret-value\nhttps://example.test/items?token=secret#sig=fragment\t" + strings.Repeat("x", 80)

	got := SanitizeDisplayPreview(input, 90)

	if utf8.RuneCountInString(got) > 90 {
		t.Fatalf("SanitizeDisplayPreview() length = %d, want <= 90: %q", utf8.RuneCountInString(got), got)
	}
	if strings.ContainsAny(got, "\n\t") || strings.Contains(got, "  ") {
		t.Fatalf("SanitizeDisplayPreview() did not collapse whitespace: %q", got)
	}
	for _, reject := range []string{"secret-value", "token=secret", "sig=fragment"} {
		if strings.Contains(got, reject) {
			t.Fatalf("SanitizeDisplayPreview() leaked %q: %q", reject, got)
		}
	}
	if !strings.Contains(got, "Authorization: Bearer [redacted]") {
		t.Fatalf("SanitizeDisplayPreview() missing authorization redaction: %q", got)
	}
}

func TestLooksSensitiveContentDetectsFragmentSecrets(t *testing.T) {
	if !LooksSensitiveContent("https://example.test/callback#access_token=secret-value") {
		t.Fatalf("LooksSensitiveContent() = false, want true for fragment access token")
	}
}
