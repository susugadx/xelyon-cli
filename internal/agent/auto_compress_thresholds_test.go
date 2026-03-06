package agent

import "testing"

func TestGetProviderCompressThreshold(t *testing.T) {
	cases := []struct {
		name     string
		provider string
		model    string
		want     int
	}{
		{name: "gemini", provider: "gemini", want: 180000},
		{name: "deepseek", provider: "deepseek", want: 50000},
		{name: "openai legacy", provider: "openai", model: "gpt-5", want: 100000},
		{name: "openai gpt-5.4", provider: "openai", model: "gpt-5.4", want: 260000},
		{name: "openai gpt-5.4-pro", provider: "openai", model: "gpt-5.4-pro", want: 260000},
		{name: "unknown", provider: "unknown", want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := GetProviderCompressThreshold(tc.provider, tc.model)
			if got != tc.want {
				t.Fatalf("GetProviderCompressThreshold(%q, %q) = %d, want %d", tc.provider, tc.model, got, tc.want)
			}
		})
	}
}

func TestGetProviderCompressThreshold_OpenAI_GPT54(t *testing.T) {
	threshold := GetProviderCompressThreshold("openai", "gpt-5.4")
	if threshold != 260000 {
		t.Errorf("expected 260000 for gpt-5.4, got %d", threshold)
	}
}

func TestGetProviderCompressThreshold_OpenAI_GPT54Pro(t *testing.T) {
	threshold := GetProviderCompressThreshold("openai", "gpt-5.4-pro")
	if threshold != 260000 {
		t.Errorf("expected 260000 for gpt-5.4-pro, got %d", threshold)
	}
}

func TestGetProviderCompressThreshold_OpenAI_Legacy(t *testing.T) {
	threshold := GetProviderCompressThreshold("openai", "gpt-5")
	if threshold != 100000 {
		t.Errorf("expected 100000 for legacy gpt-5, got %d", threshold)
	}
}
