package cmd

import (
	"encoding/json"
	"sort"
	"testing"
)

func requireDoctorJSONFields(t *testing.T, raw map[string]json.RawMessage, fields ...string) {
	t.Helper()
	for _, field := range fields {
		if _, ok := raw[field]; !ok {
			t.Fatalf("JSON field %q missing from report keys %v", field, sortedDoctorJSONFieldNames(raw))
		}
	}
}

func sortedDoctorJSONFieldNames(raw map[string]json.RawMessage) []string {
	fields := make([]string, 0, len(raw))
	for field := range raw {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return fields
}
