package readtool

import (
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

const (
	compactFallbackBeforeLines = 5
	compactFallbackAfterLines  = 50
)

func prepareReadRequests(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, requests []readRequest) []readRequest {
	resolver := newCompactReadResolver(out, cfg, cache)
	prepared := make([]readRequest, 0, len(requests))
	seen := make(map[string]struct{}, len(requests))

	for _, req := range requests {
		expanded := resolver.expand(req)
		if shouldDedupePreparedReadRequest(expanded) {
			key := dedupeReadRequestKey(expanded)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
		}
		prepared = append(prepared, expanded)
	}

	return prepared
}

func shouldDedupePreparedReadRequest(req readRequest) bool {
	if req.Detail == readDetailCompact {
		return true
	}
	return req.Source == readRequestSourceLocator && req.Detail.wholeFileOverride() && req.StartLine == 0 && req.EndLine == 0
}

type compactReadResolver struct {
	out        common.Output
	cfg        *config.Config
	cache      tools.ToolCacheInterface
	blockCache map[string]compactBlockCacheEntry
}

type compactBlockCacheEntry struct {
	blocks []common.BlockRange
	ok     bool
}

func newCompactReadResolver(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface) *compactReadResolver {
	return &compactReadResolver{
		out:        out,
		cfg:        cfg,
		cache:      cache,
		blockCache: make(map[string]compactBlockCacheEntry),
	}
}

func (r *compactReadResolver) expand(req readRequest) readRequest {
	if req.Source != readRequestSourceLocator || req.Locator == nil {
		return req
	}

	switch {
	case req.Locator.Line <= 0:
		req.StartLine = 0
		req.EndLine = 0
		req.RangeEntry = req.FilePath
	case req.Detail.wholeFileOverride():
		req.StartLine = 0
		req.EndLine = 0
		req.RangeEntry = req.FilePath
	case locatorHasExplicitRange(req.Locator):
		req.StartLine = req.Locator.Line
		req.EndLine = req.Locator.EndLine
		req.RangeEntry = formatReadRangeEntry(req.FilePath, req.StartLine, req.EndLine)
	case req.Detail == readDetailCompact:
		req.StartLine, req.EndLine = r.resolveCompactLocatorRange(req)
		req.RangeEntry = formatReadRangeEntry(req.FilePath, req.StartLine, req.EndLine)
	default:
		req.StartLine, req.EndLine = defaultLocatorReadWindow(req.Locator.Line)
		req.RangeEntry = formatReadRangeEntry(req.FilePath, req.StartLine, req.EndLine)
	}

	return req
}

func locatorHasExplicitRange(loc *locator.Location) bool {
	return loc != nil && loc.Line > 0 && loc.EndLine >= loc.Line
}

func (r *compactReadResolver) resolveCompactLocatorRange(req readRequest) (int, int) {
	line := req.Locator.Line
	blocks, ok := r.blockMap(req)
	if !ok {
		return defaultLocatorReadWindow(line)
	}
	if block, ok := smallestEnclosingBlock(blocks, line); ok {
		return block.StartLine, block.EndLine
	}

	return defaultLocatorReadWindow(line)
}

func (r *compactReadResolver) blockMap(req readRequest) ([]common.BlockRange, bool) {
	cacheKey := req.readPath()
	if cached, ok := r.blockCache[cacheKey]; ok {
		return cached.blocks, cached.ok
	}

	ctx, errResult := newReadFileContextForRequest(r.out, r.cfg, r.cache, req, DefaultFullLines)
	if errResult != "" {
		r.blockCache[cacheKey] = compactBlockCacheEntry{ok: false}
		return nil, false
	}
	if !shouldResolveCompactLocatorByBlock(ctx) {
		r.blockCache[cacheKey] = compactBlockCacheEntry{ok: false}
		return nil, false
	}

	contentStr, errResult := loadReadContent(ctx, 0, 0)
	if errResult != "" || isBinaryContent(contentStr) {
		r.blockCache[cacheKey] = compactBlockCacheEntry{ok: false}
		return nil, false
	}

	blocks := common.BuildBlockMap(contentStr, common.IsBraceLanguage(req.FilePath))
	r.blockCache[cacheKey] = compactBlockCacheEntry{
		blocks: blocks,
		ok:     true,
	}
	return blocks, true
}

func shouldResolveCompactLocatorByBlock(ctx readFileContext) bool {
	return ctx.fileInfo == nil || ctx.fileInfo.Size() <= LargeFileThreshold
}

func smallestEnclosingBlock(blocks []common.BlockRange, line int) (common.BlockRange, bool) {
	var best common.BlockRange
	found := false
	bestSpan := 0
	for _, block := range blocks {
		if line < block.StartLine || line > block.EndLine {
			continue
		}
		span := block.EndLine - block.StartLine
		if !found || span < bestSpan {
			best = block
			bestSpan = span
			found = true
		}
	}
	return best, found
}

func defaultLocatorReadWindow(line int) (int, int) {
	return max(1, line-compactFallbackBeforeLines), line + compactFallbackAfterLines
}
