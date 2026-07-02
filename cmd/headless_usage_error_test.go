package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/agent"
)

func TestRootCommand_InvalidExitCodePolicyReturnsCommandError(t *testing.T) {
	withRootCommandTest(t)
	rootCmd.SetArgs([]string{"--exit-code-policy", "strict", "--no-update-check", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid exit code policy")
	}
	if !strings.Contains(err.Error(), "invalid --exit-code-policy") {
		t.Fatalf("unexpected error message: %v", err)
	}
	var exitErr exitCodeCarrier
	if errors.As(err, &exitErr) {
		t.Fatalf("invalid exit-code-policy error carries exit code %d, want normal command error", exitErr.ExitCode())
	}
}

func TestRootCommand_HeadlessInvalidExitCodePolicyReturnsJSONUsageError(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "headless",
			args: []string{"--headless", "--exit-code-policy", "strict", "--provider", "ollama", "--no-update-check", "hello"},
		},
		{
			name: "output format json",
			args: []string{"--output-format", "json", "--exit-code-policy", "strict", "--provider", "ollama", "--no-update-check", "hello"},
		},
		{
			name: "output format uppercase json",
			args: []string{"--output-format", "JSON", "--exit-code-policy", "strict", "--provider", "ollama", "--no-update-check", "hello"},
		},
		{
			name: "output format whitespace json",
			args: []string{"--output-format", " json ", "--exit-code-policy", "strict", "--provider", "ollama", "--no-update-check", "hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withRootCommandTest(t)
			t.Setenv("HOME", t.TempDir())

			parsed, output, stderr, err := executeRootCommandForHeadlessJSONTest(t, tt.args, "")

			requireHeadlessUsageErrorJSON(t, parsed, output, stderr, err, 1, agent.HeadlessExitPolicyLegacy, "invalid --exit-code-policy")
			requireHeadlessInput(t, parsed.Input, agent.HeadlessInputSourceArgs, "", len([]byte("hello")))
		})
	}
}

func TestRootCommand_HeadlessFlagParseErrorReturnsJSONUsageError(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantMessage string
	}{
		{
			name:        "headless unknown long flag",
			args:        []string{"--headless", "--exit-code-policy", "ci", "--bad-flag", "hello"},
			wantMessage: "unknown flag",
		},
		{
			name:        "headless unknown shorthand flag",
			args:        []string{"--headless", "--exit-code-policy", "ci", "-z", "hello"},
			wantMessage: "unknown shorthand flag",
		},
		{
			name:        "headless missing flag argument",
			args:        []string{"--headless", "--exit-code-policy", "ci", "--provider"},
			wantMessage: "flag needs an argument",
		},
		{
			name:        "output format uppercase json",
			args:        []string{"--output-format", "JSON", "--exit-code-policy", "ci", "--bad-flag", "hello"},
			wantMessage: "unknown flag",
		},
		{
			name:        "output format whitespace json",
			args:        []string{"--output-format", " json ", "--exit-code-policy", "ci", "--bad-flag", "hello"},
			wantMessage: "unknown flag",
		},
		{
			name:        "headless unknown flag before subcommand word",
			args:        []string{"--headless", "--exit-code-policy", "ci", "--bad-flag", "doctor"},
			wantMessage: "unknown flag",
		},
		{
			name:        "headless false after parse error keeps headless JSON",
			args:        []string{"--headless", "--exit-code-policy", "ci", "--bad-flag", "--headless=false", "hello"},
			wantMessage: "unknown flag",
		},
		{
			name:        "output format text after parse error keeps headless JSON",
			args:        []string{"--output-format", "json", "--exit-code-policy", "ci", "--bad-flag", "--output-format", "text", "hello"},
			wantMessage: "unknown flag",
		},
		{
			name:        "headless before unknown shorthand cluster",
			args:        []string{"--headless", "--exit-code-policy", "ci", "-qz", "hello"},
			wantMessage: "unknown shorthand flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withRootCommandTest(t)

			parsed, output, stderr, err := executeRootCommandForHeadlessJSONTest(t, tt.args, "")

			requireHeadlessUsageErrorJSON(t, parsed, output, stderr, err, 2, agent.HeadlessExitPolicyCI, tt.wantMessage)
			if !rootCmd.SilenceUsage {
				t.Fatal("rootCmd.SilenceUsage = false, want true after printing headless flag-parse JSON")
			}
		})
	}
}

