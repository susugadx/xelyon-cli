package tools

import (
	"fmt"
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

// ===== Special Tools with custom logic =====

// ReadFileTool needs special handling for optional line range
type ReadFileTool struct{}

func (t *ReadFileTool) Name() string { return "read_file" }

func (t *ReadFileTool) Run(args map[string]string) (string, *FileChange, error) {
	startLine, endLine := 0, 0
	if args["start_line"] != "" {
		if n, err := parseInt(args["start_line"]); err == nil {
			startLine = n
		}
	}
	if args["end_line"] != "" {
		if n, err := parseInt(args["end_line"]); err == nil {
			endLine = n
		}
	}
	return executeReadFile(args["path"], startLine, endLine), nil, nil
}

// GrepReplaceTool requires confirmation prompt
type GrepReplaceTool struct{}

func (t *GrepReplaceTool) Name() string { return "grep_replace" }

func (t *GrepReplaceTool) Run(args map[string]string) (string, *FileChange, error) {
	dryRun := args["dry_run"] == "true"
	if !dryRun {
		yellow.Printf("🔄 Bulk replace across files\n")
		fmt.Printf("  Pattern: %s\n", args["pattern"])
		fmt.Printf("  Replacement: %s\n", args["replacement"])
		fmt.Printf("  Path: %s\n", args["path"])
		fmt.Printf("  File pattern: %s\n", args["file_pattern"])

		decision := Confirm("Execute replacement? ")
		if decision.Action == ConfirmNo {
			return "Replacement cancelled by user", nil, nil
		}
		if decision.Action == ConfirmComment {
			return fmt.Sprintf("Replacement cancelled with comment: %s", decision.Comment), nil, nil
		}
	}
	result, _, err := executeGrepReplace(args["pattern"], args["replacement"], args["path"], args["file_pattern"], dryRun)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil, err
	}
	return result, nil, nil
}

// HTTPRequestTool needs special handling for timeout
type HTTPRequestTool struct{}

func (t *HTTPRequestTool) Name() string { return "http_request" }

func (t *HTTPRequestTool) Run(args map[string]string) (string, *FileChange, error) {
	timeout := 30
	if args["timeout"] != "" {
		if n, err := parseInt(args["timeout"]); err == nil && n > 0 {
			timeout = n
		}
	}
	result, err := executeHTTPRequest(args["method"], args["url"], args["headers"], args["body"], timeout)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil, err
	}
	return result, nil, nil
}

// DiffFilesTool needs special handling for context lines
type DiffFilesTool struct{}

func (t *DiffFilesTool) Name() string { return "diff_files" }

func (t *DiffFilesTool) Run(args map[string]string) (string, *FileChange, error) {
	contextLines := 3
	if args["context"] != "" {
		if n, err := parseInt(args["context"]); err == nil && n > 0 {
			contextLines = n
		}
	}
	result, err := executeDiffFiles(args["file1"], args["file2"], contextLines)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil, err
	}
	return result, nil, nil
}

// RestoreBackupTool needs special error handling
type RestoreBackupTool struct{}

func (t *RestoreBackupTool) Name() string { return "restore_backup" }

func (t *RestoreBackupTool) Run(args map[string]string) (string, *FileChange, error) {
	result, err := executeRestoreBackup(args["path"], args["backup_path"])
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil, err
	}
	return result, nil, nil
}

// ===== Registry Registration =====

