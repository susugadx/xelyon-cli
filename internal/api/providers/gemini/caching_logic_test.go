package gemini

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/token"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestEstimateTokens(t *testing.T) {
	systemPrompt := "You are a helpful assistant."
	history := []api.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}

	// "You are a helpful assistant." (28) + "Hello" (5) + "Hi there!" (9) = 42
	expected := token.EstimateTokenCountForModel("gemini-1.5-pro", systemPrompt) +
		token.EstimateTokenCountForModel("gemini-1.5-pro", "Hello") +
		token.EstimateTokenCountForModel("gemini-1.5-pro", "Hi there!")
	actual := estimateTokens("gemini-1.5-pro", systemPrompt, history, nil)

	if actual != expected {
		t.Errorf("expected %d tokens, got %d", expected, actual)
	}
}

func TestCacheConstants(t *testing.T) {
	if minCacheTokens != 4096 {
		t.Fatalf("minCacheTokens = %d, want 4096", minCacheTokens)
	}
	if maxDiffMessages != 200 {
		t.Fatalf("maxDiffMessages = %d, want 200", maxDiffMessages)
	}
	if defaultCacheTTL != 3600 {
		t.Fatalf("defaultCacheTTL = %d, want 3600", defaultCacheTTL)
	}
}

func TestCacheExpireTime_SafetyMargin(t *testing.T) {
	before := time.Now()
	ttl := getCacheTTL()
	expireTime := time.Now().Add(time.Duration(ttl)*time.Second - 60*time.Second)
	after := time.Now()

	wantMin := before.Add(time.Duration(ttl)*time.Second - 60*time.Second)
	wantMax := after.Add(time.Duration(ttl)*time.Second - 60*time.Second)
	if expireTime.Before(wantMin) || expireTime.After(wantMax) {
		t.Fatalf("expireTime = %v, want between %v and %v", expireTime, wantMin, wantMax)
	}
}

func TestUpdateOrUseCache(t *testing.T) {
	// テスト用の閾値設定
	longContent := cacheEligibleContent()

	tests := []struct {
		name             string
		systemPrompt     string
		history          []api.Message
		activeCacheName  string
		cachedMsgCount   int
		expectCache      bool
		expectCreateCall bool
		expectDeleteCall bool
		expectDiffOnly   bool
	}{
		{
			name:         "Small content - No cache",
			systemPrompt: "sys",
			history:      []api.Message{{Role: "user", Content: "hi"}},
			expectCache:  false,
		},
		{
			name:             "Large content - Create cache",
			systemPrompt:     "sys",
			history:          []api.Message{{Role: "user", Content: longContent}, {Role: "user", Content: "last"}},
			expectCache:      true,
			expectCreateCall: true,
			expectDiffOnly:   true,
		},
		{
			name:             "Use existing cache - Small diff",
			systemPrompt:     "sys",
			history:          []api.Message{{Role: "user", Content: longContent}, {Role: "user", Content: "last"}, {Role: "assistant", Content: "res"}, {Role: "user", Content: "next"}},
			activeCacheName:  "cachedContents/old",
			cachedMsgCount:   2,
			expectCache:      true,
			expectCreateCall: false,
			expectDiffOnly:   true,
		},
		{
			name:             "Large diff - Recreate cache",
			systemPrompt:     "sys",
			history:          makeHistory(longContent, 2+maxDiffMessages+1),
			activeCacheName:  "cachedContents/old",
			cachedMsgCount:   2,
			expectCache:      true,
			expectCreateCall: true,
			expectDeleteCall: true,
			expectDiffOnly:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックサーバー設定（実際には使われないが念のため）
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			defer server.Close()

			p := New("test-key")
			p.httpClient = &http.Client{
				Transport: &mockTransport{
					roundTripFunc: func(req *http.Request) (*http.Response, error) {
						if req.Method == "POST" && strings.Contains(req.URL.Path, "cachedContents") {
							if !tt.expectCreateCall {
								return nil, fmt.Errorf("unexpected CreateCachedContent call")
							}
							respBody := `{"name": "cachedContents/new", "createTime": "2024-01-01T00:00:00Z"}`
							return &http.Response{
								StatusCode: 201,
								Body:       io.NopCloser(strings.NewReader(respBody)),
							}, nil
						}
						if req.Method == "DELETE" && strings.Contains(req.URL.Path, "cachedContents") {
							if !tt.expectDeleteCall {
								return nil, fmt.Errorf("unexpected DeleteCachedContent call")
							}
							return &http.Response{
								StatusCode: 204,
								Body:       io.NopCloser(strings.NewReader("")),
							}, nil
						}
						return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					},
				},
			}

			if tt.activeCacheName != "" {
				p.cacheMap = map[string]*cacheEntry{
					"gemini-1.5-pro": {
						name:         tt.activeCacheName,
						model:        "gemini-1.5-pro",
						messageCount: tt.cachedMsgCount,
						expireTime:   time.Now().Add(1 * time.Hour),
					},
				}
			}

			var receivedStorageCost float64
			p.SetUsageCallback(func(usage api.Usage) {
				receivedStorageCost = usage.StorageCost
			})

			os.Setenv("GEMINI_CONTEXT_CACHING", "1")
			defer os.Unsetenv("GEMINI_CONTEXT_CACHING")

			cacheName, msgs, err := p.updateOrUseCache(context.Background(), tt.systemPrompt, tt.history, "gemini-1.5-pro", nil, nil)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectCreateCall && receivedStorageCost <= 0 {
				t.Errorf("expected StorageCost > 0 when creating cache, got %v", receivedStorageCost)
			}

			if tt.expectCache {
				if cacheName == "" {
					t.Error("expected cache usage, but got empty cacheName")
				}
			} else {
				if cacheName != "" {
					t.Errorf("expected no cache usage, but got %s", cacheName)
				}
			}

			if tt.expectDiffOnly {
				if len(msgs) == len(tt.history) {
					t.Error("expected diff messages, but got full history")
				}
			} else {
				if len(msgs) != len(tt.history) {
					t.Errorf("expected full history (%d), but got %d", len(tt.history), len(msgs))
				}
			}
		})
	}
}

