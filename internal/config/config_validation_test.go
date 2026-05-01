package config

import "testing"

func TestValidateModel(t *testing.T) {
	tests := []string{"any-model", "gpt-4", "deepseek-coder", ""}
	for _, model := range tests {
		if !ValidateModel(model) {
			t.Errorf("ValidateModel(%q) should always return true", model)
		}
	}
}
