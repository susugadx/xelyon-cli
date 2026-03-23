package file

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// ReadFileTool は1回の呼び出しで1個以上のファイルを読み込むツール。
type ReadFileTool struct{}

func (t *ReadFileTool) Name() string { return "read_file" }

func (t *ReadFileTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

func (t *ReadFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"paths": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"maxItems":    MaxReadFilesPaths,
				"description": "Files to read (1-10). Returns full content. Do not re-read files already returned.",
			},
			"targets": map[string]interface{}{
				"type":        "string",
				"description": "Locator IDs to read (e.g. '[L1,L5]'). Alternative to paths.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *ReadFileTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	out := execCtx.Output()

	rawTargets := args["targets"]
	rawPaths := args["paths"]

	var paths []string

	// targets が指定されている場合はLocator IDから解決
	if rawTargets != "" {
		reg := execCtx.EffectiveLocatorRegistry()
		locs := reg.ResolveMulti(rawTargets)
		if len(locs) == 0 {
			return fmt.Sprintf("Error: no valid locator IDs found in targets: %s", rawTargets), nil, nil
		}
		for _, loc := range locs {
			if loc.EndLine > 0 && loc.Line > 0 {
				// 行範囲付き: "path:startLine-endLine" 形式に変換
				paths = append(paths, fmt.Sprintf("%s:%d-%d", loc.FilePath, loc.Line, loc.EndLine))
			} else if loc.Line > 0 {
				// 行番号のみ: シンボル周辺のコンテキストを付与
				paths = append(paths, fmt.Sprintf("%s:%d-%d", loc.FilePath, max(1, loc.Line-5), loc.Line+50))
			} else {
				// ファイル全体
				paths = append(paths, loc.FilePath)
			}
		}
	} else if rawPaths != "" {
		if err := json.Unmarshal([]byte(rawPaths), &paths); err != nil {
			return fmt.Sprintf("Error: invalid paths format: %v", err), nil, nil
		}
	}

	if len(paths) == 0 {
		return "Error: either paths or targets is required", nil, nil
	}

	budgetOverride := 0
	if strings.EqualFold(args["_full_budget"], "true") {
		budgetOverride = DefaultFullLines
	}

	// targets 経由の場合は ID が既に Registry に登録済みなので再登録しない
	var reg *locator.Registry
	if rawTargets == "" {
		reg = execCtx.EffectiveLocatorRegistry()
	}

	return ExecuteReadFilesWithLocator(
		out,
		execCtx.EffectiveConfig(),
		execCtx.EffectiveToolCache(),
		paths,
		budgetOverride,
		reg,
	), nil, nil
}

// WriteFileTool wraps write_file execution
type WriteFileTool struct{}

func (t *WriteFileTool) Name() string { return "write_file" }

func (t *WriteFileTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

func (t *WriteFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path":    map[string]interface{}{"type": "string", "description": "File path to write to"},
			"content": map[string]interface{}{"type": "string", "description": "Content to write to the file"},
		},
		"required":             []string{"path", "content"},
		"additionalProperties": false,
	}
}

func (t *WriteFileTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	result, err := ExecuteWriteFileWithPromptIOAndOptionsAndLSPClient(execCtx.PromptIO(), execCtx.ConfirmOptions(), execCtx.EffectiveLSPClient(), args["path"], args["content"])
	if err != nil {
		return result, nil, err
	}
	return result, &tools.FileChange{
		FilePath:    args["path"],
		Timestamp:   common.GetCurrentTime(),
		Tool:        "write_file",
		Description: "Wrote file " + args["path"],
		LinesAdded:  countLines(args["content"]),
	}, nil
}

// StrReplaceTool wraps str_replace execution
type StrReplaceTool struct{}

func (t *StrReplaceTool) Name() string { return "str_replace" }

func (t *StrReplaceTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

func (t *StrReplaceTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path":       map[string]interface{}{"type": "string", "description": "File path to edit"},
			"old_str":    map[string]interface{}{"type": "string", "description": "Exact string to find and replace"},
			"new_str":    map[string]interface{}{"type": "string", "description": "New string to replace with"},
			"start_line": map[string]interface{}{"type": "string", "description": "Start line number to limit search scope (optional)"},
			"end_line":   map[string]interface{}{"type": "string", "description": "End line number to limit search scope (optional)"},
			"edits": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"old_str": map[string]interface{}{"type": "string"},
						"new_str": map[string]interface{}{"type": "string"},
					},
					"required": []string{"old_str", "new_str"},
				},
				"description": "Batch edits: array of {old_str, new_str} pairs applied sequentially",
			},
		},
		"required":             []string{"path"},
		"additionalProperties": false,
	}
}

