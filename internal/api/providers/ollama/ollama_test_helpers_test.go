package ollama

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func mockAPIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

// ollamaStreamingHandler はOllama形式のJSON Linesストリーミングハンドラー。
func ollamaStreamingHandler(texts []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		for _, text := range texts {
			resp := OllamaStreamResponse{
				Message: OllamaMessageContent{Content: text},
				Done:    false,
			}
			data, _ := json.Marshal(resp)
			fmt.Fprintln(w, string(data))
			flusher.Flush()
		}

		doneResp := OllamaStreamResponse{Done: true}
		data, _ := json.Marshal(doneResp)
		fmt.Fprintln(w, string(data))
		flusher.Flush()
	}
}
