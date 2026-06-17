package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

type doctorJSONCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type doctorJSONContractReport struct {
	Checks []doctorJSONCheck `json:"checks"`
}

func unmarshalDoctorJSON[T any](t *testing.T, out *bytes.Buffer) T {
	t.Helper()
	var report T
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, out.String())
	}
	return report
}

func requireContainsAll(t *testing.T, label, got string, wants []string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Fatalf("%s = %q, want substring %q", label, got, want)
		}
	}
}

func requireDoctorJSONCheck(t *testing.T, checks []doctorJSONCheck, name string) doctorJSONCheck {
	t.Helper()
	for _, check := range checks {
		if check.Name == name {
			return check
		}
	}
	t.Fatalf("missing %s check: %#v", name, checks)
	return doctorJSONCheck{}
}

func requireNoDoctorJSONChecks(t *testing.T, checks []doctorJSONCheck, names ...string) {
	t.Helper()
	for _, check := range checks {
		for _, name := range names {
			if check.Name == name {
				t.Fatalf("%s check should be skipped: %#v", check.Name, checks)
			}
		}
	}
}

func requireDoctorJSONCheckStatus(t *testing.T, check doctorJSONCheck, want string) {
	t.Helper()
	if check.Status != want {
		t.Fatalf("%s check status = %q, want %q: %#v", check.Name, check.Status, want, check)
	}
}

func requireDoctorJSONCheckDetailContains(t *testing.T, check doctorJSONCheck, want string) {
	t.Helper()
	if !strings.Contains(check.Detail, want) {
		t.Fatalf("%s check detail = %q, want substring %q", check.Name, check.Detail, want)
	}
}
