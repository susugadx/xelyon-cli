package commandoutputs

import (
	"strings"
	"testing"
)

func TestBuildReplacementFailureCompactRedactsProviderFacingText(t *testing.T) {
	command := "go test ./... --token=command-secret " + strings.Repeat("-run TestVeryLongCommandSegment ", 8)
	output := strings.Repeat(strings.Join([]string{
		"Error: exit status 1",
		"Authorization: Bearer abcdef",
		"https://example.test/items?token=secret#sig=fragment-secret",
		"main.go:12: undefined: value",
	}, "\n")+"\n", 120)

	replacement, reason, ok := BuildReplacement(NewRequest(command, output))
	if !ok || reason != "" {
		t.Fatalf("BuildReplacement() = (%#v, %q, %v), want failure compact", replacement, reason, ok)
	}

	text := replacement.Text()
	for _, want := range []string{
		"classifier=validation_failure",
		"--token=[redacted]",
		`Authorization: Bearer [redacted]`,
		"https://example.test/items?token=[redacted]",
		"...",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("failure compact missing sanitized marker %q:\n%s", want, text)
		}
	}
	for _, reject := range []string{
		"Bearer abcdef",
		"token=secret",
		"sig=fragment-secret",
		"command-secret",
		"--token=command-secret",
	} {
		if strings.Contains(text, reject) {
			t.Fatalf("failure compact leaked %q:\n%s", reject, text)
		}
	}
}

func TestBuildReplacementValidationSuccessRedactsCommandHeader(t *testing.T) {
	output := strings.Repeat("ok\tgithub.com/acme/project\t0.001s\n", 80)

	replacement, reason, ok := BuildReplacement(NewRequest("go test ./... --token=command-secret", output))
	if !ok || reason != "" {
		t.Fatalf("BuildReplacement() = (%#v, %q, %v), want validation success compact", replacement, reason, ok)
	}

	text := replacement.Text()
	if !strings.Contains(text, "--token=[redacted]") {
		t.Fatalf("validation success compact missing redacted command token:\n%s", text)
	}
	for _, reject := range []string{"command-secret", "--token=command-secret"} {
		if strings.Contains(text, reject) {
			t.Fatalf("validation success compact leaked %q:\n%s", reject, text)
		}
	}
}
