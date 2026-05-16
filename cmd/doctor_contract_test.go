package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func requireDoctorContractTextContainsAll(t *testing.T, output string, wants []string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func renderedDoctorContractValue(t *testing.T, value any) string {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%T) error = %v", value, err)
	}
	return string(payload)
}
