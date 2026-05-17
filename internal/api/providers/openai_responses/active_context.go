package openairesponses

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// ActiveContextFromContext は Responses request に送る active context block を返す。
func ActiveContextFromContext(ctx context.Context) []api.ActiveContextBlock {
	return api.ActiveContextBlocksFromContext(ctx)
}
