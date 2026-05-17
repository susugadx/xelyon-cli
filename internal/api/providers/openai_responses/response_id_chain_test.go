package openairesponses

import (
	"context"
	"errors"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestResponseIDChainReusable_RequiresStoreAndNoActiveContext(t *testing.T) {
	cfg := config.DefaultConfig()
	ctx := config.WithContext(context.Background(), cfg)
	if !ResponseIDChainReusable(ctx) {
		t.Fatal("ResponseIDChainReusable() = false, want true when responses.store is enabled without active context")
	}
	if !ResponseIDChainCacheable(ctx) {
		t.Fatal("ResponseIDChainCacheable() = false, want true when responses.store is enabled without active context")
	}

	withActiveContext := api.WithActiveContextBlocks(ctx, activeContextBlocks(responsesTestActiveContextSnapshot))
	if ResponseIDChainReusable(withActiveContext) {
		t.Fatal("ResponseIDChainReusable() = true, want false when active context is present")
	}
	if ResponseIDChainCacheable(withActiveContext) {
		t.Fatal("ResponseIDChainCacheable() = true, want false when active context is present")
	}

	withDisabledChain := api.WithResponseIDChainDisabled(ctx)
	if ResponseIDChainReusable(withDisabledChain) {
		t.Fatal("ResponseIDChainReusable() = true, want false when request disables response ID chain")
	}
	if !ResponseIDChainCacheable(withDisabledChain) {
		t.Fatal("ResponseIDChainCacheable() = false, want true so the full-history response can start a new chain")
	}

	cfg.Responses.Store = false
	storeDisabled := config.WithContext(context.Background(), cfg)
	if ResponseIDChainReusable(storeDisabled) {
		t.Fatal("ResponseIDChainReusable() = true, want false when responses.store is disabled")
	}
	if ResponseIDChainCacheable(storeDisabled) {
		t.Fatal("ResponseIDChainCacheable() = true, want false when responses.store is disabled")
	}
}

func TestResponseIDChainPolicy_PostRequestState(t *testing.T) {
	errFailed := errors.New("request failed")
	tests := []struct {
		name         string
		policy       ResponseIDChainPolicy
		err          error
		responseID   string
		wantStore    bool
		wantClear    bool
		wantReusable bool
	}{
		{
			name:         "reuse and cache stores successful response",
			policy:       ResponseIDChainPolicy{ReusePrevious: true, CacheNext: true},
			responseID:   "resp_new",
			wantStore:    true,
			wantClear:    false,
			wantReusable: true,
		},
		{
			name:         "reuse and cache preserves old chain when response omits id",
			policy:       ResponseIDChainPolicy{ReusePrevious: true, CacheNext: true},
			wantStore:    false,
			wantClear:    false,
			wantReusable: true,
		},
		{
			name:       "full history reset stores new chain",
			policy:     ResponseIDChainPolicy{ReusePrevious: false, CacheNext: true},
			responseID: "resp_new",
			wantStore:  true,
			wantClear:  false,
		},
		{
			name:      "full history reset clears old chain when no new id",
			policy:    ResponseIDChainPolicy{ReusePrevious: false, CacheNext: true},
			wantStore: false,
			wantClear: true,
		},
		{
			name:       "active context clears even if response returns id",
			policy:     ResponseIDChainPolicy{ReusePrevious: false, CacheNext: false},
			responseID: "resp_new",
			wantClear:  true,
		},
		{
			name:       "failed full history reset clears old chain",
			policy:     ResponseIDChainPolicy{ReusePrevious: false, CacheNext: true},
			err:        errFailed,
			responseID: "resp_new",
			wantClear:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.policy.ShouldStoreNext(tt.err, tt.responseID); got != tt.wantStore {
				t.Fatalf("ShouldStoreNext() = %t, want %t", got, tt.wantStore)
			}
			if got := tt.policy.ShouldClearStored(tt.err, tt.responseID); got != tt.wantClear {
				t.Fatalf("ShouldClearStored() = %t, want %t", got, tt.wantClear)
			}
			if got := tt.policy.HasReusablePrevious(true); got != tt.wantReusable {
				t.Fatalf("HasReusablePrevious(true) = %t, want %t", got, tt.wantReusable)
			}
		})
	}
}
