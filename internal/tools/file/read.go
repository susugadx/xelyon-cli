package file

import (
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

// executeReadFileCore は outlineThreshold を指定可能な内部関数。
// outlineThreshold 行を超えるファイルはアウトラインモードで返す。
func executeReadFileCore(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, path string, startLine, endLine, outlineThreshold int) string {
	ctx, errResult := newReadFileContext(out, cfg, cache, path, outlineThreshold)
	if errResult != "" {
		return errResult
	}

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
