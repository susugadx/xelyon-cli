package cmd

import (
	"encoding/json"
	"testing"
)

func TestRootDoctorLocalCapabilityFlagsSkipEndpointAndAuth(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		env            map[string]string
		omittedChecks  []string
		requiredDetail string
	}{
		{
			name: "deepseek",
			args: []string{
				"doctor", "deepseek",
				"--model", "deepseek-v4-flash",
				"--catalog-model", "deepseek-v4-flash",
				"--capabilities",
				"--require-capability", "function_calling",
				"--json",
			},
			env: map[string]string{
				"DEEPSEEK_API_KEY":          "",
				"DEEPSEEK_API_URL":          "://bad",
				"DEEPSEEK_FUNCTION_CALLING": "1",
				"XELYON_MODEL":              "",
			},
			omittedChecks:  []string{"auth", "endpoint"},
			requiredDetail: "function_calling=ok",
		},
		{
			name: "groq",
			args: []string{
				"doctor", "groq",
				"--model", "meta-llama/llama-4-scout-17b-16e-instruct",
				"--catalog-model", "meta-llama/llama-4-scout-17b-16e-instruct",
				"--capabilities",
				"--require-capability", "function_calling",
				"--json",
			},
			env: map[string]string{
				"GROQ_API_KEY":          "",
				"GROQ_API_URL":          "://bad",
				"GROQ_FUNCTION_CALLING": "1",
				"XELYON_MODEL":          "",
			},
			omittedChecks:  []string{"auth", "endpoint"},
			requiredDetail: "function_calling=ok",
		},
		{
			name: "kimi",
			args: []string{
				"doctor", "kimi",
				"--model", "kimi-k2.6",
				"--catalog-model", "kimi-k2.6",
				"--capabilities",
				"--require-capability", "chat_completions",
				"--json",
			},
			env: map[string]string{
				"MOONSHOT_API_KEY":      "",
				"KIMI_API_URL":          "://bad",
				"KIMI_FUNCTION_CALLING": "1",
				"XELYON_MODEL":          "",
			},
			omittedChecks:  []string{"auth", "api_url", "api_url_path"},
			requiredDetail: "chat_completions=ok",
		},
		{
			name: "gemini",
			args: []string{
				"doctor", "gemini",
				"--model", "gemini-3.1-pro-preview-customtools",
				"--catalog-model", "gemini-3.1-pro-preview-customtools",
				"--capabilities",
				"--require-capability", "image_input",
				"--json",
			},
			env: map[string]string{
				"GEMINI_API_KEY": "",
				"GEMINI_API_URL": "://bad",
				"XELYON_MODEL":   "",
			},
			omittedChecks:  []string{"auth", "endpoint"},
			requiredDetail: "image_input=ok",
		},
		{
			name: "claude",
			args: []string{
				"doctor", "claude",
				"--model", "claude-sonnet-4-6",
				"--catalog-model", "claude-sonnet-4-6",
				"--capabilities",
				"--require-capability", "image_input",
				"--json",
			},
			env: map[string]string{
				"ANTHROPIC_API_KEY":       "",
				"ANTHROPIC_API_URL":       "://bad",
				"CLAUDE_FUNCTION_CALLING": "1",
				"XELYON_MODEL":            "",
			},
			omittedChecks:  []string{"auth", "endpoint"},
			requiredDetail: "image_input=ok",
		},
		{
			name: "openrouter",
			args: []string{
				"doctor", "openrouter",
				"--model", "corp-openrouter-gpt",
				"--catalog-model", "openai/gpt-5.4",
				"--capabilities",
				"--require-capability", "function_calling",
				"--json",
			},
			env: map[string]string{
				"OPENROUTER_API_KEY":          "",
				"OPENROUTER_API_URL":          "://bad",
				"OPENROUTER_FUNCTION_CALLING": "1",
				"XELYON_MODEL":                "",
			},
			omittedChecks:  []string{"auth", "endpoint"},
			requiredDetail: "function_calling=ok",
		},
		{
			name: "openai",
			args: []string{
				"doctor", "openai",
				"--model", "corp-openai-deployment",
				"--catalog-model", "gpt-5.4",
				"--capabilities",
				"--require-capability", "function_calling",
				"--json",
			},
			env: map[string]string{
				"OPENAI_API_KEY":          "",
				"OPENAI_API_URL":          "://bad",
				"OPENAI_RESPONSES_URL":    "://bad",
				"OPENAI_FUNCTION_CALLING": "1",
				"XELYON_MODEL":            "",
			},
			omittedChecks:  []string{"auth", "api_url", "api_url_path", "responses_url", "responses_url_path"},
			requiredDetail: "function_calling=ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			out := newRootCommandExecutionTest(t)
			rootCmd.SetArgs(tt.args)
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("doctor command failed: %v\n%s", err, out.String())
			}

			var raw map[string]json.RawMessage
			if err := json.Unmarshal(out.Bytes(), &raw); err != nil {
				t.Fatalf("unmarshal raw report: %v\n%s", err, out.String())
			}
			requireDoctorJSONFields(t, raw, "capabilities")

			report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
			requireNoDoctorJSONChecks(t, report.Checks, tt.omittedChecks...)

			required := requireDoctorJSONCheck(t, report.Checks, "required_capability")
			requireDoctorJSONCheckStatus(t, required, "ok")
			requireDoctorJSONCheckDetailContains(t, required, tt.requiredDetail)
		})
	}
}