// RegisterBuiltinTools はすべての組み込みツールを登録
func RegisterBuiltinTools(r *Registry) {
	// === Simple Tools (no FileChange) ===
	simpleTools := []struct {
		name    string
		execute func(args map[string]string) string
	}{
		{"bash", func(args map[string]string) string { return executeBash(args["command"]) }},
		{"list_dir", func(args map[string]string) string { return executeListDir(args["path"]) }},
		{"git_status", func(args map[string]string) string { return executeGitStatus() }},
		{"git_diff", func(args map[string]string) string { return executeGitDiff(args["path"]) }},
		{"git_add", func(args map[string]string) string { return executeGitAdd(args["path"]) }},
		{"git_commit", func(args map[string]string) string { return executeGitCommit(args["message"]) }},
		{"git_push", func(args map[string]string) string { return executeGitPush() }},
		{"git_log", func(args map[string]string) string { return executeGitLog() }},
		{"git_branch", func(args map[string]string) string { return executeGitBranch(args["action"], args["branch_name"]) }},
		{"git_stash", func(args map[string]string) string { return executeGitStash(args["action"], args["message"]) }},
		{"search_code", func(args map[string]string) string { return executeSearchCode(args["pattern"], args["path"]) }},
		{"search_file", func(args map[string]string) string { return executeSearchFile(args["pattern"], args["path"]) }},
		{"web_search", func(args map[string]string) string { return executeWebSearch(args["query"]) }},
		{"create_dir", func(args map[string]string) string { return executeCreateDir(args["path"]) }},
		{"run_test", func(args map[string]string) string { return executeRunTest(args["path"]) }},
		{"ast_grep", func(args map[string]string) string {
			return executeAstGrep(args["pattern"], args["lang"], args["path"])
		}},
		{"list_backups", func(args map[string]string) string { return executeListBackups(args["path"]) }},
	}
	for _, t := range simpleTools {
		r.Register(&SimpleTool{name: t.name, execute: t.execute})
	}

	// === File-modifying Tools (with FileChange) ===
	fileTools := []struct {
		name        string
		execute     func(args map[string]string) (string, string, error)
		description func(args map[string]string) string
		getFilePath func(args map[string]string) string
	}{
		{
			name: "write_file",
			execute: func(args map[string]string) (string, string, error) {
				return executeWriteFile(args["path"], args["content"])
			},
			description: func(args map[string]string) string { return "Wrote file " + args["path"] },
			getFilePath: func(args map[string]string) string { return args["path"] },
		},
		{
			name: "str_replace",
			execute: func(args map[string]string) (string, string, error) {
				return executeStrReplace(args["path"], args["old_str"], args["new_str"], args["start_line"], args["end_line"])
			},
			description: func(args map[string]string) string { return "Replaced in " + args["path"] },
			getFilePath: func(args map[string]string) string { return args["path"] },
		},
		{
			name: "append_file",
			execute: func(args map[string]string) (string, string, error) {
				return executeAppendFile(args["path"], args["content"])
			},
			description: func(args map[string]string) string { return "Appended to file " + args["path"] },
			getFilePath: func(args map[string]string) string { return args["path"] },
		},
		{
			name: "prepend_file",
			execute: func(args map[string]string) (string, string, error) {
				return executePrependFile(args["path"], args["content"])
			},
			description: func(args map[string]string) string { return "Prepended to file " + args["path"] },
			getFilePath: func(args map[string]string) string { return args["path"] },
		},
		{
			name:        "format",
			execute:     func(args map[string]string) (string, string, error) { return executeFormat(args["path"]) },
			description: func(args map[string]string) string { return "Formatted file " + args["path"] },
			getFilePath: func(args map[string]string) string { return args["path"] },
		},
		{
			name:        "git_checkout",
			execute:     func(args map[string]string) (string, string, error) { return executeGitCheckout(args["target"]) },
			description: func(args map[string]string) string { return "Restored from HEAD: " + args["target"] },
			getFilePath: func(args map[string]string) string { return args["target"] },
		},
		{
			name: "insert_after",
			execute: func(args map[string]string) (string, string, error) {
				return executeInsertAfter(args["path"], args["pattern"], args["content"])
			},
			description: func(args map[string]string) string { return "Inserted content after pattern in " + args["path"] },
			getFilePath: func(args map[string]string) string { return args["path"] },
		},
		{
			name: "insert_before",
			execute: func(args map[string]string) (string, string, error) {
				return executeInsertBefore(args["path"], args["pattern"], args["content"])
			},
			description: func(args map[string]string) string { return "Inserted content before pattern in " + args["path"] },
			getFilePath: func(args map[string]string) string { return args["path"] },
		},
		{
			name: "copy_file",
			execute: func(args map[string]string) (string, string, error) {
				return executeCopyFile(args["src"], args["dest"])
			},
			description: func(args map[string]string) string { return "Copied " + args["src"] + " to " + args["dest"] },
			getFilePath: func(args map[string]string) string { return args["dest"] },
		},
		{
			name: "delete_lines",
			execute: func(args map[string]string) (string, string, error) {
				return executeDeleteLines(args["path"], args["start_line"], args["end_line"])
			},
			description: func(args map[string]string) string {
				return "Deleted lines " + args["start_line"] + "-" + args["end_line"] + " in " + args["path"]
			},
			getFilePath: func(args map[string]string) string { return args["path"] },
		},
		{
			name:        "delete_file",
			execute:     func(args map[string]string) (string, string, error) { return executeDeleteFile(args["path"]) },
			description: func(args map[string]string) string { return "Deleted file " + args["path"] },
			getFilePath: func(args map[string]string) string { return args["path"] },
		},
		{
			name: "move_file",
			execute: func(args map[string]string) (string, string, error) {
				return executeMoveFile(args["src"], args["dest"])
			},
			description: func(args map[string]string) string { return "Moved " + args["src"] + " to " + args["dest"] },
			getFilePath: func(args map[string]string) string { return args["dest"] },
		},
		{
			name: "lint",
			execute: func(args map[string]string) (string, string, error) {
				return executeLint(args["path"], args["auto_fix"])
			},
			description: func(args map[string]string) string { return "Lint auto-fix on " + args["path"] },
			getFilePath: func(args map[string]string) string { return args["path"] },
		},
	}
	for _, t := range fileTools {
		r.Register(&FileModifyingTool{
			name:        t.name,
			execute:     t.execute,
			description: t.description,
			getFilePath: t.getFilePath,
		})
	}

	// === Special Tools (custom logic) ===
	r.Register(&ReadFileTool{})
	r.Register(&GrepReplaceTool{})
	r.Register(&HTTPRequestTool{})
	r.Register(&DiffFilesTool{})
	r.Register(&RestoreBackupTool{})
}

// init は自動的にデフォルトレジストリに全ツールを登録
func init() {
	RegisterBuiltinTools(DefaultRegistry)
}
