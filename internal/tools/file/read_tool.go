package file

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ReadFileTool は1回の呼び出しで1個以上のファイルを読み込むツール。
type ReadFileTool struct{}

func (t *ReadFileTool) Name() string { return "read_file" }

func (t *ReadFileTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

func (t *ReadFileTool) Parameters() map[string]interface{} {
	return schemaRequiredPathsParameters(MaxReadFilesPaths)
}

func (t *ReadFileTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	paths, reg, result, err := resolveReadTargets(execCtx, args["targets"], args["paths"])
	if result != "" || err != nil {
		return result, nil, err
	}

	budgetOverride := 0
	if strings.EqualFold(args["_full_budget"], "true") {
		budgetOverride = DefaultFullLines
	}

	return ExecuteReadFilesWithLocator(
		execCtx.Output(),
		execCtx.EffectiveConfig(),
		execCtx.EffectiveToolCache(),
		paths,
		budgetOverride,
		reg,
	), nil, nil
}

func resolveReadTargets(execCtx tools.ExecutionContext, rawTargets, rawPaths string) ([]string, *locator.Registry, string, error) {
	var paths []string

	if rawTargets != "" {
		reg := execCtx.EffectiveLocatorRegistry()
		locs := reg.ResolveMulti(rawTargets)
		if len(locs) == 0 {
			return nil, nil, fmt.Sprintf("Error: no valid locator IDs found in targets: %s", rawTargets), nil
		}
		for _, loc := range locs {
			switch {
			case loc.EndLine > 0 && loc.Line > 0:
				paths = append(paths, fmt.Sprintf("%s:%d-%d", loc.FilePath, loc.Line, loc.EndLine))
			case loc.Line > 0:
				paths = append(paths, fmt.Sprintf("%s:%d-%d", loc.FilePath, max(1, loc.Line-5), loc.Line+50))
			default:
				paths = append(paths, loc.FilePath)
			}
		}
		return paths, nil, "", nil
	}

	if rawPaths != "" {
		if err := json.Unmarshal([]byte(rawPaths), &paths); err != nil {
			return nil, nil, fmt.Sprintf("Error: invalid paths format: %v", err), nil
		}
	}

	if len(paths) == 0 {
		return nil, nil, "Error: either paths or targets is required", nil
	}

	return paths, execCtx.EffectiveLocatorRegistry(), "", nil
}
