package file

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// ReadFileTool needs special handling for optional line range
type ReadFileTool struct{}

func (t *ReadFileTool) Name() string { return "read_file" }

func (t *ReadFileTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

func (t *ReadFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path":       map[string]interface{}{"type": "string", "description": "Absolute or relative file path to read. Omitting start_line/end_line reads up to the first 300 lines."},
			"start_line": map[string]interface{}{"type": "integer", "description": "Start line number (1-indexed). Use only when the target section is already known."},
			"end_line":   map[string]interface{}{"type": "integer", "description": "End line number (1-indexed). Omit to read up to 300 lines from start_line."},
			"symbol":     map[string]interface{}{"type": "string", "description": "Read specific symbol(s) by name. Comma-separated for multiple (e.g. \"Build,HandleRequest\"). Returns only the matching function/type body with surrounding context. Go files only (Phase 1). When symbol is provided, start_line/end_line are ignored."},
			"paths": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"maxItems":    MaxReadFilesPaths,
				"description": "Read multiple files in one call (max 10). Format: \"path\" or \"path:start-end\". Prefer plain \"path\" when investigating; add ranges only for known sections. When paths is provided, path/start_line/end_line are ignored.",
			},
		},
		"required":             []string{},
		"additionalProperties": false,
	}
}

func (t *ReadFileTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	out := execCtx.Output()

	// バッチモード: paths が指定されている場合
	if args["paths"] != "" {
		var paths []string
		if err := json.Unmarshal([]byte(args["paths"]), &paths); err != nil {
			return fmt.Sprintf("Error: invalid paths format: %v", err), nil, nil
		}
		return ExecuteReadFilesWithOutput(out, paths), nil, nil
	}

	if args["symbol"] != "" && args["path"] != "" {
		return ExecuteReadFileBySymbolWithRuntime(out, execCtx.EffectiveConfig(), execCtx.EffectiveToolCache(), args["path"], args["symbol"]), nil, nil
	}

	// 単体モード: 従来の path + start_line/end_line
	if args["path"] == "" {
		return "Error: path or paths is required", nil, nil
	}
	startLine, endLine := 0, 0
	if args["start_line"] != "" {
		if n, err := strconv.Atoi(args["start_line"]); err == nil {
			startLine = n
		}
	}
	if args["end_line"] != "" {
		if n, err := strconv.Atoi(args["end_line"]); err == nil {
			endLine = n
		}
	}
	return ExecuteReadFileWithRuntime(out, execCtx.EffectiveConfig(), execCtx.EffectiveToolCache(), args["path"], startLine, endLine), nil, nil
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
	return result, &tools.FileChange{
		FilePath:     args["path"],
		Timestamp:    common.GetCurrentTime(),
		Tool:         "str_replace",
		Description:  "Replaced in " + args["path"],
		LinesAdded:   countLines(args["new_str"]),
		LinesRemoved: countLines(args["old_str"]),
	}, nil
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
