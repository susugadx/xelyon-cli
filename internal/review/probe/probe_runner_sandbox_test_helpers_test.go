package probe

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

type probeModeBlockedCase struct {
	name          string
	request       ReviewProbeRequest
	errorContains string
}

func runProbeModeBlockedCases(t *testing.T, mode ReviewProbeMode, cases []probeModeBlockedCase) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
			runner := NewProbeRunner(repo)

			req := tt.request
			req.Mode = mode
			result, err := runner.Run(context.Background(), req)
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			assertProbeBlockedWithoutExecution(t, result, tt.errorContains)
		})
	}
}

func assertProbeBlockedWithoutExecution(t *testing.T, result ReviewProbeResult, errorContains string) {
	t.Helper()

	if result.Status != ReviewProbeBlocked {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbeBlocked, result.Error)
	}
	if len(result.CommandResults) != 0 {
		t.Fatalf("len(CommandResults) = %d, want 0", len(result.CommandResults))
	}
	if !strings.Contains(result.Error, errorContains) {
		t.Fatalf("Error = %q, want to contain %q", result.Error, errorContains)
	}
}

type isolatedProbeEnvExpectation struct {
	repoRootKey   string
	repoRootValue string
	modeRootKey   string
	modeRootValue string
	rootDir       string
}

func decodeEnvMapFromFirstCommandOutput(t *testing.T, result ReviewProbeResult) map[string]string {
	t.Helper()

	if len(result.CommandResults) == 0 {
		t.Fatal("CommandResults is empty")
	}
	var envMap map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(result.CommandResults[0].Output)), &envMap); err != nil {
		t.Fatalf("json.Unmarshal(output) error = %v, output=%q", err, result.CommandResults[0].Output)
	}
	return envMap
}

func assertIsolatedProbeEnv(t *testing.T, envMap map[string]string, expected isolatedProbeEnvExpectation) {
	t.Helper()

	if envMap["SECRET_TOKEN"] != "" || envMap["OPENAI_API_KEY"] != "" {
		t.Fatalf("secret env leaked: %#v", envMap)
	}
	if envMap[expected.repoRootKey] != expected.repoRootValue {
		t.Fatalf("%s = %q, want %q", expected.repoRootKey, envMap[expected.repoRootKey], expected.repoRootValue)
	}
	if envMap[expected.modeRootKey] != expected.modeRootValue {
		t.Fatalf("%s = %q, want %q", expected.modeRootKey, envMap[expected.modeRootKey], expected.modeRootValue)
	}

	if envMap["HOME"] != filepath.Join(expected.rootDir, "home") {
		t.Fatalf("HOME = %q, want %q", envMap["HOME"], filepath.Join(expected.rootDir, "home"))
	}
	if envMap["TMPDIR"] != filepath.Join(expected.rootDir, "tmp") {
		t.Fatalf("TMPDIR = %q, want %q", envMap["TMPDIR"], filepath.Join(expected.rootDir, "tmp"))
	}
	if envMap["GOCACHE"] != filepath.Join(expected.rootDir, "cache", "go-build") {
		t.Fatalf("GOCACHE = %q, want %q", envMap["GOCACHE"], filepath.Join(expected.rootDir, "cache", "go-build"))
	}
	if envMap["GOMODCACHE"] != filepath.Join(expected.rootDir, "cache", "go-mod") {
		t.Fatalf("GOMODCACHE = %q, want %q", envMap["GOMODCACHE"], filepath.Join(expected.rootDir, "cache", "go-mod"))
	}
	if envMap["GOTMPDIR"] != filepath.Join(expected.rootDir, "tmp", "go") {
		t.Fatalf("GOTMPDIR = %q, want %q", envMap["GOTMPDIR"], filepath.Join(expected.rootDir, "tmp", "go"))
	}
	if envMap["GOTOOLCHAIN"] != "local" || envMap["GOPROXY"] != "off" || envMap["GOSUMDB"] != "off" {
		t.Fatalf("go env hardening mismatch: %#v", envMap)
	}
	if envMap["PYTHONDONTWRITEBYTECODE"] != "1" || envMap["PYTHONNOUSERSITE"] != "1" || envMap["PIP_NO_INDEX"] != "1" {
		t.Fatalf("python env hardening mismatch: %#v", envMap)
	}
}

func assertCommandResolutionPassed(t *testing.T, result ReviewProbeResult, outputContains string) {
	t.Helper()

	if result.Status != ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbePassed, result.Error)
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}
	if outputContains != "" && !strings.Contains(result.CommandResults[0].Output, outputContains) {
		t.Fatalf("output = %q, want to contain %q", result.CommandResults[0].Output, outputContains)
	}
}

func assertCommandResolutionBlocked(t *testing.T, result ReviewProbeResult) {
	t.Helper()

	if result.Status != ReviewProbeBlocked {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbeBlocked, result.Error)
	}
	if len(result.CommandResults) != 0 {
		t.Fatalf("len(CommandResults) = %d, want 0", len(result.CommandResults))
	}
}