func TestRootCommand_FlagParseErrorWithoutParsedHeadlessDoesNotReturnHeadlessJSON(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantFragment string
	}{
		{
			name:         "explicit false headless before parse error",
			args:         []string{"--headless=false", "--bad-flag", "prompt"},
			wantFragment: "unknown flag",
		},
		{
			name:         "headless only after parse error",
			args:         []string{"--bad-flag", "--headless", "prompt"},
			wantFragment: "unknown flag",
		},
		{
			name:         "quiet shorthand cluster before headless",
			args:         []string{"-qz", "--headless", "prompt"},
			wantFragment: "unknown shorthand flag",
		},
		{
			name:         "auto approve shorthand cluster before headless",
			args:         []string{"-yz", "--headless", "prompt"},
			wantFragment: "unknown shorthand flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withRootCommandTest(t)

			var stdout, stderr bytes.Buffer
			rootCmd.SetOut(&stdout)
			rootCmd.SetErr(&stderr)
			rootCmd.SetArgs(tt.args)
			rootExecutionArgs = append(rootExecutionArgs[:0], tt.args...)
			t.Cleanup(func() {
				rootCmd.SetOut(nil)
				rootCmd.SetErr(nil)
				rootExecutionArgs = nil
			})

			err := rootCmd.Execute()
			output := stdout.String()
			stderrText := stderr.String()
			combined := output + stderrText

			if err == nil {
				t.Fatal("expected root flag parse error")
			}
			if strings.Contains(output, `"schema_version": "xelyon.headless.v1"`) {
				t.Fatalf("stdout contains headless JSON for non-headless parse error:\n%s", output)
			}
			if !strings.Contains(combined, "Usage:") {
				t.Fatalf("combined output = %q, want Cobra usage", combined)
			}
			if !strings.Contains(combined, tt.wantFragment) && !strings.Contains(err.Error(), tt.wantFragment) {
				t.Fatalf("combined output = %q, error = %q, want %q", combined, err.Error(), tt.wantFragment)
			}
		})
	}
}

func TestRootCommand_HeadlessFlagParseErrorPreservesRawInputMetadata(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		wantSource        agent.HeadlessInputSource
		wantPromptFile    string
		wantBytes         int
		wantImagePath     string
		wantImageProvider bool
	}{
		{
			name:           "prompt file after invalid flag",
			args:           []string{"--headless", "--exit-code-policy", "ci", "--bad-flag", "--prompt-file", "prompt.md"},
			wantSource:     agent.HeadlessInputSourcePromptFile,
			wantPromptFile: "prompt.md",
		},
		{
			name:              "image and provider after invalid flag",
			args:              []string{"--headless", "--exit-code-policy", "ci", "--bad-flag", "--provider", "openai", "--image", "screen.png"},
			wantSource:        agent.HeadlessInputSourceArgs,
			wantImagePath:     "screen.png",
			wantImageProvider: true,
		},
		{
			name:              "attached shorthand image and provider before invalid flag",
			args:              []string{"--headless", "--exit-code-policy", "ci", "-popenai", "-iscreen.png", "--bad-flag", "prompt"},
			wantSource:        agent.HeadlessInputSourceArgs,
			wantBytes:         len([]byte("prompt")),
			wantImagePath:     "screen.png",
			wantImageProvider: true,
		},
		{
			name:              "equals shorthand image and provider before invalid flag",
			args:              []string{"--headless", "--exit-code-policy", "ci", "-p=groq", "-i=screen.png", "--bad-flag", "prompt"},
			wantSource:        agent.HeadlessInputSourceArgs,
			wantBytes:         len([]byte("prompt")),
			wantImagePath:     "screen.png",
			wantImageProvider: false,
		},
		{
			name:       "valid shorthand cluster with model before invalid flag",
			args:       []string{"--headless", "--exit-code-policy", "ci", "-qm", "gpt", "--bad-flag", "prompt"},
			wantSource: agent.HeadlessInputSourceArgs,
			wantBytes:  len([]byte("prompt")),
		},
		{
			name:              "valid shorthand cluster with image before invalid flag",
			args:              []string{"--headless", "--exit-code-policy", "ci", "--provider", "openai", "-qiscreen.png", "--bad-flag", "prompt"},
			wantSource:        agent.HeadlessInputSourceArgs,
			wantBytes:         len([]byte("prompt")),
			wantImagePath:     "screen.png",
			wantImageProvider: true,
		},
		{
			name:              "unsupported image provider after invalid flag",
			args:              []string{"--headless", "--exit-code-policy", "ci", "--bad-flag", "--provider", "groq", "--image", "screen.png"},
			wantSource:        agent.HeadlessInputSourceArgs,
			wantImagePath:     "screen.png",
			wantImageProvider: false,
		},
		{
			name:       "subcommand word after root invalid flag stays positional input",
			args:       []string{"--headless", "--exit-code-policy", "ci", "--bad-flag", "doctor"},
			wantSource: agent.HeadlessInputSourceArgs,
			wantBytes:  len([]byte("doctor")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withRootCommandTest(t)

			parsed, output, stderr, err := executeRootCommandForHeadlessJSONTest(t, tt.args, "")

			requireHeadlessUsageErrorJSON(t, parsed, output, stderr, err, 2, agent.HeadlessExitPolicyCI, "unknown flag")
			requireHeadlessInput(t, parsed.Input, tt.wantSource, tt.wantPromptFile, tt.wantBytes)
			if tt.wantImagePath != "" {
				requireHeadlessInputImage(t, parsed.Input, tt.wantImagePath, "", 0, tt.wantImageProvider)
			}
		})
	}
}

