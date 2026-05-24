package tui

import "context"

type reviewRunIDContextKey struct{}

// ContextWithReviewRunID は /review progress message を active timeline run に紐付ける。
func ContextWithReviewRunID(ctx context.Context, id int) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, reviewRunIDContextKey{}, id)
}

// ReviewRunIDFromContext は ContextWithReviewRunID で埋め込んだ review run ID を返す。
func ReviewRunIDFromContext(ctx context.Context) (int, bool) {
	if ctx == nil {
		return 0, false
	}
	id, ok := ctx.Value(reviewRunIDContextKey{}).(int)
	return id, ok
}

func contextWithReviewRunID(ctx context.Context, id reviewRunID) context.Context {
	return ContextWithReviewRunID(ctx, int(id))
}
