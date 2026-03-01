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
		{name: "openai", provider: "openai", want: 100000},
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
