package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProvider_Embed(t *testing.T) {
	tests := []struct {
		name        string
		texts       []string
		mockStatus  int
		mockBody    any
		wantErr     bool
		wantLen     int
		errContains string
	}{
		{
			name:       "正常系: バッチ送信",
			texts:      []string{"test1", "test2"},
			mockStatus: http.StatusOK,
			mockBody: EmbedResponse{
				Model: "qwen3-embedding:0.6b",
				Embeddings: [][]float32{
					{0.1, 0.2, 0.3},
					{0.4, 0.5, 0.6},
				},
			},
			wantErr: false,
			wantLen: 2,
		},
		{
			name:    "正常系: 空入力",
			texts:   []string{},
			wantErr: false,
			wantLen: 0,
		},
		{
			name:        "異常系: モデル未インストール(404)",
			texts:       []string{"test"},
			mockStatus:  http.StatusNotFound,
			mockBody:    map[string]string{"error": "model not found"},
			wantErr:     true,
			errContains: "api error: status 404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/embed" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				w.WriteHeader(tt.mockStatus)
				if tt.mockBody != nil {
					_ = json.NewEncoder(w).Encode(tt.mockBody)
				}
			}))
			defer ts.Close()

			p := New(ts.URL, "qwen3-embedding:0.6b")
			res, err := p.Embed(context.Background(), tt.texts)

			if (err != nil) != tt.wantErr {
				t.Errorf("Embed() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Embed() error = %v, errContains %v", err, tt.errContains)
				}
			}
			if len(res) != tt.wantLen {
				t.Errorf("Embed() len = %v, wantLen %v", len(res), tt.wantLen)
			}
		})
	}
}

func TestProvider_EmbedSingle(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(EmbedResponse{
			Model: "qwen3-embedding:0.6b",
			Embeddings: [][]float32{
				{0.1, 0.2, 0.3},
			},
		})
	}))
	defer ts.Close()

	p := New(ts.URL, "qwen3-embedding:0.6b")
	res, err := p.EmbedSingle(context.Background(), "test")
	if err != nil {
		t.Errorf("EmbedSingle() error = %v", err)
	}
	if len(res) != 3 {
		t.Errorf("EmbedSingle() len = %v, want %v", len(res), 3)
	}
}

func TestProvider_ConnectionRefused(t *testing.T) {
	// 起動していないポートを指定
	p := New("http://localhost:19999", "qwen3-embedding:0.6b")

	// Fast fail configuration for test
	p.httpClient.Timeout = 50 * time.Millisecond

	_, err := p.EmbedSingle(context.Background(), "test")
	if err == nil {
		t.Error("EmbedSingle() expected error but got nil")
	} else if !strings.Contains(err.Error(), "ollama is not running") && !strings.Contains(err.Error(), "failed to do request") {
		t.Errorf("EmbedSingle() error = %v", err)
	}
}

func TestProvider_ContextCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	p := New(ts.URL, "qwen3-embedding:0.6b")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 即座にキャンセル

	_, err := p.EmbedSingle(ctx, "test")
	if err == nil {
		t.Error("EmbedSingle() expected error due to context cancellation")
	}
}

func TestProvider_IsAvailable(t *testing.T) {
	tests := []struct {
		name       string
		mockStatus int
		mockBody   any
		want       bool
	}{
		{
			name:       "モデルあり",
			mockStatus: http.StatusOK,
			mockBody: map[string]any{
				"models": []map[string]string{
					{"name": "qwen3-embedding:0.6b"},
					{"name": "llama3:latest"},
				},
			},
			want: true,
		},
		{
			name:       "モデル接頭辞一致",
			mockStatus: http.StatusOK,
			mockBody: map[string]any{
				"models": []map[string]string{
					{"name": "qwen3-embedding:0.6b:latest"},
				},
			},
			want: true,
		},
		{
			name:       "モデルなし",
			mockStatus: http.StatusOK,
			mockBody: map[string]any{
				"models": []map[string]string{
					{"name": "llama3:latest"},
				},
			},
			want: false,
		},
		{
			name:       "サーバーエラー",
			mockStatus: http.StatusInternalServerError,
			mockBody:   nil,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/tags" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				w.WriteHeader(tt.mockStatus)
				if tt.mockBody != nil {
					_ = json.NewEncoder(w).Encode(tt.mockBody)
				}
			}))
			defer ts.Close()

			p := New(ts.URL, "qwen3-embedding:0.6b")
			got := p.IsAvailable(context.Background())
			if got != tt.want {
				t.Errorf("IsAvailable() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_IsAvailable_ConnectionFailed(t *testing.T) {
	p := New("http://localhost:19999", "qwen3-embedding:0.6b")
	got := p.IsAvailable(context.Background())
	if got != false {
		t.Errorf("IsAvailable() expected false on connection failure")
	}
}