func TestUpdateOrUseCache_ReportsModelSpecificStorageCost(t *testing.T) {
	t.Setenv("GEMINI_CONTEXT_CACHING", "1")
	t.Setenv("GEMINI_CACHE_TTL", "3600")

	systemPrompt := "sys"
	history := []api.Message{
		{Role: "user", Content: cacheEligibleContent()},
		{Role: "user", Content: "last"},
	}
	tests := []struct {
		model           string
		wantStorageRate float64
		cfg             func() *config.Config
	}{
		{model: "gemini-3.1-flash-lite", wantStorageRate: 1.00},
		{model: "gemini-3.1-pro-preview", wantStorageRate: 4.50},
		{
			model:           "corp-gemini-flash-lite",
			wantStorageRate: 1.00,
			cfg: func() *config.Config {
				cfg := config.DefaultConfig()
				cfg.SetProviderModelConfig("gemini", config.ProviderModelConfig{
					DefaultModel: "corp-gemini-flash-lite",
					CatalogModel: "gemini-3.1-flash-lite",
				})
				return cfg
			},
		},
		{
			model:           "corp-gemini-pro",
			wantStorageRate: 4.50,
			cfg: func() *config.Config {
				cfg := config.DefaultConfig()
				cfg.SetProviderModelConfig("gemini", config.ProviderModelConfig{
					DefaultModel:   "corp-gemini-flash-lite",
					CatalogModel:   "gemini-3.1-flash-lite",
					ModelOverrides: map[string]config.ModelOverride{"corp-gemini-pro": {CatalogModel: "gemini-3.1-pro-preview"}},
				})
				return cfg
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			p := New("test-key")
			p.httpClient = &http.Client{
				Transport: &mockTransport{
					roundTripFunc: func(req *http.Request) (*http.Response, error) {
						if req.Method == "POST" && strings.Contains(req.URL.Path, "cachedContents") {
							return &http.Response{
								StatusCode: 201,
								Body:       io.NopCloser(strings.NewReader(`{"name": "cachedContents/new", "createTime": "2024-01-01T00:00:00Z"}`)),
							}, nil
						}
						return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					},
				},
			}

			var receivedStorageCost float64
			p.SetUsageCallback(func(usage api.Usage) {
				receivedStorageCost = usage.StorageCost
			})

			ctx := context.Background()
			if tt.cfg != nil {
				ctx = config.WithContext(ctx, tt.cfg())
			}

			cacheName, _, err := p.updateOrUseCache(ctx, systemPrompt, history, tt.model, nil, nil)
			if err != nil {
				t.Fatalf("updateOrUseCache() error = %v", err)
			}
			if cacheName == "" {
				t.Fatal("cacheName = empty, want created cache")
			}

			totalTokens := estimateTokens(tt.model, systemPrompt, history, nil)
			want := float64(totalTokens) / 1_000_000.0 * tt.wantStorageRate
			assertFloatApprox(t, receivedStorageCost, want)
		})
	}
}

