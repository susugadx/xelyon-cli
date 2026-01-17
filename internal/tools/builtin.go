package tools

import "fmt"

// builtin.go - すべての組み込みツールのWrapper実装

// ===== Bash Tool =====
type BashTool struct{}

func (t *BashTool) Name() string { return "bash" }

func (t *BashTool) Run(args map[string]string) (string, *FileChange, error) {
	result := executeBash(args["command"])
	return result, nil, nil
}

// ===== Read File Tool =====
type ReadFileTool struct{}

func (t *ReadFileTool) Name() string { return "read_file" }

func (t *ReadFileTool) Run(args map[string]string) (string, *FileChange, error) {
	result := executeReadFile(args["path"])
	return result, nil, nil
}

// ===== Write File Tool =====
type WriteFileTool struct{}

func (t *WriteFileTool) Name() string { return "write_file" }

func (t *WriteFileTool) Run(args map[string]string) (string, *FileChange, error) {
	result, backupPath, err := executeWriteFile(args["path"], args["content"])
	if err != nil {
		return result, nil, err
	}
	var change *FileChange
	if backupPath != "" {
		change = &FileChange{
			FilePath:    args["path"],
			BackupPath:  backupPath,
			Timestamp:   getCurrentTime(),
			Tool:        "write_file",
			Description: "Wrote file " + args["path"],
		}
	}
	return result, change, nil
}

// ===== Str Replace Tool =====
type StrReplaceTool struct{}

func (t *StrReplaceTool) Name() string { return "str_replace" }

func (t *StrReplaceTool) Run(args map[string]string) (string, *FileChange, error) {
	result, backupPath, err := executeStrReplace(args["path"], args["old_str"], args["new_str"], args["start_line"], args["end_line"])
	if err != nil {
		return result, nil, err
	}
	var change *FileChange
	if backupPath != "" {
		change = &FileChange{
			FilePath:    args["path"],
			BackupPath:  backupPath,
			Timestamp:   getCurrentTime(),
			Tool:        "str_replace",
			Description: "Replaced in " + args["path"],
		}
	}
	return result, change, nil
}

// ===== List Dir Tool =====
type ListDirTool struct{}

func (t *ListDirTool) Name() string { return "list_dir" }

func (t *ListDirTool) Run(args map[string]string) (string, *FileChange, error) {
	result := executeListDir(args["path"])
	return result, nil, nil
}

// ===== Git Status Tool =====
type GitStatusTool struct{}

func (t *GitStatusTool) Name() string { return "git_status" }

func (t *GitStatusTool) Run(args map[string]string) (string, *FileChange, error) {
	result := executeGitStatus()
	return result, nil, nil
}

// ===== Git Diff Tool =====
type GitDiffTool struct{}

func (t *GitDiffTool) Name() string { return "git_diff" }

func (t *GitDiffTool) Run(args map[string]string) (string, *FileChange, error) {
	result := executeGitDiff(args["path"])
	return result, nil, nil
}

// ===== Git Add Tool =====
type GitAddTool struct{}

func (t *GitAddTool) Name() string { return "git_add" }

func (t *GitAddTool) Run(args map[string]string) (string, *FileChange, error) {
	result := executeGitAdd(args["path"])
	return result, nil, nil
}

// ===== Git Commit Tool =====
type GitCommitTool struct{}

func (t *GitCommitTool) Name() string { return "git_commit" }

func (t *GitCommitTool) Run(args map[string]string) (string, *FileChange, error) {
	result := executeGitCommit(args["message"])
	return result, nil, nil
}

// ===== Git Push Tool =====
type GitPushTool struct{}

func (t *GitPushTool) Name() string { return "git_push" }

func (t *GitPushTool) Run(args map[string]string) (string, *FileChange, error) {
	result := executeGitPush()
	return result, nil, nil
}

// ===== Git Log Tool =====
type GitLogTool struct{}

func (t *GitLogTool) Name() string { return "git_log" }

func (t *GitLogTool) Run(args map[string]string) (string, *FileChange, error) {
	result := executeGitLog()
	return result, nil, nil
}

// ===== Search Code Tool =====
type SearchCodeTool struct{}

func (t *SearchCodeTool) Name() string { return "search_code" }

