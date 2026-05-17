package api

import "context"

type activeContextBlocksContextKey struct{}

// ActiveContextBlock は provider request に一時的に添える動的文脈を表す。
type ActiveContextBlock struct {
	Name    string
	Content string
}

// WithActiveContextBlocks は provider-neutral な active context block を request context に注入する。
func WithActiveContextBlocks(ctx context.Context, blocks []ActiveContextBlock) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(blocks) == 0 {
		return ctx
	}
	return context.WithValue(ctx, activeContextBlocksContextKey{}, cloneActiveContextBlocks(blocks))
}

// WithoutActiveContextBlocks は親 context から継承した active context block を遮断する。
func WithoutActiveContextBlocks(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, activeContextBlocksContextKey{}, []ActiveContextBlock(nil))
}

// ActiveContextBlocksFromContext は request context に注入された active context block を返す。
func ActiveContextBlocksFromContext(ctx context.Context) []ActiveContextBlock {
	if ctx == nil {
		return nil
	}
	blocks, ok := ctx.Value(activeContextBlocksContextKey{}).([]ActiveContextBlock)
	if !ok {
		return nil
	}
	return cloneActiveContextBlocks(blocks)
}

func cloneActiveContextBlocks(blocks []ActiveContextBlock) []ActiveContextBlock {
	if len(blocks) == 0 {
		return nil
	}
	cloned := make([]ActiveContextBlock, len(blocks))
	copy(cloned, blocks)
	return cloned
}
