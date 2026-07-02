package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent"
)

func executeRootCommandForHeadlessJSONTest(t *testing.T, args []string, stdin string) (agent.HeadlessResult, string, string, error) {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetIn(strings.NewReader(stdin))
	rootCmd.SetArgs(args)
	rootExecutionArgs = append(rootExecutionArgs[:0], args...)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetIn(nil)
		rootExecutionArgs = nil
	})

	execErr := rootCmd.Execute()

	output := strings.TrimSpace(stdout.String())

	var parsed agent.HeadlessResult
	if output != "" {
		if err := json.Unmarshal([]byte(output), &parsed); err != nil {
			t.Fatalf("stdout is not headless JSON: %v\noutput=%q", err, output)
		}
	}
	return parsed, output, stderr.String(), execErr
}

func requireCommandExitCode(t *testing.T, err error, want int) {
	t.Helper()
	var exitErr exitCodeCarrier
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %T, want exit code carrier", err)
	}
	if exitErr.ExitCode() != want {
		t.Fatalf("ExitCode() = %d, want %d", exitErr.ExitCode(), want)
	}
}

func requireHeadlessInput(t *testing.T, input *agent.HeadlessInput, source agent.HeadlessInputSource, path string, bytes int) {
	t.Helper()
	if input == nil {
		t.Fatal("input = nil")
	}
	if input.Source != source {
		t.Fatalf("input.source = %q, want %q", input.Source, source)
	}
	if input.PromptFile != path {
		t.Fatalf("input.prompt_file = %q, want %q", input.PromptFile, path)
	}
	if input.Bytes != bytes {
		t.Fatalf("input.bytes = %d, want %d", input.Bytes, bytes)
	}
}

func requireHeadlessInputImage(t *testing.T, input *agent.HeadlessInput, path string, mimeType string, bytes int64, providerSupported bool) {
	t.Helper()
	if input == nil {
		t.Fatal("input = nil")
	}
	if input.Image == nil {
		t.Fatal("input.image = nil")
	}
	if input.Image.Path != path {
		t.Fatalf("input.image.path = %q, want %q", input.Image.Path, path)
	}
	if input.Image.MIMEType != mimeType {
		t.Fatalf("input.image.mime_type = %q, want %q", input.Image.MIMEType, mimeType)
	}
	if input.Image.Bytes != bytes {
		t.Fatalf("input.image.bytes = %d, want %d", input.Image.Bytes, bytes)
	}
	if input.Image.ProviderSupported != providerSupported {
		t.Fatalf("input.image.provider_supported = %t, want %t", input.Image.ProviderSupported, providerSupported)
	}
}

func requireHeadlessConfigError(t *testing.T, parsed agent.HeadlessResult, stderr string, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected headless execution error")
	}
	if !strings.Contains(err.Error(), "headless execution failed") {
		t.Fatalf("error = %v, want headless execution failed", err)
	}
	if strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr contains Cobra usage after headless JSON error:\n%s", stderr)
	}
	if parsed.SchemaVersion != agent.HeadlessSchemaVersion {
		t.Fatalf("schema_version = %q, want %q", parsed.SchemaVersion, agent.HeadlessSchemaVersion)
	}
	if parsed.Status != agent.HeadlessStatusError {
		t.Fatalf("status = %q, want %q", parsed.Status, agent.HeadlessStatusError)
	}
	if parsed.Error == nil || parsed.Error.Type != agent.HeadlessErrorTypeConfig {
		t.Fatalf("error = %+v, want %s", parsed.Error, agent.HeadlessErrorTypeConfig)
	}
	if parsed.FailureReason != agent.HeadlessFailureReasonUsageError {
		t.Fatalf("failure_reason = %q, want %q", parsed.FailureReason, agent.HeadlessFailureReasonUsageError)
	}
	if parsed.ExitPolicy != agent.HeadlessExitPolicyLegacy {
		t.Fatalf("exit_policy = %q, want %q", parsed.ExitPolicy, agent.HeadlessExitPolicyLegacy)
	}
	if parsed.RecommendedExitCode != 1 {
		t.Fatalf("recommended_exit_code = %d, want 1", parsed.RecommendedExitCode)
	}
}

func writeHeadlessImageTestFile(t *testing.T, name string, body []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
