package cmd

import (
	"strings"
	"testing"
)

func TestExecute_ExitsOnError(t *testing.T) {
	result := runRootExecuteHelper(t, "unknown_flag")
	if result.exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", result.exitCode)
	}
	if !strings.Contains(result.combinedOutput(), "unknown flag") {
		t.Fatalf("combined output = %q, want cobra error message", result.combinedOutput())
	}
}

func TestExecute_ExitsWithCIUsageCodeForCobraUsageErrors(t *testing.T) {
	tests := []struct {
		name         string
		helperMode   string
		wantFragment string
	}{
		{name: "unknown long flag", helperMode: "unknown_flag_ci", wantFragment: "unknown flag"},
		{name: "unknown long flag before policy", helperMode: "unknown_flag_then_ci", wantFragment: "unknown flag"},
		{name: "unknown long flag before equals policy", helperMode: "unknown_flag_then_ci_equals", wantFragment: "unknown flag"},
		{name: "unknown shorthand flag", helperMode: "unknown_shorthand_flag_ci", wantFragment: "unknown shorthand flag"},
		{name: "missing flag argument", helperMode: "missing_flag_argument_ci", wantFragment: "flag needs an argument"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runRootExecuteHelper(t, tt.helperMode)
			if result.exitCode != 2 {
				t.Fatalf("exit code = %d, want 2\noutput=%s", result.exitCode, result.combinedOutput())
			}
			if !strings.Contains(result.combinedOutput(), tt.wantFragment) {
				t.Fatalf("combined output = %q, want %q", result.combinedOutput(), tt.wantFragment)
			}
		})
	}
}

func TestExecute_EmitsHeadlessJSONForCobraUsageErrors(t *testing.T) {
	tests := []struct {
		name              string
		helperMode        string
		wantErrorFragment string
	}{
		{name: "headless unknown flag", helperMode: "headless_unknown_flag_ci"},
		{name: "headless unknown flag policy after parse error", helperMode: "headless_unknown_flag_ci_policy_after"},
		{name: "headless false after parse error", helperMode: "headless_unknown_flag_ci_headless_false_after"},
		{name: "headless unknown flag before subcommand word", helperMode: "headless_unknown_flag_ci_before_doctor"},
		{name: "headless before unknown shorthand cluster", helperMode: "headless_unknown_shorthand_cluster_ci", wantErrorFragment: "unknown shorthand flag"},
		{name: "json output text after parse error", helperMode: "json_unknown_flag_ci_text_after"},
		{name: "json uppercase output format", helperMode: "json_unknown_flag_ci_uppercase"},
		{name: "json whitespace output format policy after parse error", helperMode: "json_unknown_flag_ci_whitespace_policy_after"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runRootExecuteHelper(t, tt.helperMode)
			wantErrorFragment := tt.wantErrorFragment
			if wantErrorFragment == "" {
				wantErrorFragment = "unknown flag"
			}
			if result.exitCode != 2 {
				t.Fatalf("exit code = %d, want 2\nstdout=%s\nstderr=%s", result.exitCode, result.stdout, result.stderr)
			}
			for _, fragment := range []string{
				`"schema_version": "xelyon.headless.v1"`,
				`"failure_reason": "usage_error"`,
				`"exit_policy": "ci"`,
				`"recommended_exit_code": 2`,
				wantErrorFragment,
			} {
				if !strings.Contains(result.stdout, fragment) {
					t.Fatalf("stdout = %q, want %q", result.stdout, fragment)
				}
			}
			if strings.Contains(result.stderr, "Usage:") {
				t.Fatalf("stderr contains Cobra usage after headless JSON error:\n%s", result.stderr)
			}
			if !strings.Contains(result.stderr, "headless execution failed") {
				t.Fatalf("stderr = %q, want headless execution failed", result.stderr)
			}
		})
	}
}

