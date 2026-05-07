package api

import "context"

type toolUseDisabledContextKey struct{}

// WithToolUseDisabled は request 単位で provider の tool/function payload を送らない mode にする。
func WithToolUseDisabled(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, toolUseDisabledContextKey{}, true)
}

// IsToolUseDisabled は request payload から tools/tool_choice/tool_config を省くべきかを返す。
func IsToolUseDisabled(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	disabled, _ := ctx.Value(toolUseDisabledContextKey{}).(bool)
	return disabled
}

// ShouldSendToolPayload は provider 側の function calling 有効判定と request mode を合成する。
func ShouldSendToolPayload(ctx context.Context, functionCallingEnabled bool) bool {
	return functionCallingEnabled && !IsToolUseDisabled(ctx)
}