func (t *SearchCodeTool) Run(args map[string]string) (string, *FileChange, error) {
	result := executeSearchCode(args["pattern"], args["path"])
	return result, nil, nil
}

// ===== Search File Tool =====
type SearchFileTool struct{}

func (t *SearchFileTool) Name() string { return "search_file" }

func (t *SearchFileTool) Run(args map[string]string) (string, *FileChange, error) {
	result := executeSearchFile(args["pattern"], args["path"])
	return result, nil, nil
}

// ===== Web Search Tool =====
type WebSearchTool struct{}

func (t *WebSearchTool) Name() string { return "web_search" }

func (t *WebSearchTool) Run(args map[string]string) (string, *FileChange, error) {
	result := executeWebSearch(args["query"])
	return result, nil, nil
}

// ===== Append File Tool =====
type AppendFileTool struct{}

func (t *AppendFileTool) Name() string { return "append_file" }

func (t *AppendFileTool) Run(args map[string]string) (string, *FileChange, error) {
	result, backupPath, err := executeAppendFile(args["path"], args["content"])
	if err != nil {
		return result, nil, err
	}
	var change *FileChange
	if backupPath != "" {
		change = &FileChange{
			FilePath:    args["path"],
			BackupPath:  backupPath,
			Timestamp:   getCurrentTime(),
			Tool:        "append_file",
			Description: "Appended to file " + args["path"],
		}
	}
	return result, change, nil
}

// ===== Prepend File Tool =====
type PrependFileTool struct{}

func (t *PrependFileTool) Name() string { return "prepend_file" }

func (t *PrependFileTool) Run(args map[string]string) (string, *FileChange, error) {
	result, backupPath, err := executePrependFile(args["path"], args["content"])
	if err != nil {
		return result, nil, err
	}
	var change *FileChange
	if backupPath != "" {
		change = &FileChange{
			FilePath:    args["path"],
			BackupPath:  backupPath,
			Timestamp:   getCurrentTime(),
			Tool:        "prepend_file",
			Description: "Prepended to file " + args["path"],
		}
	}
	return result, change, nil
}

// ===== Create Directory Tool =====
type CreateDirTool struct{}

func (t *CreateDirTool) Name() string { return "create_dir" }

func (t *CreateDirTool) Run(args map[string]string) (string, *FileChange, error) {
	result := executeCreateDir(args["path"])
	return result, nil, nil
}

// ===== Run Test Tool =====
type RunTestTool struct{}

func (t *RunTestTool) Name() string { return "run_test" }

func (t *RunTestTool) Run(args map[string]string) (string, *FileChange, error) {
	result := executeRunTest(args["path"])
	return result, nil, nil
}

// ===== Format Tool =====
type FormatTool struct{}

func (t *FormatTool) Name() string { return "format" }

func (t *FormatTool) Run(args map[string]string) (string, *FileChange, error) {
	result, backupPath, err := executeFormat(args["path"])
	if err != nil {
		return result, nil, err
	}
	var change *FileChange
	if backupPath != "" {
		change = &FileChange{
			FilePath:    args["path"],
			BackupPath:  backupPath,
			Timestamp:   getCurrentTime(),
			Tool:        "format",
			Description: "Formatted file " + args["path"],
		}
	}
	return result, change, nil
}

// ===== Git Branch Tool =====
type GitBranchTool struct{}

func (t *GitBranchTool) Name() string { return "git_branch" }

func (t *GitBranchTool) Run(args map[string]string) (string, *FileChange, error) {
	result := executeGitBranch(args["action"], args["branch_name"])
	return result, nil, nil
}

// ===== Git Checkout Tool =====
type GitCheckoutTool struct{}

func (t *GitCheckoutTool) Name() string { return "git_checkout" }

func (t *GitCheckoutTool) Run(args map[string]string) (string, *FileChange, error) {
	result, backupPath, err := executeGitCheckout(args["target"])
	if err != nil {
		return result, nil, err
	}
	var change *FileChange
	if backupPath != "" {
		change = &FileChange{
			FilePath:    args["target"],
			BackupPath:  backupPath,
			Timestamp:   getCurrentTime(),
			Tool:        "git_checkout",
			Description: "Restored from HEAD: " + args["target"],
		}
	}
	return result, change, nil
}

