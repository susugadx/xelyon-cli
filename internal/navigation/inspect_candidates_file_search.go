package navigation

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

type goFileSearchPlan struct {
	DirectFile string
	SearchPath string
}

// listGoFiles はプロジェクト内の Go ファイルを一覧する。
// pathHint が指定されている場合はそのパス配下に限定する。
func listGoFiles(pathHint string) []string {
	if !common.IsRipgrepAvailable() {
		return nil
	}

	plan, ok := planGoFileSearch(pathHint)
	if !ok {
		return nil
	}
	if plan.DirectFile != "" {
		return []string{plan.DirectFile}
	}

	output, ok := runGoFileSearch(plan.SearchPath)
	if !ok {
		return nil
	}
	return parseGoFileSearchOutput(output)
}

func planGoFileSearch(pathHint string) (goFileSearchPlan, bool) {
	searchPath := "."
	if pathHint != "" {
		// pathHint がファイルの場合はそのファイルだけ。
		if info, err := os.Stat(pathHint); err == nil && !info.IsDir() {
			if strings.HasSuffix(pathHint, ".go") {
				absPath, err := filepath.Abs(pathHint)
				if err != nil {
					return goFileSearchPlan{}, false
				}
				return goFileSearchPlan{DirectFile: absPath}, true
			}
			return goFileSearchPlan{}, false
		}
		searchPath = pathHint
	}
	return goFileSearchPlan{SearchPath: searchPath}, true
}
