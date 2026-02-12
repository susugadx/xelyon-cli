package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoWithRetry_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "POST", server.URL, nil)
	resp, err := DoWithRetry(context.Background(), server.Client(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDoWithRetry_429ThenSuccess(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if callCount.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "POST", server.URL, nil)
	start := time.Now()
	resp, err := DoWithRetry(context.Background(), server.Client(), req)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if callCount.Load() != 2 {
		t.Errorf("expected 2 calls, got %d", callCount.Load())
	}
	if elapsed < 1*time.Second {
		t.Errorf("expected at least 1s wait (Retry-After), got %v", elapsed)
	}
}

func TestDoWithRetry_429WithRetryAfterHeader(t *testing.T) {
	retrySeconds := 2
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if callCount.Add(1) == 1 {
			w.Header().Set("Retry-After", strconv.Itoa(retrySeconds))
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "POST", server.URL, nil)
	start := time.Now()
	resp, err := DoWithRetry(context.Background(), server.Client(), req)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if elapsed < time.Duration(retrySeconds)*time.Second {
		t.Errorf("expected at least %ds wait, got %v", retrySeconds, elapsed)
	}
}

func TestDoWithRetry_429ExceedsMaxRetries(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "POST", server.URL, nil)
	resp, err := DoWithRetry(context.Background(), server.Client(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	// maxRetries=3 なので合計4回呼ばれる（初回 + 3リトライ）、最後の429がそのまま返る
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected 429 after max retries, got %d", resp.StatusCode)
	}
	if callCount.Load() != 4 {
		t.Errorf("expected 4 calls (1 initial + 3 retries), got %d", callCount.Load())
	}
}

func TestDoWithRetry_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())

	req, _ := http.NewRequestWithContext(ctx, "POST", server.URL, nil)

	done := make(chan struct{})
	var retErr error
	go func() {
		_, retErr = DoWithRetry(ctx, server.Client(), req)
		close(done)
	}()

	// 少し待ってからキャンセル
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("DoWithRetry did not return after context cancel")
	}

	if retErr != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", retErr)
	}
}

func TestRetryAfterDuration_WithHeader(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{"Retry-After": []string{"30"}},
	}
	d := RetryAfterDuration(resp, 0)
	if d != 30*time.Second {
		t.Errorf("expected 30s, got %v", d)
	}
}

func TestRetryAfterDuration_WithoutHeader(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 20 * time.Second},
		{1, 40 * time.Second},
		{2, 60 * time.Second}, // capped at maxBackoff
		{3, 60 * time.Second}, // still capped
	}

	for _, tt := range tests {
		d := RetryAfterDuration(resp, tt.attempt)
		if d != tt.expected {
			t.Errorf("attempt %d: expected %v, got %v", tt.attempt, tt.expected, d)
		}
	}
}