func (t *StrReplaceTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	// batch edits モード: old_str 空 + edits 非空
	if args["old_str"] == "" && args["edits"] != "" {
		result, err := executeBatchEditsWithPromptIOAndOptions(execCtx.PromptIO(), execCtx.ConfirmOptions(), args["path"], args["edits"])
		if err != nil {
			return result, nil, err
		}
		if !shouldRecordStrReplaceChange(result) {
			return result, nil, nil
		}
		linesAdded, linesRemoved := 0, 0
		var edits []EditEntry
		if json.Unmarshal([]byte(args["edits"]), &edits) == nil {
			for _, e := range edits {
				linesAdded += countLines(e.NewStr)
				linesRemoved += countLines(e.OldStr)
			}
		}
		return result, &tools.FileChange{
			FilePath:     args["path"],
			Timestamp:    common.GetCurrentTime(),
			Tool:         "str_replace",
			Description:  "Batch replaced in " + args["path"],
			LinesAdded:   linesAdded,
			LinesRemoved: linesRemoved,
		}, nil
	}

	// 従来のシングル編集 or 行レンジ
	result, err := ExecuteStrReplaceWithPromptIOAndOptions(execCtx.PromptIO(), execCtx.ConfirmOptions(), args["path"], args["old_str"], args["new_str"], args["start_line"], args["end_line"])
	if err != nil {
		return result, nil, err
	}
	if !shouldRecordStrReplaceChange(result) {
		return result, nil, nil
	}
	return result, &tools.FileChange{
		FilePath:     args["path"],
		Timestamp:    common.GetCurrentTime(),
		Tool:         "str_replace",
		Description:  "Replaced in " + args["path"],
		LinesAdded:   countLines(args["new_str"]),
		LinesRemoved: countLines(args["old_str"]),
	}, nil
}

func shouldRecordStrReplaceChange(result string) bool {
	return result != "" &&
		!strings.HasPrefix(result, "Error:") &&
		!strings.HasPrefix(result, "[CANCELLED]") &&
		!strings.HasPrefix(result, "[COMMENT]") &&
		result != "No changes after applying all edits"
}

// DeleteFileTool wraps delete_file execution
type DeleteFileTool struct{}

func (t *DeleteFileTool) Name() string { return "delete_file" }

func (t *DeleteFileTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

func (t *DeleteFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{"type": "string", "description": "File path to delete"},
		},
		"required":             []string{"path"},
		"additionalProperties": false,
	}
}

func (t *DeleteFileTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	result, err := ExecuteDeleteFileWithPromptIOAndOptionsAndLSPClient(execCtx.PromptIO(), execCtx.ConfirmOptions(), execCtx.EffectiveLSPClient(), args["path"])
	if err != nil {
		return result, nil, err
	}
	return result, &tools.FileChange{
		FilePath:    args["path"],
		Timestamp:   common.GetCurrentTime(),
		Tool:        "delete_file",
		Description: "Deleted file " + args["path"],
	}, nil
}

// ListDirTool wraps list_dir execution
type ListDirTool struct{}

func (t *ListDirTool) Name() string { return "list_dir" }

func (t *ListDirTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

func (t *ListDirTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path":  map[string]interface{}{"type": "string", "description": "Directory path to list"},
			"depth": map[string]interface{}{"type": "integer", "description": "Recursion depth (default: 1, max: 3)"},
		},
		"required":             []string{"path"},
		"additionalProperties": false,
	}
}

func (t *ListDirTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	depth := 1
	if args["depth"] != "" {
		if n, err := strconv.Atoi(args["depth"]); err == nil {
			depth = n
		}
	}
	return ExecuteListDirWithRuntime(execCtx.EffectiveConfig(), execCtx.EffectiveToolCache(), args["path"], depth), nil, nil
}

// RegisterTools registers all file tools to the given registry
func RegisterTools(r *tools.Registry) {
	r.Register(&ReadFileTool{})
	r.Register(&WriteFileTool{})
	r.Register(&StrReplaceTool{})
	r.Register(&DeleteFileTool{})
	r.Register(&ListDirTool{})
}

// init registers all file tools to the default registry
func init() {
	RegisterTools(tools.DefaultRegistry)
}
