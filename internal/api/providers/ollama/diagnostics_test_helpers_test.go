package ollama

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newOllamaDiagnosticServer(t *testing.T, models []string, chat http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			if r.Method != http.MethodGet {
				t.Fatalf("tags method = %s, want GET", r.Method)
			}
			tags := OllamaTagsResponse{}
			for _, model := range models {
				tags.Models = append(tags.Models, OllamaModel{Name: model})
			}
			_ = json.NewEncoder(w).Encode(tags)
		case "/api/chat":
			if r.Method != http.MethodPost {
				t.Fatalf("chat method = %s, want POST", r.Method)
			}
			chat(w, r)
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
}

func writeOllamaDiagnosticJSONL(w http.ResponseWriter, responses ...OllamaStreamResponse) {
	w.Header().Set("Content-Type", "application/x-ndjson")
	for _, response := range responses {
		data, _ := json.Marshal(response)
		fmt.Fprintln(w, string(data))
	}
}

func ollamaDiagnosticCheckByName(report DiagnosticReport, name string) (DiagnosticCheck, bool) {
	for _, check := range report.Checks {
		if check.Name == name {
			return check, true
		}
	}
	return DiagnosticCheck{}, false
}

func decodeOllamaDiagnosticPreviewBody(t *testing.T, body any) OllamaRequest {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal preview body: %v", err)
	}
	var decoded OllamaRequest
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode preview body: %v\n%s", err, string(payload))
	}
	return decoded
}