// ===== Git Stash Tool =====
type GitStashTool struct{}

func (t *GitStashTool) Name() string { return "git_stash" }

func (t *GitStashTool) Run(args map[string]string) (string, *FileChange, error) {
	result := executeGitStash(args["action"], args["message"])
	return result, nil, nil
}

// ===== Insert After Tool =====
type InsertAfterTool struct{}

func (t *InsertAfterTool) Name() string { return "insert_after" }

func (t *InsertAfterTool) Run(args map[string]string) (string, *FileChange, error) {
	result, backupPath, err := executeInsertAfter(args["path"], args["pattern"], args["content"])
	if err != nil {
		return result, nil, err
	}
	var change *FileChange
	if backupPath != "" {
		change = &FileChange{
			FilePath:    args["path"],
			BackupPath:  backupPath,
			Timestamp:   getCurrentTime(),
			Tool:        "insert_after",
			Description: "Inserted content after pattern in " + args["path"],
		}
	}
	return result, change, nil
}

// ===== Insert Before Tool =====
type InsertBeforeTool struct{}

func (t *InsertBeforeTool) Name() string { return "insert_before" }

func (t *InsertBeforeTool) Run(args map[string]string) (string, *FileChange, error) {
	result, backupPath, err := executeInsertBefore(args["path"], args["pattern"], args["content"])
	if err != nil {
		return result, nil, err
	}
	var change *FileChange
	if backupPath != "" {
		change = &FileChange{
			FilePath:    args["path"],
			BackupPath:  backupPath,
			Timestamp:   getCurrentTime(),
			Tool:        "insert_before",
			Description: "Inserted content before pattern in " + args["path"],
		}
	}
	return result, change, nil
}

// ===== Copy File Tool =====
type CopyFileTool struct{}

func (t *CopyFileTool) Name() string { return "copy_file" }

func (t *CopyFileTool) Run(args map[string]string) (string, *FileChange, error) {
	result, backupPath, err := executeCopyFile(args["src"], args["dest"])
	if err != nil {
		return result, nil, err
	}
	var change *FileChange
	if backupPath != "" {
		change = &FileChange{
			FilePath:    args["dest"],
			BackupPath:  backupPath,
			Timestamp:   getCurrentTime(),
			Tool:        "copy_file",
			Description: "Copied " + args["src"] + " to " + args["dest"],
		}
	}
	return result, change, nil
}

// ===== Phase 4: Destructive/Complex Tools =====

// DeleteLinesTool deletes a range of lines from a file
type DeleteLinesTool struct{}

func (t *DeleteLinesTool) Name() string { return "delete_lines" }

func (t *DeleteLinesTool) Run(args map[string]string) (string, *FileChange, error) {
	result, backupPath, err := executeDeleteLines(args["path"], args["start_line"], args["end_line"])
	if err != nil {
		return result, nil, err
	}
	var change *FileChange
	if backupPath != "" {
		change = &FileChange{
			FilePath:    args["path"],
			BackupPath:  backupPath,
			Timestamp:   getCurrentTime(),
			Tool:        "delete_lines",
			Description: "Deleted lines " + args["start_line"] + "-" + args["end_line"] + " in " + args["path"],
		}
	}
	return result, change, nil
}

// DeleteFileTool deletes a file permanently (with backup)
type DeleteFileTool struct{}

func (t *DeleteFileTool) Name() string { return "delete_file" }

func (t *DeleteFileTool) Run(args map[string]string) (string, *FileChange, error) {
	result, backupPath, err := executeDeleteFile(args["path"])
	if err != nil {
		return result, nil, err
	}
	var change *FileChange
	if backupPath != "" {
		change = &FileChange{
			FilePath:    args["path"],
			BackupPath:  backupPath,
			Timestamp:   getCurrentTime(),
			Tool:        "delete_file",
			Description: "Deleted file " + args["path"],
		}
	}
	return result, change, nil
}

// MoveFileTool moves/renames a file
type MoveFileTool struct{}

func (t *MoveFileTool) Name() string { return "move_file" }