func TestExecute_DoesNotEmitHeadlessJSONForUnparsedHeadlessFlags(t *testing.T) {
	tests := []struct {
		name         string
		helperMode   string
		wantFragment string
	}{
		{name: "explicit false headless before parse error", helperMode: "headless_false_unknown_flag", wantFragment: "unknown flag"},
		{name: "headless only after parse error", helperMode: "unknown_flag_then_headless", wantFragment: "unknown flag"},
		{name: "quiet shorthand cluster before headless", helperMode: "quiet_cluster_then_headless", wantFragment: "unknown shorthand flag"},
		{name: "auto approve shorthand cluster before headless", helperMode: "auto_approve_cluster_then_headless", wantFragment: "unknown shorthand flag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runRootExecuteHelper(t, tt.helperMode)
			if result.exitCode != 1 {
				t.Fatalf("exit code = %d, want 1\nstdout=%s\nstderr=%s", result.exitCode, result.stdout, result.stderr)
			}
			if strings.Contains(result.stdout, `"schema_version": "xelyon.headless.v1"`) {
				t.Fatalf("stdout contains headless JSON for non-headless parse error:\n%s", result.stdout)
			}
			if !strings.Contains(result.stderr, "Usage:") {
				t.Fatalf("stderr = %q, want Cobra usage", result.stderr)
			}
			if !strings.Contains(result.stderr, tt.wantFragment) {
				t.Fatalf("stderr = %q, want %q", result.stderr, tt.wantFragment)
			}
		})
	}
}

func TestExecute_KeepsSubcommandFlagParseErrorsText(t *testing.T) {
	tests := []struct {
		name       string
		helperMode string
		wantFlag   string
	}{
		{name: "headless before doctor", helperMode: "headless_before_doctor", wantFlag: "--headless"},
		{name: "json before doctor", helperMode: "json_before_doctor", wantFlag: "--output-format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runRootExecuteHelper(t, tt.helperMode)
			if result.exitCode != 1 {
				t.Fatalf("exit code = %d, want 1\nstdout=%s\nstderr=%s", result.exitCode, result.stdout, result.stderr)
			}
			if strings.Contains(result.stdout, `"schema_version": "xelyon.headless.v1"`) {
				t.Fatalf("stdout contains headless JSON for subcommand flag error:\n%s", result.stdout)
			}
			if !strings.Contains(result.stderr, "Usage:") {
				t.Fatalf("stderr = %q, want Cobra usage", result.stderr)
			}
			if !strings.Contains(result.stderr, "unknown flag: "+tt.wantFlag) {
				t.Fatalf("stderr = %q, want unknown flag %s", result.stderr, tt.wantFlag)
			}
		})
	}
}

func TestExecute_PreservesHeadlessUsageErrorInputMetadata(t *testing.T) {
	tests := []struct {
		name          string
		helperMode    string
		wantFragments []string
	}{
		{
			name:       "prompt file after invalid flag",
			helperMode: "headless_unknown_flag_ci_prompt_file_after",
			wantFragments: []string{
				`"source": "prompt_file"`,
				`"prompt_file": "prompt.md"`,
			},
		},
		{
			name:       "image after invalid flag",
			helperMode: "headless_unknown_flag_ci_image_after",
			wantFragments: []string{
				`"source": "args"`,
				`"image": {`,
				`"path": "screen.png"`,
				`"provider_supported": true`,
			},
		},
		{
			name:       "attached shorthand image after invalid flag",
			helperMode: "headless_unknown_flag_ci_image_attached_shorthand",
			wantFragments: []string{
				`"source": "args"`,
				`"bytes": 6`,
				`"image": {`,
				`"path": "screen.png"`,
				`"provider_supported": true`,
			},
		},
		{
			name:       "equals shorthand image after invalid flag",
			helperMode: "headless_unknown_flag_ci_image_equals_shorthand",
			wantFragments: []string{
				`"source": "args"`,
				`"bytes": 6`,
				`"image": {`,
				`"path": "screen.png"`,
				`"provider_supported": false`,
			},
		},
		{
			name:       "valid shorthand cluster with model before invalid flag",
			helperMode: "headless_unknown_flag_ci_model_cluster",
			wantFragments: []string{
				`"source": "args"`,
				`"bytes": 6`,
			},
		},
		{
			name:       "valid shorthand cluster with image before invalid flag",
			helperMode: "headless_unknown_flag_ci_image_cluster",
			wantFragments: []string{
				`"source": "args"`,
				`"bytes": 6`,
				`"image": {`,
				`"path": "screen.png"`,
				`"provider_supported": true`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runRootExecuteHelper(t, tt.helperMode)
			if result.exitCode != 2 {
				t.Fatalf("exit code = %d, want 2\nstdout=%s\nstderr=%s", result.exitCode, result.stdout, result.stderr)
			}
			if !strings.Contains(result.stdout, `"schema_version": "xelyon.headless.v1"`) {
				t.Fatalf("stdout = %q, want headless JSON", result.stdout)
			}
			for _, fragment := range tt.wantFragments {
				if !strings.Contains(result.stdout, fragment) {
					t.Fatalf("stdout = %q, want %q", result.stdout, fragment)
				}
			}
			if strings.Contains(result.stderr, "Usage:") {
				t.Fatalf("stderr contains Cobra usage after headless JSON error:\n%s", result.stderr)
			}
		})
	}
}

