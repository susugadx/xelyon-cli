package applypatch

import (
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

type patchPlanner struct {
	ops       []plannedFileOp
	snapshots map[string]fileSnapshot
	result    *ApplyResult
}

// ApplyPatch はパッチテキストを解析してファイルへ適用する。
func buildPlannedFileOps(parsed *ParsedPatch) ([]plannedFileOp, map[string]fileSnapshot, *ApplyResult, error) {
	planner := newPatchPlanner(len(parsed.Hunks))
	for _, hunk := range parsed.Hunks {
		if err := planner.planHunk(hunk); err != nil {
			return nil, nil, nil, err
		}
	}
	return planner.ops, planner.snapshots, planner.result, nil
}

func newPatchPlanner(hunkCount int) *patchPlanner {
	return &patchPlanner{
		ops:       make([]plannedFileOp, 0, hunkCount*2),
		snapshots: make(map[string]fileSnapshot, hunkCount*2),
		result: &ApplyResult{
			Added:    make([]string, 0),
			Modified: make([]string, 0),
			Deleted:  make([]string, 0),
			details:  make([]applyResultDetail, 0, hunkCount),
		},
	}
}

func (p *patchPlanner) planHunk(hunk Hunk) error {
	switch hunk.Type {
	case "add":
		return p.planAddHunk(hunk)
	case "delete":
		return p.planDeleteHunk(hunk)
	case "update":
		return p.planUpdateHunk(hunk)
	default:
		return fmt.Errorf("unsupported hunk type: %s", hunk.Type)
	}
}

func (p *patchPlanner) planAddHunk(hunk Hunk) error {
	absPath, err := common.ValidatePath(hunk.Path)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", hunk.Path, err)
	}
	snapshot, err := captureSnapshot(p.snapshots, absPath)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", hunk.Path, err)
	}
	if snapshot.Existed {
		return fmt.Errorf("failed to add file %s: file already exists", hunk.Path)
	}

	p.ops = append(p.ops, plannedFileOp{
		Kind:     "write",
		Path:     absPath,
		Contents: []byte(hunk.Contents),
		Perm:     0o644,
	})
	p.result.Added = append(p.result.Added, hunk.Path)
	p.result.details = append(p.result.details, applyResultDetail{
		Path:       hunk.Path,
		Action:     "created",
		LinesAdded: countContentLines(hunk.Contents),
		NewContent: string(hunk.Contents),
	})
	return nil
}

func (p *patchPlanner) planDeleteHunk(hunk Hunk) error {
	absPath, err := common.ValidatePath(hunk.Path)
	if err != nil {
		return fmt.Errorf("failed to delete file %s: %w", hunk.Path, err)
	}
	snapshot, err := captureSnapshot(p.snapshots, absPath)
	if err != nil {
		return fmt.Errorf("failed to delete file %s: %w", hunk.Path, err)
	}
	if !snapshot.Existed {
		return fmt.Errorf("failed to delete file %s: %w", hunk.Path, &os.PathError{
			Op:   "remove",
			Path: hunk.Path,
			Err:  os.ErrNotExist,
		})
	}

	p.ops = append(p.ops, plannedFileOp{Kind: "delete", Path: absPath})
	p.result.Deleted = append(p.result.Deleted, hunk.Path)
	p.result.details = append(p.result.details, applyResultDetail{
		Path:         hunk.Path,
		Action:       "deleted",
		LinesRemoved: countContentLines(string(snapshot.Contents)),
		OldContent:   string(snapshot.Contents),
	})
	return nil
}

func (p *patchPlanner) planUpdateHunk(hunk Hunk) error {
	absPath, snapshot, err := p.loadUpdateSource(hunk.Path)
	if err != nil {
		return err
	}

	newContents, err := deriveNewContentsFromChunks(hunk.Path, snapshot.Contents, hunk.Chunks)
	if err != nil {
		return err
	}

	if hunk.MovePath != "" {
		moved, err := p.tryPlanMoveUpdate(hunk, absPath, snapshot, newContents)
		if err != nil {
			return err
		}
		if moved {
			return nil
		}
	}

	p.ops = append(p.ops, plannedFileOp{
		Kind:     "write",
		Path:     absPath,
		Contents: []byte(newContents),
		Perm:     snapshot.Perm,
	})
	p.result.Modified = append(p.result.Modified, hunk.Path)
	p.result.details = append(p.result.details, applyResultDetail{
		Path:       hunk.Path,
		Action:     "modified",
		OldContent: string(snapshot.Contents),
		NewContent: newContents,
	})
	return nil
}

func (p *patchPlanner) loadUpdateSource(path string) (string, fileSnapshot, error) {
	absPath, err := common.ValidatePath(path)
	if err != nil {
		return "", fileSnapshot{}, fmt.Errorf("failed to read file to update %s: %w", path, err)
	}
	snapshot, err := captureSnapshot(p.snapshots, absPath)
	if err != nil {
		return "", fileSnapshot{}, fmt.Errorf("failed to read file to update %s: %w", path, err)
	}
	if !snapshot.Existed {
		return "", fileSnapshot{}, fmt.Errorf("failed to read file to update %s: %w", path, &os.PathError{
			Op:   "open",
			Path: path,
			Err:  os.ErrNotExist,
		})
	}
	return absPath, snapshot, nil
}

func (p *patchPlanner) tryPlanMoveUpdate(hunk Hunk, sourcePath string, snapshot fileSnapshot, newContents string) (bool, error) {
	destPath, err := common.ValidatePath(hunk.MovePath)
	if err != nil {
		return false, fmt.Errorf("failed to move file to %s: %w", hunk.MovePath, err)
	}
	if destPath == sourcePath {
		return false, nil
	}

	if _, err := captureSnapshot(p.snapshots, destPath); err != nil {
		return false, fmt.Errorf("failed to move file to %s: %w", hunk.MovePath, err)
	}

	p.ops = append(p.ops,
		plannedFileOp{
			Kind:     "write",
			Path:     destPath,
			Contents: []byte(newContents),
			Perm:     snapshot.Perm,
		},
		plannedFileOp{
			Kind: "delete",
			Path: sourcePath,
		},
	)
	p.result.Modified = append(p.result.Modified, hunk.MovePath)
	p.result.details = append(p.result.details, applyResultDetail{
		Path:       hunk.MovePath,
		Action:     "moved",
		OldContent: string(snapshot.Contents),
		NewContent: newContents,
	})
	return true, nil
}