func TestProviderClearCache_UsesInjectedRuntime(t *testing.T) {
	t.Setenv("XELYON_DEBUG_GEMINI", "1")

	var errBuf bytes.Buffer
	p := New("test-key")
	p.SetUIRuntime(ui.NewRuntime(nil, io.Discard, &errBuf))
	p.cacheMap = map[string]*cacheEntry{
		"gemini-1.5-pro": {
			name: "cachedContents/test-cache",
		},
	}
	p.httpClient = &http.Client{
		Transport: &mockTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			},
		},
	}

	p.ClearCache()

	if !strings.Contains(errBuf.String(), "[DEBUG Gemini] Clearing cache for gemini-1.5-pro: cachedContents/test-cache") {
		t.Fatalf("expected debug log in injected runtime, got %q", errBuf.String())
	}
}

// TestCacheMapModelIsolation はモデル別キャッシュが独立して管理されることを検証
func TestCacheMapModelIsolation(t *testing.T) {
	longContent := cacheEligibleContent()
	history := []api.Message{
		{Role: "user", Content: longContent},
		{Role: "user", Content: "last"},
	}

	createCount := 0
	p := New("test-key")
	p.httpClient = &http.Client{
		Transport: &mockTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				if req.Method == "POST" && strings.Contains(req.URL.Path, "cachedContents") {
					createCount++
					respBody := fmt.Sprintf(`{"name": "cachedContents/cache_%d", "createTime": "2024-01-01T00:00:00Z"}`, createCount)
					return &http.Response{
						StatusCode: 201,
						Body:       io.NopCloser(strings.NewReader(respBody)),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			},
		},
	}

	os.Setenv("GEMINI_CONTEXT_CACHING", "1")
	defer os.Unsetenv("GEMINI_CONTEXT_CACHING")

	ctx := context.Background()

	// モデルAでキャッシュ作成
	cacheA, _, err := p.updateOrUseCache(ctx, "sys", history, "model-a", nil, nil)
	if err != nil {
		t.Fatalf("model-a cache creation failed: %v", err)
	}
	if cacheA == "" {
		t.Fatal("expected cache for model-a, got empty")
	}

	// モデルBでキャッシュ作成（モデルAのキャッシュは残る）
	cacheB, _, err := p.updateOrUseCache(ctx, "sys", history, "model-b", nil, nil)
	if err != nil {
		t.Fatalf("model-b cache creation failed: %v", err)
	}
	if cacheB == "" {
		t.Fatal("expected cache for model-b, got empty")
	}

	// 両方のキャッシュが独立して存在
	if cacheA == cacheB {
		t.Errorf("model-a and model-b should have different cache names, both got %s", cacheA)
	}
	if len(p.cacheMap) != 2 {
		t.Errorf("expected 2 entries in cacheMap, got %d", len(p.cacheMap))
	}
	if p.cacheMap["model-a"] == nil || p.cacheMap["model-a"].name != cacheA {
		t.Errorf("cacheMap[model-a] mismatch: got %v", p.cacheMap["model-a"])
	}
	if p.cacheMap["model-b"] == nil || p.cacheMap["model-b"].name != cacheB {
		t.Errorf("cacheMap[model-b] mismatch: got %v", p.cacheMap["model-b"])
	}

	// モデルAで再度呼び出し → 既存キャッシュ再利用（差分のみ）
	moreHistory := append(history, api.Message{Role: "assistant", Content: "ok"}, api.Message{Role: "user", Content: "next"})
	cacheA2, msgs, err := p.updateOrUseCache(ctx, "sys", moreHistory, "model-a", nil, nil)
	if err != nil {
		t.Fatalf("model-a cache reuse failed: %v", err)
	}
	if cacheA2 != cacheA {
		t.Errorf("expected same cache name for model-a, got %s (want %s)", cacheA2, cacheA)
	}
	// 差分メッセージ = moreHistory[1:] (cachedMsgCount=1, history最後1つを除いてキャッシュ)
	if len(msgs) == len(moreHistory) {
		t.Error("expected diff messages for model-a reuse, but got full history")
	}

	// createCount は2（model-a, model-b の作成のみ、reuse時は増えない）
	if createCount != 2 {
		t.Errorf("expected 2 cache create calls, got %d", createCount)
	}
}

