package file

import (
	"strconv"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// ReadFileTool needs special handling for optional line range
type ReadFileTool struct{}

func (t *ReadFileTool) Name() string { return "read_file" }

func (t *ReadFileTool) Run(args map[string]string) (string, *tools.FileChange, error) {
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
	return ExecuteReadFile(args["path"], startLine, endLine), nil, nil
}

// WriteFileTool wraps write_file execution
type WriteFileTool struct{}

func (t *WriteFileTool) Name() string { return "write_file" }

func (t *WriteFileTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	result, backupPath, err := ExecuteWriteFile(args["path"], args["content"])
	if err != nil {
		return result, nil, err
	}
	if backupPath == "" {
		return result, nil, nil
	}
	return result, &tools.FileChange{
		FilePath:    args["path"],
		BackupPath:  backupPath,
		Timestamp:   common.GetCurrentTime(),
		Tool:        "write_file",
		Description: "Wrote file " + args["path"],
	}, nil
}

// StrReplaceTool wraps str_replace execution
type StrReplaceTool struct{}

func (t *StrReplaceTool) Name() string { return "str_replace" }

func (t *StrReplaceTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	result, backupPath, err := ExecuteStrReplace(args["path"], args["old_str"], args["new_str"], args["start_line"], args["end_line"])
	if err != nil {
		return result, nil, err
	}
	if backupPath == "" {
		return result, nil, nil
	}
	return result, &tools.FileChange{
		FilePath:    args["path"],
		BackupPath:  backupPath,
		Timestamp:   common.GetCurrentTime(),
		Tool:        "str_replace",
		Description: "Replaced in " + args["path"],
	}, nil
}

// DeleteFileTool wraps delete_file execution
type DeleteFileTool struct{}

func (t *DeleteFileTool) Name() string { return "delete_file" }

func (t *DeleteFileTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	result, backupPath, err := ExecuteDeleteFile(args["path"])
	if err != nil {
		return result, nil, err
	}
	if backupPath == "" {
		return result, nil, nil
	}
	return result, &tools.FileChange{
		FilePath:    args["path"],
		BackupPath:  backupPath,
		Timestamp:   common.GetCurrentTime(),
		Tool:        "delete_file",
		Description: "Deleted file " + args["path"],
	}, nil
}

// ListDirTool wraps list_dir execution
type ListDirTool struct{}

func (t *ListDirTool) Name() string { return "list_dir" }

func (t *ListDirTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	return ExecuteListDir(args["path"]), nil, nil
}

// RestoreBackupTool wraps restore_backup execution
type RestoreBackupTool struct{}

func (t *RestoreBackupTool) Name() string { return "restore_backup" }

func (t *RestoreBackupTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	result, err := ExecuteRestoreBackup(args["path"], args["backup_path"])
	if err != nil {
		return result, nil, err
	}
	return result, nil, nil
}

// ListBackupsTool wraps list_backups execution
type ListBackupsTool struct{}

func (t *ListBackupsTool) Name() string { return "list_backups" }

func (t *ListBackupsTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	return ExecuteListBackups(args["path"]), nil, nil
}

// RegisterTools registers all file tools to the given registry
func RegisterTools(r *tools.Registry) {
	r.Register(&ReadFileTool{})
	r.Register(&WriteFileTool{})
	r.Register(&StrReplaceTool{})
	r.Register(&DeleteFileTool{})
	r.Register(&ListDirTool{})
	r.Register(&RestoreBackupTool{})
	r.Register(&ListBackupsTool{})
}

// init registers all file tools to the default registry
func init() {
	RegisterTools(tools.DefaultRegistry)
}