func TestRootCommand_SubcommandFlagParseErrorDoesNotReturnHeadlessJSON(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantFlag string
	}{
		{
			name:     "headless before doctor",
			args:     []string{"--headless", "doctor"},
			wantFlag: "--headless",
		},
		{
			name:     "output format json before doctor",
			args:     []string{"--output-format", "json", "doctor"},
			wantFlag: "--output-format",
		},
		{
			name:     "headless inside doctor",
			args:     []string{"doctor", "--headless"},
			wantFlag: "--headless",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withRootCommandTest(t)

			var stdout, stderr bytes.Buffer
			rootCmd.SetOut(&stdout)
			rootCmd.SetErr(&stderr)
			rootCmd.SetArgs(tt.args)
			t.Cleanup(func() {
				rootCmd.SetOut(nil)
				rootCmd.SetErr(nil)
			})

			err := rootCmd.Execute()
			output := stdout.String()
			stderrText := stderr.String()
			combined := output + stderrText

			if err == nil {
				t.Fatal("expected subcommand flag parse error")
			}
			var exitErr exitCodeCarrier
			if errors.As(err, &exitErr) {
				t.Fatalf("subcommand flag parse error carries exit code %d, want normal Cobra error", exitErr.ExitCode())
			}
			if strings.Contains(output, `"schema_version": "xelyon.headless.v1"`) {
				t.Fatalf("stdout contains headless JSON for subcommand flag error:\n%s", output)
			}
			if !strings.Contains(combined, "Usage:") {
				t.Fatalf("combined output = %q, want Cobra usage", combined)
			}
			if !strings.Contains(combined, "unknown flag: "+tt.wantFlag) && !strings.Contains(err.Error(), "unknown flag: "+tt.wantFlag) {
				t.Fatalf("combined output = %q, error = %q, want unknown flag %s", combined, err.Error(), tt.wantFlag)
			}
		})
	}
}

