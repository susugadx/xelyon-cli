package websearch

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func resetRegistry(t *testing.T) {
	t.Helper()
	registryMu.Lock()
	defer registryMu.Unlock()
	// テスト前にレジストリをクリア
	old := registry
	registry = make(map[string]SearchFuncWithContext)
	t.Cleanup(func() {
		registryMu.Lock()
		defer registryMu.Unlock()
		registry = old
	})
}

func TestRegister_BasicSearch(t *testing.T) {
	resetRegistry(t)
	called := false
	Register("test-provider", func(query, model string) (string, error) {
		called = true
		return "result:" + query, nil
	})

	result, err := SearchWithContext(context.Background(), "test-provider", "hello", "model1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("search function was not called")
	}
	if result != "result:hello" {
		t.Errorf("got %q, want %q", result, "result:hello")
	}
}

func TestRegisterWithContext_BasicSearch(t *testing.T) {
	resetRegistry(t)
	RegisterWithContext("ctx-provider", func(ctx context.Context, query, model string) (string, error) {
		return fmt.Sprintf("ctx:%s:%s", query, model), nil
	})

	result, err := SearchWithContext(context.Background(), "ctx-provider", "q", "m")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ctx:q:m" {
		t.Errorf("got %q, want %q", result, "ctx:q:m")
	}
}

func TestUsageCallbackContext(t *testing.T) {
	var got api.Usage
	ctx := WithUsageCallback(context.Background(), func(usage api.Usage) {
		got = usage
	})

	callback := UsageCallbackFromContext(ctx)
	if callback == nil {
		t.Fatal("UsageCallbackFromContext() = nil, want callback")
	}
	callback(api.Usage{InputTokens: 12, OutputTokens: 3, CachedInputTokens: 4})
	if got.InputTokens != 12 || got.OutputTokens != 3 || got.CachedInputTokens != 4 {
		t.Fatalf("usage = %+v, want callback payload", got)
	}
	if UsageCallbackFromContext(context.Background()) != nil {
		t.Fatal("UsageCallbackFromContext(background) = non-nil, want nil")
	}
}

func TestRegister_NilFunction(t *testing.T) {
	resetRegistry(t)
	Register("nil-provider", nil)

	_, err := SearchWithContext(context.Background(), "nil-provider", "q", "m")
	if err == nil {
		t.Fatal("expected error for nil-registered provider")
	}
}

func TestRegisterWithContext_NilFunction(t *testing.T) {
	resetRegistry(t)
	RegisterWithContext("nil-ctx", nil)

	_, err := SearchWithContext(context.Background(), "nil-ctx", "q", "m")
	if err == nil {
		t.Fatal("expected error for nil-registered provider")
	}
}

func TestSearchWithContext_UnregisteredProvider(t *testing.T) {
	resetRegistry(t)
	_, err := SearchWithContext(context.Background(), "unknown", "q", "m")
	if err == nil {
		t.Fatal("expected error for unregistered provider")
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Errorf("error should mention 'not registered', got: %v", err)
	}
}

func TestRegister_CaseInsensitive(t *testing.T) {
	resetRegistry(t)
	Register("MyProvider", func(query, model string) (string, error) {
		return "ok", nil
	})

	// 小文字で検索
	result, err := SearchWithContext(context.Background(), "myprovider", "q", "m")
	if err != nil {
		t.Fatalf("lowercase lookup failed: %v", err)
	}
	if result != "ok" {
		t.Errorf("got %q, want %q", result, "ok")
	}

	// 大文字で検索
	result, err = SearchWithContext(context.Background(), "MYPROVIDER", "q", "m")
	if err != nil {
		t.Fatalf("uppercase lookup failed: %v", err)
	}
	if result != "ok" {
		t.Errorf("got %q, want %q", result, "ok")
	}
}

func TestRegister_TrimWhitespace(t *testing.T) {
	resetRegistry(t)
	Register("  spaced  ", func(query, model string) (string, error) {
		return "trimmed", nil
	})

	result, err := SearchWithContext(context.Background(), "spaced", "q", "m")
	if err != nil {
		t.Fatalf("trimmed lookup failed: %v", err)
	}
	if result != "trimmed" {
		t.Errorf("got %q, want %q", result, "trimmed")
	}
}

func TestRegister_OverwritesPrevious(t *testing.T) {
	resetRegistry(t)
	Register("dup", func(query, model string) (string, error) {
		return "first", nil
	})
	Register("dup", func(query, model string) (string, error) {
		return "second", nil
	})

	result, err := SearchWithContext(context.Background(), "dup", "q", "m")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "second" {
		t.Errorf("got %q, want %q (overwrite failed)", result, "second")
	}
}

func TestSearchWithContext_PropagatesError(t *testing.T) {
	resetRegistry(t)
	Register("err-provider", func(query, model string) (string, error) {
		return "", fmt.Errorf("api error")
	})

	_, err := SearchWithContext(context.Background(), "err-provider", "q", "m")
	if err == nil {
		t.Fatal("expected error from search function")
	}
	if !strings.Contains(err.Error(), "api error") {
		t.Errorf("error should contain 'api error', got: %v", err)
	}
}
