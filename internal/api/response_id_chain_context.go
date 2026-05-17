package api

import "context"

type responseIDChainDisabledContextKey struct{}

// WithResponseIDChainDisabled は request context で previous_response_id chain の再利用を禁止する。
func WithResponseIDChainDisabled(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, responseIDChainDisabledContextKey{}, true)
}

// ResponseIDChainDisabledFromContext は request context で response ID chain 再利用が禁止されているか返す。
func ResponseIDChainDisabledFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	disabled, ok := ctx.Value(responseIDChainDisabledContextKey{}).(bool)
	return ok && disabled
}
