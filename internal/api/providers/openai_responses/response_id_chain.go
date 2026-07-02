package openairesponses

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// ResponseIDChainPolicy は Responses API の previous_response_id 利用と次回 ID 保存の可否を表す。
type ResponseIDChainPolicy struct {
	ReusePrevious bool
	CacheNext     bool
}

// ResponseIDChainPolicyFromContext は request context から response ID chain policy を解決する。
func ResponseIDChainPolicyFromContext(ctx context.Context) ResponseIDChainPolicy {
	storeEnabled := config.FromContext(ctx).ResponsesStoreEnabled()
	activeContextAllowsChain := ResponseIDChainAllowed(ActiveContextFromContext(ctx))
	return ResponseIDChainPolicy{
		ReusePrevious: storeEnabled && activeContextAllowsChain && !api.ResponseIDChainDisabledFromContext(ctx),
		CacheNext:     storeEnabled && activeContextAllowsChain,
	}
}

// HasReusablePrevious は provider が保持する previous_response_id を今回使ってよいか返す。
func (p ResponseIDChainPolicy) HasReusablePrevious(hasCachedResponseID bool) bool {
	return p.ReusePrevious && hasCachedResponseID
}

// ShouldStoreNext は request 後に新しい response_id を chain として保存してよいか返す。
func (p ResponseIDChainPolicy) ShouldStoreNext(err error, responseID string) bool {
	return p.CacheNext && err == nil && responseID != ""
}

// ShouldClearStored は request 後に provider 側の保存済み response_id を消すべきか返す。
func (p ResponseIDChainPolicy) ShouldClearStored(err error, responseID string) bool {
	if !p.CacheNext {
		return true
	}
	if p.ShouldStoreNext(err, responseID) {
		return false
	}
	return !p.ReusePrevious
}

// PreviousResponseIDForRequestContext は request context と active context に従って previous_response_id を返す。
func PreviousResponseIDForRequestContext(ctx context.Context, previousResponseID string, activeContext []api.ActiveContextBlock) string {
	if !ResponseIDChainAllowedForRequest(ctx, activeContext) {
		return ""
	}
	return previousResponseID
}

// PreviousResponseIDForChatRequest は chat history の最新入力を previous_response_id で省略できる場合だけ ID を返す。
func PreviousResponseIDForChatRequest(ctx context.Context, previousResponseID string, activeContext []api.ActiveContextBlock, history []api.Message) string {
	previousResponseID = PreviousResponseIDForRequestContext(ctx, previousResponseID, activeContext)
	return PreviousResponseIDForChatHistory(previousResponseID, history)
}

// PreviousResponseIDForChatHistory は chat history の latest input が full payload を必要とする場合だけ ID を落とす。
func PreviousResponseIDForChatHistory(previousResponseID string, history []api.Message) string {
	if !ResponseIDChainAllowedForHistory(history) {
		return ""
	}
	return previousResponseID
}

// ResponseIDChainAllowedForHistory は latest message が full payload を必要としない場合だけ chain 利用を許可する。
func ResponseIDChainAllowedForHistory(history []api.Message) bool {
	if len(history) == 0 {
		return true
	}
	return !history[len(history)-1].HasImage()
}

// HasReusablePreviousForHistory は provider が保持する previous_response_id を今回の history で使ってよいか返す。
func (p ResponseIDChainPolicy) HasReusablePreviousForHistory(hasCachedResponseID bool, history []api.Message) bool {
	return p.HasReusablePrevious(hasCachedResponseID) && ResponseIDChainAllowedForHistory(history)
}

// PreviousResponseIDForActiveContext は active context を送る request では response-id chain を使わない。
func PreviousResponseIDForActiveContext(previousResponseID string, activeContext []api.ActiveContextBlock) string {
	if !ResponseIDChainAllowed(activeContext) {
		return ""
	}
	return previousResponseID
}

// ResponseIDChainAllowed は active context を含まない request だけ response ID chain の再利用を許可する。
func ResponseIDChainAllowed(activeContext []api.ActiveContextBlock) bool {
	return !HasActiveContext(activeContext)
}

// ResponseIDChainAllowedForContext は request context の active context から response-id chain 可否を返す。
func ResponseIDChainAllowedForContext(ctx context.Context) bool {
	return ResponseIDChainAllowedForRequest(ctx, ActiveContextFromContext(ctx))
}

// ResponseIDChainAllowedForRequest は request context と active context から response-id chain 可否を返す。
func ResponseIDChainAllowedForRequest(ctx context.Context, activeContext []api.ActiveContextBlock) bool {
	return !api.ResponseIDChainDisabledFromContext(ctx) && ResponseIDChainAllowed(activeContext)
}

// ResponseIDChainReusable は Responses storage と active context を合わせて response ID 再利用可否を返す。
func ResponseIDChainReusable(ctx context.Context) bool {
	return ResponseIDChainPolicyFromContext(ctx).ReusePrevious
}

// ResponseIDChainCacheable は request 後に新しい response ID を保存してよいか返す。
func ResponseIDChainCacheable(ctx context.Context) bool {
	return ResponseIDChainPolicyFromContext(ctx).CacheNext
}
