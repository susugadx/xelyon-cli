package openrouter

import (
	"encoding/json"
	"net/http"
	"testing"
)

func writeOpenRouterChatCompletionsSSE(w http.ResponseWriter, chunks ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	for _, chunk := range chunks {
		_, _ = w.Write([]byte("data: " + chunk + "\n\n"))
	}
	_, _ = w.Write([]byte("data: [DONE]\n\n"))
}

func writeOpenRouterAnthropicSSE(w http.ResponseWriter, chunks ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	for _, chunk := range chunks {
		_, _ = w.Write([]byte("data: " + chunk + "\n\n"))
	}
}

func requireOpenRouterDiagnosticCheckStatus(t *testing.T, report DiagnosticReport, name string, status DiagnosticStatus) DiagnosticCheck {
	t.Helper()

	check, ok := openRouterDiagnosticCheckByName(report, name)
	if !ok {
		t.Fatalf("%s check missing: %#v", name, report.Checks)
	}
	if check.Status != status {
		t.Fatalf("%s check = %#v; want %s", name, check, status)
	}
	return check
}

func requireOpenRouterDiagnosticCheckAbsent(t *testing.T, report DiagnosticReport, name string) {
	t.Helper()

	if check, ok := openRouterDiagnosticCheckByName(report, name); ok {
		t.Fatalf("%s check was added unexpectedly: %#v", name, check)
	}
}

func openRouterDiagnosticCheckByName(report DiagnosticReport, name string) (DiagnosticCheck, bool) {
	for _, check := range report.Checks {
		if check.Name == name {
			return check, true
		}
	}
	return DiagnosticCheck{}, false
}

func decodeOpenRouterDiagnosticPreviewBodyForTest(t *testing.T, body any) map[string]any {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal preview body: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode preview body: %v\n%s", err, string(payload))
	}
	return decoded
}
