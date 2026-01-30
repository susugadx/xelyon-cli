package serper

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/cache"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Go version", "go version"},
		{"  Go  ", "go"},
		{"GO VERSION", "go version"},
		{" React hooks ", "react hooks"},
		{"  ", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := normalizeQuery(tt.input); got != tt.want {
			t.Errorf("normalizeQuery(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestWebSearchWithCache_CacheHit(t *testing.T) {
	// モックサーバー設定
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"organic":[{"title":"Test","link":"https://test.com","snippet":"Test snippet"}]}`))
	}))
	defer server.Close()

	// 環境変数設定
	os.Setenv("SERPER_API_URL", server.URL)
	os.Setenv("SERPER_API_KEY", "test-key")
	defer os.Unsetenv("SERPER_API_URL")
	defer os.Unsetenv("SERPER_API_KEY")

	// キャッシュを手動で初期化（テスト用）
	webSearchCache = cache.New(cache.Config{
		Enabled:    true,
		Capacity:   10,
		DefaultTTL: 5 * time.Minute,
	}, nil)
	cacheOnce = sync.Once{} // リセット

	// 1回目: API呼び出し
	result1, cached1, err := WebSearchWithCache("test query")
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}
	if cached1 {
		t.Error("First call should not be cached")
	}
	if callCount != 1 {
		t.Errorf("Expected 1 API call, got %d", callCount)
	}

	// 2回目: キャッシュヒット
	result2, cached2, err := WebSearchWithCache("test query")
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}
	if !cached2 {
		t.Error("Second call should be cached")
	}
	if callCount != 1 {
		t.Errorf("Expected still 1 API call (cached), got %d", callCount)
	}

	// 結果が同じことを確認
	if result1 != result2 {
		t.Error("Cached result should be the same as original")
	}
}

func TestWebSearchWithCache_CacheNormalization(t *testing.T) {
	// モックサーバー設定
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"organic":[{"title":"Go Test","link":"https://go.dev","snippet":"Go programming"}]}`))
	}))
	defer server.Close()

	// 環境変数設定
	os.Setenv("SERPER_API_URL", server.URL)
	os.Setenv("SERPER_API_KEY", "test-key")
	defer os.Unsetenv("SERPER_API_URL")
	defer os.Unsetenv("SERPER_API_KEY")

	// キャッシュを手動で初期化（テスト用）
	webSearchCache = cache.New(cache.Config{
		Enabled:    true,
		Capacity:   10,
		DefaultTTL: 5 * time.Minute,
	}, nil)
	cacheOnce = sync.Once{} // リセット

	// 1回目: "Go version"
	_, cached1, err := WebSearchWithCache("Go version")
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}
	if cached1 {
		t.Error("First call should not be cached")
	}

	// 2回目: "go version" (同じクエリとして扱われるべき)
	_, cached2, err := WebSearchWithCache("go version")
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}
	if !cached2 {
		t.Error("Second call should be cached (normalized query)")
	}

	// 3回目: "  GO VERSION  " (同じクエリとして扱われるべき)
	_, cached3, err := WebSearchWithCache("  GO VERSION  ")
	if err != nil {
		t.Fatalf("Third call failed: %v", err)
	}
	if !cached3 {
		t.Error("Third call should be cached (normalized query)")
	}

	// API呼び出しは1回のみ
	if callCount != 1 {
		t.Errorf("Expected 1 API call (all normalized to same query), got %d", callCount)
	}
}

func TestWebSearchWithCache_CacheDisabled(t *testing.T) {
	// モックサーバー設定
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"organic":[{"title":"Test","link":"https://test.com","snippet":"Test"}]}`))
	}))
	defer server.Close()

	// 環境変数設定
	os.Setenv("SERPER_API_URL", server.URL)
	os.Setenv("SERPER_API_KEY", "test-key")
	defer os.Unsetenv("SERPER_API_URL")
	defer os.Unsetenv("SERPER_API_KEY")

	// キャッシュを無効化（nil）+ cacheOnceをリセット
	// 注: cacheOnce.Do(initCache) が呼ばれるが、initCache内で
	// config.WebSearch.CacheEnabled がfalseならwebSearchCache=nilのまま
	webSearchCache = nil

	// 設定を一時的に変更してキャッシュ無効に
	originalConfig := config.GetGlobalConfig()
	testConfig := config.DefaultConfig()
	testConfig.WebSearch.CacheEnabled = false
	config.SetGlobalConfig(testConfig)
	defer config.SetGlobalConfig(originalConfig)

	// cacheOnce をリセット
	cacheOnce = sync.Once{}

	// 1回目
	_, cached1, err := WebSearchWithCache("disabled-test")
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}
	if cached1 {
		t.Error("Should not be cached when cache is disabled")
	}

	// 2回目（キャッシュ無効なので再度API呼び出し）
	_, cached2, err := WebSearchWithCache("disabled-test")
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}
	if cached2 {
		t.Error("Should not be cached when cache is disabled")
	}

	// API呼び出しは2回
	if callCount != 2 {
		t.Errorf("Expected 2 API calls (cache disabled), got %d", callCount)
	}
}

func TestInitCache(t *testing.T) {
	// グローバル設定を一時的に変更
	originalConfig := config.GetGlobalConfig()
	defer config.SetGlobalConfig(originalConfig)

	// テスト用設定
	testConfig := config.DefaultConfig()
	testConfig.WebSearch.CacheEnabled = true
	testConfig.WebSearch.CacheSize = 50
	testConfig.WebSearch.CacheTTL = 600
	config.SetGlobalConfig(testConfig)

	// キャッシュをリセット
	webSearchCache = nil
	cacheOnce = sync.Once{}

	// initCache を呼び出し
	initCache()

	// キャッシュが初期化されていることを確認
	if webSearchCache == nil {
		t.Error("Cache should be initialized")
	}
}

func TestInitCache_Disabled(t *testing.T) {
	// グローバル設定を一時的に変更
	originalConfig := config.GetGlobalConfig()
	defer config.SetGlobalConfig(originalConfig)

	// キャッシュ無効の設定
	testConfig := config.DefaultConfig()
	testConfig.WebSearch.CacheEnabled = false
	config.SetGlobalConfig(testConfig)

	// キャッシュをリセット
	webSearchCache = nil
	cacheOnce = sync.Once{}

	// initCache を呼び出し
	initCache()

	// キャッシュが初期化されていないことを確認
	if webSearchCache != nil {
		t.Error("Cache should not be initialized when disabled")
	}
}
