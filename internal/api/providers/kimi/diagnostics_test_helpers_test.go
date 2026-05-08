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
	for _, check := range report.Checks {
		if check.Name == name && check.Status == status {
			return true
		}
	}
	return false
}
