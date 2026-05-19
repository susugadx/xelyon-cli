package kimi

import (
	"fmt"
	"net/http"
	"testing"
)

func writeKimiDiagnosticSSE(t *testing.T, w http.ResponseWriter, chunks ...string) {
	t.Helper()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	for _, chunk := range chunks {
		if _, err := fmt.Fprintf(w, "data: %s\n\n", chunk); err != nil {
			t.Fatalf("write chunk: %v", err)
		}
	}
	if _, err := fmt.Fprint(w, "data: [DONE]\n\n"); err != nil {
		t.Fatalf("write done: %v", err)
	}
}

func hasKimiDiagnosticCheck(report DiagnosticReport, name string, status DiagnosticStatus) bool {
	check, ok := kimiDiagnosticCheckByName(report, name)
	return ok && check.Status == status
}

func requireKimiDiagnosticCheckStatus(t *testing.T, report DiagnosticReport, name string, status DiagnosticStatus) DiagnosticCheck {
	t.Helper()

	check, ok := kimiDiagnosticCheckByName(report, name)
	if !ok || check.Status != status {
		t.Fatalf("%s check = %#v, %v; want %s", name, check, ok, status)
	}
	return check
}

func kimiDiagnosticCheckByName(report DiagnosticReport, name string) (DiagnosticCheck, bool) {
	for _, check := range report.Checks {
		if check.Name == name {
			return check, true
		}
	}
	return DiagnosticCheck{}, false
}

func hasKimiDiagnosticCheckName(report DiagnosticReport, name string) bool {
	_, ok := kimiDiagnosticCheckByName(report, name)
	return ok
}

func requireKimiSmokeRequest(t *testing.T, smoke *DiagnosticSmokeResult, name string) DiagnosticSmokeRequestResult {
	t.Helper()
	if smoke == nil {
		t.Fatalf("Smoke = nil, want request %q", name)
	}
	var found *DiagnosticSmokeRequestResult
	for i := range smoke.Requests {
		if smoke.Requests[i].Name != name {
			continue
		}
		if found != nil {
			t.Fatalf("Smoke has duplicate request name %q: %#v", name, smoke.Requests)
		}
		found = &smoke.Requests[i]
	}
	if found == nil {
		t.Fatalf("Smoke missing request %q: %#v", name, smoke.Requests)
	}
	return *found
}