func TestRootUsageErrorIntentFromArgs(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		command           func(t *testing.T) *cobra.Command
		wantHeadlessJSON  bool
		wantPolicy        agent.HeadlessExitPolicy
		wantSource        agent.HeadlessInputSource
		wantPromptFile    string
		wantBytes         int
		wantImagePath     string
		wantImageProvider bool
		boundHeadless     bool
		boundOutputFormat string
	}{
		{
			name:             "headless policy after parse error",
			args:             []string{"--headless", "--bad-flag", "--exit-code-policy", "ci", "prompt"},
			wantHeadlessJSON: true,
			wantPolicy:       agent.HeadlessExitPolicyCI,
			wantSource:       agent.HeadlessInputSourceArgs,
			wantBytes:        len([]byte("prompt")),
			boundHeadless:    true,
		},
		{
			name:              "json output format normalizes whitespace",
			args:              []string{"--output-format", " json ", "--bad-flag", "--exit-code-policy=ci", "prompt"},
			wantHeadlessJSON:  true,
			wantPolicy:        agent.HeadlessExitPolicyCI,
			wantSource:        agent.HeadlessInputSourceArgs,
			wantBytes:         len([]byte("prompt")),
			boundOutputFormat: " json ",
		},
		{
			name:             "explicit false headless is not json",
			args:             []string{"--headless=false", "--bad-flag", "prompt"},
			wantHeadlessJSON: false,
			wantPolicy:       agent.HeadlessExitPolicyLegacy,
			wantSource:       agent.HeadlessInputSourceArgs,
			wantBytes:        len([]byte("prompt")),
		},
		{
			name:             "subcommand word after root parse error remains root json",
			args:             []string{"--headless", "--bad-flag", "doctor"},
			wantHeadlessJSON: true,
			wantPolicy:       agent.HeadlessExitPolicyLegacy,
			wantSource:       agent.HeadlessInputSourceArgs,
			wantBytes:        len([]byte("doctor")),
			boundHeadless:    true,
		},
		{
			name:             "headless false after parse error does not revoke root json",
			args:             []string{"--headless", "--bad-flag", "--headless=false", "prompt"},
			wantHeadlessJSON: true,
			wantPolicy:       agent.HeadlessExitPolicyLegacy,
			wantSource:       agent.HeadlessInputSourceArgs,
			wantBytes:        len([]byte("prompt")),
			boundHeadless:    true,
		},
		{
			name:              "output format text after parse error does not revoke root json",
			args:              []string{"--output-format", "json", "--bad-flag", "--output-format", "text", "prompt"},
			wantHeadlessJSON:  true,
			wantPolicy:        agent.HeadlessExitPolicyLegacy,
			wantSource:        agent.HeadlessInputSourceArgs,
			wantBytes:         len([]byte("prompt")),
			boundOutputFormat: "json",
		},
		{
			name:             "headless false before parse error revokes root json",
			args:             []string{"--headless", "--headless=false", "--bad-flag", "prompt"},
			wantHeadlessJSON: false,
			wantPolicy:       agent.HeadlessExitPolicyLegacy,
			wantSource:       agent.HeadlessInputSourceArgs,
			wantBytes:        len([]byte("prompt")),
		},
		{
			name:              "output format text before parse error revokes root json",
			args:              []string{"--output-format", "json", "--output-format", "text", "--bad-flag", "prompt"},
			wantHeadlessJSON:  false,
			wantPolicy:        agent.HeadlessExitPolicyLegacy,
			wantSource:        agent.HeadlessInputSourceArgs,
			wantBytes:         len([]byte("prompt")),
			boundOutputFormat: "text",
		},
		{
			name:             "headless request only after parse error does not create root json",
			args:             []string{"--bad-flag", "--headless", "prompt"},
			wantHeadlessJSON: false,
			wantPolicy:       agent.HeadlessExitPolicyLegacy,
			wantSource:       agent.HeadlessInputSourceArgs,
			wantBytes:        len([]byte("prompt")),
		},
		{
			name:             "quiet shorthand cluster before headless does not create root json",
			args:             []string{"-qz", "--headless", "prompt"},
			wantHeadlessJSON: false,
			wantPolicy:       agent.HeadlessExitPolicyLegacy,
			wantSource:       agent.HeadlessInputSourceArgs,
			wantBytes:        len([]byte("prompt")),
		},
		{
			name:             "auto approve shorthand cluster before headless does not create root json",
			args:             []string{"-yz", "--headless", "prompt"},
			wantHeadlessJSON: false,
			wantPolicy:       agent.HeadlessExitPolicyLegacy,
			wantSource:       agent.HeadlessInputSourceArgs,
			wantBytes:        len([]byte("prompt")),
		},
		{
			name:             "headless before quiet shorthand cluster remains root json",
			args:             []string{"--headless", "-qz", "prompt"},
			wantHeadlessJSON: true,
			wantPolicy:       agent.HeadlessExitPolicyLegacy,
			wantSource:       agent.HeadlessInputSourceArgs,
			wantBytes:        len([]byte("prompt")),
			boundHeadless:    true,
		},
		{
			name:             "headless before subcommand is not root json",
			args:             []string{"--headless", "doctor"},
			command:          rootUsageErrorIntentDoctorCommandForTest,
			wantHeadlessJSON: false,
			wantPolicy:       agent.HeadlessExitPolicyLegacy,
			wantSource:       agent.HeadlessInputSourceArgs,
			wantBytes:        len([]byte("doctor")),
			boundHeadless:    true,
		},
		{
			name:              "json before subcommand is not root json",
			args:              []string{"--output-format", "json", "doctor"},
			command:           rootUsageErrorIntentDoctorCommandForTest,
			wantHeadlessJSON:  false,
			wantPolicy:        agent.HeadlessExitPolicyLegacy,
			wantSource:        agent.HeadlessInputSourceArgs,
			wantBytes:         len([]byte("doctor")),
			boundOutputFormat: "json",
		},
		{
			name:             "root value flag before subcommand is not input",
			args:             []string{"--provider", "ollama", "--headless", "doctor"},
			command:          rootUsageErrorIntentDoctorCommandForTest,
			wantHeadlessJSON: false,
			wantPolicy:       agent.HeadlessExitPolicyLegacy,
			wantSource:       agent.HeadlessInputSourceArgs,
			wantBytes:        len([]byte("doctor")),
			boundHeadless:    true,
		},
		{
			name:              "root shorthand value flag before subcommand is not input",
			args:              []string{"-p", "ollama", "--output-format", "json", "doctor"},
			command:           rootUsageErrorIntentDoctorCommandForTest,
			wantHeadlessJSON:  false,
			wantPolicy:        agent.HeadlessExitPolicyLegacy,
			wantSource:        agent.HeadlessInputSourceArgs,
			wantBytes:         len([]byte("doctor")),
			boundOutputFormat: "json",
		},
		{
			name:             "subcommand flags do not imply root headless json",
			args:             []string{"doctor", "--headless", "--bad-flag"},
			command:          rootUsageErrorIntentDoctorCommandForTest,
			wantHeadlessJSON: false,
			wantPolicy:       agent.HeadlessExitPolicyLegacy,
			wantSource:       agent.HeadlessInputSourceArgs,
			wantBytes:        len([]byte("doctor")),
		},
		{
			name:             "prompt file after parse error becomes input metadata",
			args:             []string{"--headless", "--bad-flag", "--prompt-file", "prompt.md"},
			wantHeadlessJSON: true,
			wantPolicy:       agent.HeadlessExitPolicyLegacy,
			wantSource:       agent.HeadlessInputSourcePromptFile,
			wantPromptFile:   "prompt.md",
			boundHeadless:    true,
		},
		{
			name:              "image provider after parse error becomes input metadata",
			args:              []string{"--headless", "--bad-flag", "--provider", "openai", "--image", "screen.png"},
			wantHeadlessJSON:  true,
			wantPolicy:        agent.HeadlessExitPolicyLegacy,
			wantSource:        agent.HeadlessInputSourceArgs,
			wantImagePath:     "screen.png",
			wantImageProvider: true,
			boundHeadless:     true,
		},
		{
			name:              "attached shorthand image provider after parse error becomes input metadata",
			args:              []string{"--headless", "-popenai", "-iscreen.png", "--bad-flag", "prompt"},
			wantHeadlessJSON:  true,
			wantPolicy:        agent.HeadlessExitPolicyLegacy,
			wantSource:        agent.HeadlessInputSourceArgs,
			wantBytes:         len([]byte("prompt")),
			wantImagePath:     "screen.png",
			wantImageProvider: true,
			boundHeadless:     true,
		},
		{
			name:              "equals shorthand image provider after parse error becomes input metadata",
			args:              []string{"--headless", "-p=groq", "-i=screen.png", "--bad-flag", "prompt"},
			wantHeadlessJSON:  true,
			wantPolicy:        agent.HeadlessExitPolicyLegacy,
			wantSource:        agent.HeadlessInputSourceArgs,
			wantBytes:         len([]byte("prompt")),
			wantImagePath:     "screen.png",
			wantImageProvider: false,
			boundHeadless:     true,
		},
		{
			name:             "valid shorthand cluster with model does not become input metadata",
			args:             []string{"--headless", "-qm", "gpt", "--bad-flag", "prompt"},
			wantHeadlessJSON: true,
			wantPolicy:       agent.HeadlessExitPolicyLegacy,
			wantSource:       agent.HeadlessInputSourceArgs,
			wantBytes:        len([]byte("prompt")),
			boundHeadless:    true,
		},
		{
			name:              "valid shorthand cluster with image becomes input metadata",
			args:              []string{"--headless", "--provider", "openai", "-qiscreen.png", "--bad-flag", "prompt"},
			wantHeadlessJSON:  true,
			wantPolicy:        agent.HeadlessExitPolicyLegacy,
			wantSource:        agent.HeadlessInputSourceArgs,
			wantBytes:         len([]byte("prompt")),
			wantImagePath:     "screen.png",
			wantImageProvider: true,
			boundHeadless:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withRootCommandTest(t)
			command := rootCmd
			if tt.command != nil {
				command = tt.command(t)
			}
			headless = tt.boundHeadless
			if tt.boundOutputFormat != "" {
				outputFormat = tt.boundOutputFormat
			}
			intent := rootUsageErrorIntentFromCommandArgs(command, tt.args)

			if intent.writeHeadlessJSON != tt.wantHeadlessJSON {
				t.Fatalf("writeHeadlessJSON = %t, want %t", intent.writeHeadlessJSON, tt.wantHeadlessJSON)
			}
			if intent.exitPolicy != tt.wantPolicy {
				t.Fatalf("exitPolicy = %q, want %q", intent.exitPolicy, tt.wantPolicy)
			}
			requireHeadlessInput(t, &intent.input, tt.wantSource, tt.wantPromptFile, tt.wantBytes)
			if tt.wantImagePath != "" {
				requireHeadlessInputImage(t, &intent.input, tt.wantImagePath, "", 0, tt.wantImageProvider)
			}
		})
	}
}