// TestInvalidateCacheForModel は特定モデルのキャッシュのみ無効化することを検証
func TestInvalidateCacheForModel(t *testing.T) {
	p := New("test-key")
	namespacedKey := cacheMapKey(api.WithProviderCacheNamespace(context.Background(), "repair"), "model-a")
	p.cacheMap = map[string]*cacheEntry{
		"model-a":     {name: "cachedContents/a", model: "model-a", messageCount: 5, expireTime: time.Now().Add(1 * time.Hour)},
		namespacedKey: {name: "cachedContents/a-repair", model: "model-a", messageCount: 2, expireTime: time.Now().Add(1 * time.Hour)},
		"model-b":     {name: "cachedContents/b", model: "model-b", messageCount: 3, expireTime: time.Now().Add(1 * time.Hour)},
	}

	// model-a のみ無効化
	p.invalidateCacheForModel("model-a")

	if _, ok := p.cacheMap["model-a"]; ok {
		t.Error("model-a should be removed from cacheMap")
	}
	if _, ok := p.cacheMap[namespacedKey]; ok {
		t.Error("namespaced model-a cache should be removed from cacheMap")
	}
	if _, ok := p.cacheMap["model-b"]; !ok {
		t.Error("model-b should still exist in cacheMap")
	}
}

func TestInvalidateCacheForRequestOnlyRemovesCurrentNamespace(t *testing.T) {
	p := New("test-key")
	repairCtx := api.WithProviderCacheNamespace(context.Background(), "repair")
	normalKey := "model-a"
	repairKey := cacheMapKey(repairCtx, "model-a")
	p.cacheMap = map[string]*cacheEntry{
		normalKey: {name: "cachedContents/a", model: "model-a", messageCount: 5, expireTime: time.Now().Add(1 * time.Hour)},
		repairKey: {name: "cachedContents/a-repair", model: "model-a", messageCount: 2, expireTime: time.Now().Add(1 * time.Hour)},
		"model-b": {name: "cachedContents/b", model: "model-b", messageCount: 3, expireTime: time.Now().Add(1 * time.Hour)},
	}

	p.invalidateCacheForRequest(repairCtx, "model-a")

	if _, ok := p.cacheMap[repairKey]; ok {
		t.Error("repair namespace cache should be removed from cacheMap")
	}
	if _, ok := p.cacheMap[normalKey]; !ok {
		t.Error("normal model-a cache should still exist in cacheMap")
	}
	if _, ok := p.cacheMap["model-b"]; !ok {
		t.Error("model-b should still exist in cacheMap")
	}
}

func TestUpdateOrUseCache_ProviderCacheNamespaceIsolatesCacheKey(t *testing.T) {
	t.Setenv("GEMINI_CONTEXT_CACHING", "1")

	longContent := cacheEligibleContent()
	model := "gemini-1.5-pro"
	history := []api.Message{
		{Role: "user", Content: longContent},
		{Role: "user", Content: "repair request"},
	}

	createCount := 0
	p := New("test-key")
	p.cacheMap = map[string]*cacheEntry{
		model: {
			name:         "cachedContents/main",
			model:        model,
			messageCount: 1,
			expireTime:   time.Now().Add(1 * time.Hour),
		},
	}
	p.httpClient = &http.Client{
		Transport: &mockTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				if req.Method == "POST" && strings.Contains(req.URL.Path, "cachedContents") {
					createCount++
					respBody := `{"name": "cachedContents/repair", "createTime": "2024-01-01T00:00:00Z"}`
					return &http.Response{
						StatusCode: 201,
						Body:       io.NopCloser(strings.NewReader(respBody)),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			},
		},
	}

	repairCtx := api.WithProviderCacheNamespace(context.Background(), "gemini_apply_patch_repair")
	repairCache, repairMsgs, err := p.updateOrUseCache(repairCtx, "repair sys", history, model, nil, nil)
	if err != nil {
		t.Fatalf("repair cache update failed: %v", err)
	}
	if repairCache != "cachedContents/repair" {
		t.Fatalf("repair cache = %q, want cachedContents/repair", repairCache)
	}
	if len(repairMsgs) != 1 || repairMsgs[0].Content != "repair request" {
		t.Fatalf("repair messages = %+v, want only last repair request", repairMsgs)
	}
	if p.cacheMap[model] == nil || p.cacheMap[model].name != "cachedContents/main" {
		t.Fatalf("normal cache entry was overwritten: %+v", p.cacheMap[model])
	}
	repairKey := cacheMapKey(repairCtx, model)
	if repairKey == model {
		t.Fatal("repair cache key should be distinct from normal model key")
	}
	if p.cacheMap[repairKey] == nil || p.cacheMap[repairKey].name != "cachedContents/repair" {
		t.Fatalf("repair cache entry missing: %+v", p.cacheMap[repairKey])
	}

	normalHistory := append(history, api.Message{Role: "assistant", Content: "ok"}, api.Message{Role: "user", Content: "next"})
	normalCache, _, err := p.updateOrUseCache(context.Background(), "main sys", normalHistory, model, nil, nil)
	if err != nil {
		t.Fatalf("normal cache reuse failed: %v", err)
	}
	if normalCache != "cachedContents/main" {
		t.Fatalf("normal cache = %q, want cachedContents/main", normalCache)
	}
	if createCount != 1 {
		t.Fatalf("cache create calls = %d, want only repair cache creation", createCount)
	}
}

