package file

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// DefaultFullLines is the threshold for switching from full content to outline mode.
// Most files should be returned in full so the model does not reread narrow ranges.
const DefaultFullLines = 2000

// MaxReadLines is the default maximum window for range reads when end_line is omitted.
const MaxReadLines = 1000

// LargeFileThreshold は「大容量ファイル」と判定するサイズ（1MB）
const LargeFileThreshold = 1024 * 1024

// ExecuteReadFile はファイルを読み込む（行範囲指定対応）。
// startLine, endLine が指定されている場合はその範囲のみ返す。
// 指定がない場合は小さいファイルは全文、大きいファイルはアウトラインを返す。
func ExecuteReadFile(path string, startLine, endLine int) string {
	return ExecuteReadFileWithOutput(common.DefaultOutput(), path, startLine, endLine)
}

// ExecuteReadFileWithOutput は出力先を指定してファイルを読み込む。
func ExecuteReadFileWithOutput(out common.Output, path string, startLine, endLine int) string {
	return ExecuteReadFileWithRuntime(out, config.DefaultConfig(), nil, path, startLine, endLine)
}

// ExecuteReadFileWithRuntime は runtime 設定を指定してファイルを読み込む。
func ExecuteReadFileWithRuntime(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, path string, startLine, endLine int) string {
	return executeReadFileCore(out, cfg, cache, path, startLine, endLine, DefaultFullLines)
}

func executeReadFileRequest(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, req readRequest, wholeFileThreshold int) string {
	req = normalizeReadRequestForExecution(req)

	if req.StartLine > 0 || req.EndLine > 0 {
		if result, handled := maybeExecuteCompactRangeRead(out, cfg, cache, req); handled {
			return result
		}
		return executeReadFileRequestCore(out, cfg, cache, req, DefaultFullLines)
	}

	if req.Detail == readDetailCompact {
		if canFallbackCompactWholeFile(req) {
			return executeCompactWholeFileFallback(out, cfg, cache, req, wholeFileThreshold)
		}
		return `Error: detail="compact" requires locator targets or explicit path ranges`
	}

	if req.Detail.wholeFileOverride() {
		return executeReadRequestWholeFileWithDetail(out, cfg, cache, req, req.Detail)
	}

	return executeReadFileRequestCore(out, cfg, cache, req, wholeFileThreshold)
}

func normalizeReadRequestForExecution(req readRequest) readRequest {
	if req.Detail == readDetailCompact && req.Source == readRequestSourcePathRange && req.StartLine > 0 && req.EndLine == 0 {
		req.EndLine = req.StartLine
	}
	return req
}

func canFallbackCompactWholeFile(req readRequest) bool {
	return req.Source == readRequestSourceLocator
}

func executeCompactWholeFileFallback(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, req readRequest, wholeFileThreshold int) string {
	ctx, errResult := newReadFileContextForRequest(out, cfg, cache, req, wholeFileThreshold)
	if errResult != "" {
		return errResult
	}
	if ctx.fileInfo != nil && ctx.fileInfo.Size() > LargeFileThreshold {
		return executeLargeFileOutline(ctx)
	}
	return executeReadFileRequestCore(out, cfg, cache, req, wholeFileThreshold)
}

func maybeExecuteCompactRangeRead(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, req readRequest) (string, bool) {
	if req.Detail != readDetailCompact {
		return "", false
	}

	ctx, errResult := newReadFileContextForRequest(out, cfg, cache, req, DefaultFullLines)
	if errResult != "" {
		return errResult, true
	}
	if cached, hit := getCachedReadContent(ctx); hit {
		return executeCompactRangeFromContent(ctx, cached, req.StartLine, req.EndLine), true
	}
	return executeStreamedReadRange(ctx, req.StartLine, req.EndLine), true
}

// executeReadFileCore は outlineThreshold を指定可能な内部関数。
// outlineThreshold 行を超えるファイルはアウトラインモードで返す。
func executeReadFileCore(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, path string, startLine, endLine, outlineThreshold int) string {
	ctx, errResult := newReadFileContext(out, cfg, cache, path, outlineThreshold)
	if errResult != "" {
		return errResult
	}
	return executeReadFileWithContext(ctx, startLine, endLine)
}

func executeReadFileRequestCore(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, req readRequest, outlineThreshold int) string {
	ctx, errResult := newReadFileContextForRequest(out, cfg, cache, req, outlineThreshold)
	if errResult != "" {
		return errResult
	}
	return executeReadFileWithContext(ctx, req.StartLine, req.EndLine)
}

func executeReadFileWithContext(ctx readFileContext, startLine, endLine int) string {
	if startLine == 0 && endLine == 0 {
		if result, handled := maybeReadLargeFile(ctx); handled {
			return result
		}
	}

	contentStr, errResult := loadReadContent(ctx, startLine, endLine)
	if errResult != "" {
		return errResult
	}
	if isBinaryContent(contentStr) {
		return binaryFileError(ctx.path)
	}

	return renderReadResult(ctx, contentStr, startLine, endLine)
}

func executeReadRequestWholeFileWithDetail(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, req readRequest, detail readDetailMode) string {
	ctx, errResult := newReadFileContextForRequest(out, cfg, cache, req, DefaultFullLines)
	if errResult != "" {
		return errResult
	}
	return executeReadWholeFileWithDetailContext(ctx, detail)
}

func executeReadWholeFileWithDetailContext(ctx readFileContext, detail readDetailMode) string {
	if detail == readDetailOutline {
		if ctx.fileInfo != nil && ctx.fileInfo.Size() > LargeFileThreshold {
			ctx.outlineThreshold = 0
			return executeLargeFileOutline(ctx)
		}

		contentStr, errResult := loadReadContent(ctx, 0, 0)
		if errResult != "" {
			return errResult
		}
		if isBinaryContent(contentStr) {
			return binaryFileError(ctx.path)
		}

		ctx.outlineThreshold = 0
		return renderReadResult(ctx, contentStr, 0, 0)
	}

	contentStr, errResult := loadReadContent(ctx, 0, 0)
	if errResult != "" {
		return errResult
	}
	if isBinaryContent(contentStr) {
		return binaryFileError(ctx.path)
	}

	switch detail {
	case readDetailFull:
		ctx.outlineThreshold = strings.Count(contentStr, "\n") + 2
	case readDetailOutline:
		ctx.outlineThreshold = 0
	}

	return renderReadResult(ctx, contentStr, 0, 0)
}