func TestExecute_ExitsWithHeadlessRecommendedCode(t *testing.T) {
	result := runRootExecuteHelper(t, "headless_usage_ci")
	if result.exitCode != 2 {
		t.Fatalf("exit code = %d, want 2\noutput=%s", result.exitCode, result.combinedOutput())
	}
	if !strings.Contains(result.combinedOutput(), `"failure_reason": "usage_error"`) {
		t.Fatalf("combined output = %q, want usage_error JSON", result.combinedOutput())
	}
	if !strings.Contains(result.combinedOutput(), `"recommended_exit_code": 2`) {
		t.Fatalf("combined output = %q, want recommended_exit_code 2", result.combinedOutput())
	}
}

func TestExecute_ExitsWithHeadlessToolErrorRecommendedCode(t *testing.T) {
	result := runRootExecuteHelper(t, "headless_tool_error_ci")
	if result.exitCode != 4 {
		t.Fatalf("exit code = %d, want 4\noutput=%s", result.exitCode, result.combinedOutput())
	}
	if !strings.Contains(result.combinedOutput(), `"failure_reason": "tool_error"`) {
		t.Fatalf("combined output = %q, want tool_error JSON", result.combinedOutput())
	}
	if !strings.Contains(result.combinedOutput(), `"recommended_exit_code": 4`) {
		t.Fatalf("combined output = %q, want recommended_exit_code 4", result.combinedOutput())
	}
}

func TestExecute_ExitsWithHeadlessReadOnlyViolationRecommendedCode(t *testing.T) {
	result := runRootExecuteHelper(t, "headless_read_only_violation_ci")
	if result.exitCode != 8 {
		t.Fatalf("exit code = %d, want 8\noutput=%s", result.exitCode, result.combinedOutput())
	}
	if !strings.Contains(result.combinedOutput(), `"failure_reason": "read_only_violation"`) {
		t.Fatalf("combined output = %q, want read_only_violation JSON", result.combinedOutput())
	}
	if !strings.Contains(result.combinedOutput(), `"recommended_exit_code": 8`) {
		t.Fatalf("combined output = %q, want recommended_exit_code 8", result.combinedOutput())
	}
	if strings.Contains(result.combinedOutput(), "Usage:") {
		t.Fatalf("combined output contains Cobra usage after headless JSON error:\n%s", result.combinedOutput())
	}
}

func TestExecute_ExitsWithCIUsageCodeForRootErrors(t *testing.T) {
	result := runRootExecuteHelper(t, "root_usage_ci")
	if result.exitCode != 2 {
		t.Fatalf("exit code = %d, want 2\noutput=%s", result.exitCode, result.combinedOutput())
	}
	if !strings.Contains(result.combinedOutput(), "invalid --output-format") {
		t.Fatalf("combined output = %q, want output-format error", result.combinedOutput())
	}
}

func TestExecute_InvalidExitPolicyKeepsLegacyErrorCode(t *testing.T) {
	result := runRootExecuteHelper(t, "invalid_exit_policy")
	if result.exitCode != 1 {
		t.Fatalf("exit code = %d, want 1\noutput=%s", result.exitCode, result.combinedOutput())
	}
	if !strings.Contains(result.combinedOutput(), "invalid --exit-code-policy") {
		t.Fatalf("combined output = %q, want exit policy error", result.combinedOutput())
	}
}
