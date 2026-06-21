package readtool

import (
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// MaxReadFilesPaths は一度に読み込めるファイル数の上限
const MaxReadFilesPaths = 10

// ExecuteReadFiles は複数ファイルを一括読み込みする
func ExecuteReadFiles(paths []string) string {
	return ExecuteReadFilesWithOutput(common.DefaultOutput(), paths)
}

// ExecuteReadFilesWithOutput は出力先を指定して複数ファイルを一括読み込みする。
func ExecuteReadFilesWithOutput(out common.Output, paths []string) string {
	return ExecuteReadFilesWithBudget(out, paths, 0)
}

// ExecuteReadFilesWithRuntime は config/cache を指定して複数ファイルを一括読み込みする。
// 自動 batch merge で使用し、単発 read_file と同じ runtime 文脈（config 反映、cache 反映）を維持する。
func ExecuteReadFilesWithRuntime(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, paths []string, budgetOverride int) string {
	return executeReadFilesCore(out, cfg, cache, paths, budgetOverride, nil)
}

// ExecuteReadFilesWithLocator は Locator Registry 付きで複数ファイルを一括読み込みする。
// reg が nil でない場合、ファイルヘッダーに Locator ID を付与する。
func ExecuteReadFilesWithLocator(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, paths []string, budgetOverride int, reg *locator.Registry) string {
	return executeReadFilesCore(out, cfg, cache, paths, budgetOverride, reg)
}

// ExecuteReadFilesWithBudget は出力先とファイルあたりのアウトライン閾値を指定して
// 複数ファイルを一括読み込みする。budgetOverride が 0 の場合は DefaultFullLines を使用する。
// 自動 batch merge では DefaultFullLines を渡し、単発 read と同等の閾値を維持する。
func ExecuteReadFilesWithBudget(out common.Output, paths []string, budgetOverride int) string {
	return executeReadFilesCore(out, nil, nil, paths, budgetOverride, nil)
}

// executeReadFilesCore は複数ファイル読み込みの内部実装。
// cfg/cache が nil の場合は単発 read_file のフォールバック動作（設定なし）を使用する。
// reg が nil でない場合、ファイルヘッダーに Locator ID を付与する。
func executeReadFilesCore(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, paths []string, budgetOverride int, reg *locator.Registry) string {
	requests := buildReadRequestsFromPaths(paths, readDetailAuto)
	return executeReadFilesRequestsCore(out, cfg, cache, requests, budgetOverride, reg)
}

func executeReadFilesRequestsCore(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, requests []readRequest, budgetOverride int, reg *locator.Registry) string {
	return renderReadExecutionSections(executeReadFilesRequestsSections(out, cfg, cache, requests, budgetOverride, reg))
}