func (t *MoveFileTool) Run(args map[string]string) (string, *FileChange, error) {
	result, backupPath, err := executeMoveFile(args["src"], args["dest"])
	if err != nil {
		return result, nil, err
	}
	var change *FileChange
	if backupPath != "" {
		// 移動先が上書きされた場合のみFileChange作成
		change = &FileChange{
			FilePath:    args["dest"], // 移動先を追跡
			BackupPath:  backupPath,
			Timestamp:   getCurrentTime(),
			Tool:        "move_file",
			Description: "Moved " + args["src"] + " to " + args["dest"],
		}
	}
	return result, change, nil
}

// LintTool runs linter with optional auto-fix
type LintTool struct{}

func (t *LintTool) Name() string { return "lint" }

func (t *LintTool) Run(args map[string]string) (string, *FileChange, error) {
	result, backupPath, err := executeLint(args["path"], args["auto_fix"])
	if err != nil {
		return result, nil, err
	}
	var change *FileChange
	if backupPath != "" {
		change = &FileChange{
			FilePath:    args["path"],
			BackupPath:  backupPath,
			Timestamp:   getCurrentTime(),
			Tool:        "lint",
			Description: "Lint auto-fix on " + args["path"],
		}
	}
	return result, change, nil
}

// ===== Ast Grep Tool =====

// AstGrepTool provides structural code search using ast-grep (Tree-sitter based)
type AstGrepTool struct{}

func (t *AstGrepTool) Name() string { return "ast_grep" }

func (t *AstGrepTool) Run(args map[string]string) (string, *FileChange, error) {
	result := executeAstGrep(args["pattern"], args["lang"], args["path"])
	return result, nil, nil
}

// ===== Restore Backup Tool =====

// RestoreBackupTool restores a file from its backup
type RestoreBackupTool struct{}

func (t *RestoreBackupTool) Name() string { return "restore_backup" }

func (t *RestoreBackupTool) Run(args map[string]string) (string, *FileChange, error) {
	result, err := executeRestoreBackup(args["path"], args["backup_path"])
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil, err
	}
	return result, nil, nil
}

// ===== List Backups Tool =====

// ListBackupsTool lists available backups for a file
type ListBackupsTool struct{}

func (t *ListBackupsTool) Name() string { return "list_backups" }

func (t *ListBackupsTool) Run(args map[string]string) (string, *FileChange, error) {
	result := executeListBackups(args["path"])
	return result, nil, nil
}

// ===== Registry Registration =====

// RegisterBuiltinTools はすべての組み込みツールを登録
func RegisterBuiltinTools(r *Registry) {
	r.Register(&BashTool{})
	r.Register(&ReadFileTool{})
	r.Register(&WriteFileTool{})
	r.Register(&StrReplaceTool{})
	r.Register(&ListDirTool{})
	r.Register(&GitStatusTool{})
	r.Register(&GitDiffTool{})
	r.Register(&GitAddTool{})
	r.Register(&GitCommitTool{})
	r.Register(&GitPushTool{})
	r.Register(&GitLogTool{})
	r.Register(&GitBranchTool{})   // NEW: Phase 2
	r.Register(&GitCheckoutTool{}) // NEW: Phase 2
	r.Register(&GitStashTool{})    // NEW: Phase 2
	r.Register(&SearchCodeTool{})
	r.Register(&SearchFileTool{})
	r.Register(&WebSearchTool{})
	r.Register(&AppendFileTool{})
	r.Register(&PrependFileTool{})
	r.Register(&CreateDirTool{})
	r.Register(&RunTestTool{})
	r.Register(&FormatTool{})
	r.Register(&InsertAfterTool{})   // NEW: Phase 3
	r.Register(&InsertBeforeTool{})  // NEW: Phase 3
	r.Register(&CopyFileTool{})      // NEW: Phase 3
	r.Register(&DeleteLinesTool{})   // NEW: Phase 4
	r.Register(&DeleteFileTool{})    // NEW: Phase 4
	r.Register(&MoveFileTool{})      // NEW: Phase 4
	r.Register(&LintTool{})          // NEW: Phase 4
	r.Register(&AstGrepTool{})       // NEW: Issue #55 - ast-grep structural search
	r.Register(&RestoreBackupTool{}) // NEW: restore file from backup
	r.Register(&ListBackupsTool{})   // NEW: list available backups
}

// init は自動的にデフォルトレジストリに全ツールを登録
func init() {
	RegisterBuiltinTools(DefaultRegistry)
}
