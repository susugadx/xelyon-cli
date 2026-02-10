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

func (t *SimpleTool) Run(args map[string]string) (string, *FileChange, error) {
	return t.execute(args), nil, nil
}

// FileModifyingTool はファイルを変更しFileChangeを返すツール
type FileModifyingTool struct {
	name        string
	execute     func(args map[string]string) (result string, backupPath string, err error)
	description func(args map[string]string) string
	getFilePath func(args map[string]string) string
}

func (t *FileModifyingTool) Name() string { return t.name }

func (t *FileModifyingTool) Run(args map[string]string) (string, *FileChange, error) {
	result, backupPath, err := t.execute(args)
	if err != nil {
		return result, nil, err
	}
	if backupPath == "" {
		return result, nil, nil
	}
	return result, &FileChange{
		FilePath:    t.getFilePath(args),
		BackupPath:  backupPath,
		Timestamp:   getCurrentTime(),
		Tool:        t.name,
		Description: t.description(args),
	}, nil
}

// ===== Registry Registration =====

// RegisterBuiltinTools はすべての組み込みツールを登録
// NOTE: All tools are now registered by subpackages:
//   - tools/file: read_file, write_file, str_replace, delete_file, list_dir
//   - tools/search: search_code, search_file, web_search, grep_replace
//   - tools/dev: bash
//   - tools/lsp: lsp_find
func RegisterBuiltinTools(r *Registry) {
	// All tools are now registered by subpackages via init()
	// This function is kept for backward compatibility
}

// init は自動的にデフォルトレジストリに全ツールを登録
func init() {
	RegisterBuiltinTools(DefaultRegistry)
}