// TestInvalidateCacheAll は全モデルのキャッシュが無効化されることを検証
func TestInvalidateCacheAll(t *testing.T) {
	p := New("test-key")
	p.cacheMap = map[string]*cacheEntry{
		"model-a": {name: "cachedContents/a", model: "model-a"},
		"model-b": {name: "cachedContents/b", model: "model-b"},
	}

	p.invalidateCache()

	if len(p.cacheMap) != 0 {
		t.Errorf("expected empty cacheMap after invalidateCache, got %d entries", len(p.cacheMap))
	}
}

// TestCacheMapDifferentModelNoReuse は異なるモデルのキャッシュが混在しないことを検証
func TestCacheMapDifferentModelNoReuse(t *testing.T) {
	longContent := cacheEligibleContent()
	history := []api.Message{
		{Role: "user", Content: longContent},
		{Role: "user", Content: "last"},
	}

	createCount := 0
	p := New("test-key")
	p.httpClient = &http.Client{
		Transport: &mockTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				if req.Method == "POST" && strings.Contains(req.URL.Path, "cachedContents") {
					createCount++
					respBody := fmt.Sprintf(`{"name": "cachedContents/cache_%d", "createTime": "2024-01-01T00:00:00Z"}`, createCount)
					return &http.Response{
						StatusCode: 201,
						Body:       io.NopCloser(strings.NewReader(respBody)),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			},
		},
	}

	os.Setenv("GEMINI_CONTEXT_CACHING", "1")
	defer os.Unsetenv("GEMINI_CONTEXT_CACHING")

	ctx := context.Background()

	// pro でキャッシュ作成
	_, _, err := p.updateOrUseCache(ctx, "sys", history, "gemini-3.1-pro-preview", nil, nil)
	if err != nil {
		t.Fatalf("pro cache creation failed: %v", err)
	}

	// customtools で呼び出し → pro のキャッシュは再利用されず、新規作成される
	cache2, _, err := p.updateOrUseCache(ctx, "sys", history, "gemini-3.1-pro-preview-customtools", nil, nil)
	if err != nil {
		t.Fatalf("customtools cache creation failed: %v", err)
	}
	if cache2 == "" {
		t.Fatal("expected cache for customtools model")
	}

	// 2回キャッシュ作成が呼ばれるはず（pro, customtools 各1回）
	if createCount != 2 {
		t.Errorf("expected 2 create calls (pro + customtools), got %d", createCount)
	}
}

func makeHistory(longContent string, count int) []api.Message {
	hist := make([]api.Message, count)
	hist[0] = api.Message{Role: "user", Content: longContent}
	for i := 1; i < count; i++ {
		hist[i] = api.Message{Role: "user", Content: "msg"}
	}
	return hist
}

func cacheEligibleContent() string {
	// 非 ASCII を使って 1 rune ~= 1 token の安全側推定を満たす。
	return strings.Repeat("日", minCacheTokens+100)
}

func assertFloatApprox(t *testing.T, got, want float64) {
	t.Helper()
	const tolerance = 0.000001
	if got < want-tolerance || got > want+tolerance {
		t.Fatalf("got %f, want %f", got, want)
	}
}

type mockTransport struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}
