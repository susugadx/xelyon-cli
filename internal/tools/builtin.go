package tools

import (
	"strconv"
)

// parseInt は文字列を整数に変換するヘルパー関数
func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// ===== Generic Tool Wrappers =====

// SimpleTool は引数を受け取り結果のみ返すシンプルなツール
type SimpleTool struct {
	name    string
	execute func(args map[string]string) string
}

func (t *SimpleTool) Name() string { return t.name }

func (t *SimpleTool) Run(_ ExecutionContext, args map[string]string) (string, *FileChange, error) {
	return t.execute(args), nil, nil
}

// FileModifyingTool はファイルを変更しFileChangeを返すツール
type FileModifyingTool struct {
	name        string
	execute     func(args map[string]string) (result string, err error)
	description func(args map[string]string) string
	getFilePath func(args map[string]string) string
}

func (t *FileModifyingTool) Name() string { return t.name }

func (t *FileModifyingTool) Run(_ ExecutionContext, args map[string]string) (string, *FileChange, error) {
	result, err := t.execute(args)
	if err != nil {
		return result, nil, err
	}
	return result, &FileChange{
		FilePath:    t.getFilePath(args),
		Timestamp:   getCurrentTime(),
		Tool:        t.name,
		Description: t.description(args),
	}, nil
}

// ===== Registry Registration =====

// RegisterBuiltinTools はすべての組み込みツールを登録
// NOTE: All tools are now registered by subpackages:
//   - tools/file: read_file, write_file, str_replace, delete_file, list_dir
//   - tools/search: web_search, search_code
//   - tools/dev: bash
//   - tools/lsp: (diagnostics only)
func RegisterBuiltinTools(r *Registry) {
	// All tools are now registered by subpackages via init()
	// This function is kept for backward compatibility
}

// init は自動的にデフォルトレジストリに全ツールを登録
func init() {
	RegisterBuiltinTools(DefaultRegistry)
}