func rootUsageErrorIntentDoctorCommandForTest(t *testing.T) *cobra.Command {
	t.Helper()
	for _, command := range rootCmd.Commands() {
		if command.Name() == "doctor" {
			return command
		}
	}
	t.Fatal("doctor command not found")
	return nil
}

func requireHeadlessUsageErrorJSON(t *testing.T, parsed agent.HeadlessResult, output string, stderr string, err error, wantCode int, wantPolicy agent.HeadlessExitPolicy, messageFragment string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected headless usage error")
	}
	requireCommandExitCode(t, err, wantCode)
	if output == "" {
		t.Fatal("stdout is empty, want headless JSON")
	}
	if strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr contains Cobra usage after headless JSON error:\n%s", stderr)
	}
	if parsed.Status != agent.HeadlessStatusError {
		t.Fatalf("status = %q, want %q", parsed.Status, agent.HeadlessStatusError)
	}
	if parsed.Error == nil || parsed.Error.Type != agent.HeadlessErrorTypeConfig {
		t.Fatalf("error = %+v, want %s", parsed.Error, agent.HeadlessErrorTypeConfig)
	}
	if !strings.Contains(parsed.Error.Message, messageFragment) {
		t.Fatalf("error message = %q, want %q", parsed.Error.Message, messageFragment)
	}
	if parsed.FailureReason != agent.HeadlessFailureReasonUsageError {
		t.Fatalf("failure_reason = %q, want %q", parsed.FailureReason, agent.HeadlessFailureReasonUsageError)
	}
	if parsed.ExitPolicy != wantPolicy {
		t.Fatalf("exit_policy = %q, want %q", parsed.ExitPolicy, wantPolicy)
	}
	if parsed.RecommendedExitCode != wantCode {
		t.Fatalf("recommended_exit_code = %d, want %d", parsed.RecommendedExitCode, wantCode)
	}
}
