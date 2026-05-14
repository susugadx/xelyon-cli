package openairesponses

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// ActiveContextFromContext は Responses request に送る active context block を返す。
func ActiveContextFromContext(ctx context.Context) []api.ActiveContextBlock {
	return api.ActiveContextBlocksFromContext(ctx)
}

// ResponseIDChainAllowedForContext は request context の active context から response-id chain 可否を返す。
func ResponseIDChainAllowedForContext(ctx context.Context) bool {
	return ResponseIDChainAllowed(ActiveContextFromContext(ctx))
}

// ResponseIDChainReusable は Responses storage と active context を合わせて response ID 再利用可否を返す。
func ResponseIDChainReusable(ctx context.Context) bool {
	return config.FromContext(ctx).ResponsesStoreEnabled() && ResponseIDChainAllowedForContext(ctx)
}
