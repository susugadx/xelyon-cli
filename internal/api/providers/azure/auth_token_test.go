package azure

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunAzureAuthTokenCommand_ReturnsFirstNonEmptyLine(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell token command test uses POSIX printf")
	}

	token, err := runAzureAuthTokenCommand(context.Background(), "printf '\\n token-from-command \\nignored\\n'", time.Second)
	if err != nil {
		t.Fatalf("runAzureAuthTokenCommand() error = %v", err)
	}
	if token != "token-from-command" {
		t.Fatalf("token = %q, want token-from-command", token)
	}
}

func TestRunAzureAuthTokenCommand_RedactsFailureOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell token command test uses POSIX shell")
	}

	_, err := runAzureAuthTokenCommand(context.Background(), "printf 'failed eyJabc1234567.eyJdef1234567.signature1234567' >&2; exit 2", time.Second)
	if err == nil {
		t.Fatal("runAzureAuthTokenCommand() error = nil, want failure")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "[REDACTED]") {
		t.Fatalf("error = %q, want redacted token", errMsg)
	}
	if strings.Contains(errMsg, "eyJabc1234567") {
		t.Fatalf("error = %q, should not include raw token", errMsg)
	}
}

func TestSanitizeAzureAuthCommandOutput_RedactsLongJWTBeforeTruncation(t *testing.T) {
	header := strings.Repeat("a", 260)
	claims := strings.Repeat("b", 260)
	signature := strings.Repeat("c", 80)
	token := header + "." + claims + "." + signature

	detail := sanitizeAzureAuthCommandOutput("command failed with token " + token)
	if !strings.Contains(detail, "[REDACTED]") {
		t.Fatalf("detail = %q, want redacted token", detail)
	}
	for _, leaked := range []string{header[:64], claims[:64], signature[:64]} {
		if strings.Contains(detail, leaked) {
			t.Fatalf("detail = %q, should not include raw JWT segment %q", detail, leaked)
		}
	}
}

func TestRunAzureAuthTokenCommand_TimesOutProcessGroup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group timeout test uses POSIX shell")
	}

	start := time.Now()
	_, err := runAzureAuthTokenCommand(
		context.Background(),
		"sh -c 'sleep 5 >&2'",
		100*time.Millisecond,
	)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("runAzureAuthTokenCommand() error = nil, want timeout")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("error = %v, want timeout", err)
	}
	if elapsed > time.Second {
		t.Fatalf("elapsed = %s, want timeout to return without waiting for child process", elapsed)
	}
}

func TestRunAzureAuthTokenCommand_ReturnsContextCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("cancellation test uses POSIX shell")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := runAzureAuthTokenCommand(ctx, "sleep 5", time.Second)
	if err == nil {
		t.Fatal("runAzureAuthTokenCommand() error = nil, want cancellation")
	}
	if !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("error = %v, want canceled", err)
	}
}

func TestParseAzureAuthTokenCommandTimeout(t *testing.T) {
	t.Setenv(authTokenCommandTimeoutEnv, "250ms")
	timeout, err := parseAzureAuthTokenCommandTimeout()
	if err != nil {
		t.Fatalf("parseAzureAuthTokenCommandTimeout() error = %v", err)
	}
	if timeout != 250*time.Millisecond {
		t.Fatalf("timeout = %s, want 250ms", timeout)
	}

	t.Setenv(authTokenCommandTimeoutEnv, "0")
	timeout, err = parseAzureAuthTokenCommandTimeout()
	if err == nil {
		t.Fatal("parseAzureAuthTokenCommandTimeout() error = nil, want invalid timeout")
	}
	if timeout != defaultAuthTokenCommandTimeout {
		t.Fatalf("timeout = %s, want default %s", timeout, defaultAuthTokenCommandTimeout)
	}
}
