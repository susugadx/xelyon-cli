package applypatch

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// ApplyResult は apply_patch の適用結果を表す。
type ApplyResult struct {
	Added    []string
	Modified []string
	Deleted  []string
	details  []applyResultDetail
}

type replacement struct {
	StartIndex int
	OldLen     int
	NewLines   []string
}

type applyResultDetail struct {
	Path         string
	Action       string
	LinesAdded   int
	LinesRemoved int
	OldContent   string
	NewContent   string
}

type fileSnapshot struct {
	Existed  bool
	Contents []byte
	Perm     os.FileMode
}

type plannedFileOp struct {
	Kind     string
	Path     string
	Contents []byte
	Perm     os.FileMode
}

type patchPlanner struct {
	ops       []plannedFileOp
	snapshots map[string]fileSnapshot
	result    *ApplyResult
}

// ApplyPatch はパッチテキストを解析してファイルへ適用する。
func ApplyPatch(patchText string) (*ApplyResult, error) {
	parsed, err := ParsePatch(patchText)
	if err != nil {
		return nil, err
	}
	return applyParsedPatch(parsed)
}

func applyParsedPatch(parsed *ParsedPatch) (*ApplyResult, error) {
	if parsed == nil || len(parsed.Hunks) == 0 {
		return nil, fmt.Errorf("no files were modified")
	}

	ops, snapshots, result, err := buildPlannedFileOps(parsed)
	if err != nil {
		return nil, err
	}

	if err := applyFileOps(ops, snapshots); err != nil {
		return nil, err
	}

	return result, nil
}

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

func deriveNewContentsFromChunks(path string, originalContents []byte, chunks []UpdateFileChunk) (string, error) {
	originalLines := strings.Split(string(originalContents), "\n")
	if len(originalLines) > 0 && originalLines[len(originalLines)-1] == "" {
		originalLines = originalLines[:len(originalLines)-1]
	}

	replacements, err := computeReplacements(originalLines, path, chunks)
	if err != nil {
		return "", err
	}

	newLines := applyReplacements(originalLines, replacements)
	if len(newLines) == 0 || newLines[len(newLines)-1] != "" {
		newLines = append(newLines, "")
	}

	return strings.Join(newLines, "\n"), nil
}

func captureSnapshot(snapshots map[string]fileSnapshot, path string) (fileSnapshot, error) {
	if snapshot, ok := snapshots[path]; ok {
		return snapshot, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			snapshot := fileSnapshot{}
			snapshots[path] = snapshot
			return snapshot, nil
		}
		return fileSnapshot{}, err
	}
	if info.IsDir() {
		return fileSnapshot{}, fmt.Errorf("path is a directory: %s", path)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return fileSnapshot{}, err
	}

	snapshot := fileSnapshot{
		Existed:  true,
		Contents: contents,
		Perm:     info.Mode().Perm(),
	}
	snapshots[path] = snapshot
	return snapshot, nil
}

func applyFileOps(ops []plannedFileOp, snapshots map[string]fileSnapshot) error {
	for _, op := range ops {
		var err error
		switch op.Kind {
		case "write":
			err = writeFileWithPerm(op.Path, op.Contents, op.Perm)
		case "delete":
			err = os.Remove(op.Path)
		default:
			err = fmt.Errorf("unsupported file operation: %s", op.Kind)
		}
		if err == nil {
			continue
		}
		if rollbackErr := rollbackSnapshots(snapshots); rollbackErr != nil {
			return fmt.Errorf("%w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}
	return nil
}

func writeFileWithPerm(path string, contents []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if perm == 0 {
		perm = 0o644
	}

	// 同じディレクトリにtempファイルを作る（同一FS上でrenameがアトミックになる）
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".xelyon-patch-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	success := false
	defer func() {
		if !success {
			tmp.Close()
			_ = os.Remove(tmpPath) //nolint:errcheck // クリーンアップ失敗は無視
		}
	}()

	if _, err := tmp.Write(contents); err != nil {
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	success = true
	return nil
}

func rollbackSnapshots(snapshots map[string]fileSnapshot) error {
	paths := make([]string, 0, len(snapshots))
	for path := range snapshots {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	var errs []string
	for _, path := range paths {
		snapshot := snapshots[path]
		if !snapshot.Existed {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			}
			continue
		}
		if err := writeFileWithPerm(path, snapshot.Contents, snapshot.Perm); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func countContentLines(s string) int {
	if s == "" {
		return 0
	}
	parts := strings.Split(s, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		return len(parts) - 1
	}
	return len(parts)
}

func computeReplacements(originalLines []string, path string, chunks []UpdateFileChunk) ([]replacement, error) {
	replacements := make([]replacement, 0, len(chunks))
	lineIndex := 0
	prevChunkEnd := 0

	for _, chunk := range chunks {
		result, err := LocateChunk(originalLines, path, chunk, lineIndex, prevChunkEnd)
		if err != nil {
			return nil, err
		}

		replacements = append(replacements, replacement{
			StartIndex: result.StartIdx,
			OldLen:     len(result.Pattern),
			NewLines:   append([]string(nil), result.NewLines...),
		})
		lineIndex = result.NextIndex
		if len(result.Pattern) > 0 {
			prevChunkEnd = result.StartIdx + len(result.Pattern)
		}
	}

	sort.Slice(replacements, func(i, j int) bool {
		return replacements[i].StartIndex < replacements[j].StartIndex
	})

	return replacements, nil
}

func applyReplacements(lines []string, replacements []replacement) []string {
	current := append([]string(nil), lines...)

	for i := len(replacements) - 1; i >= 0; i-- {
		r := replacements[i]
		end := r.StartIndex + r.OldLen
		if end > len(current) {
			end = len(current)
		}

		updated := make([]string, 0, len(current)-r.OldLen+len(r.NewLines))
		updated = append(updated, current[:r.StartIndex]...)
		updated = append(updated, r.NewLines...)
		updated = append(updated, current[end:]...)
		current = updated
	}

	return current
}

// looksLikeUnifiedDiffHeader は @@ の後のテキストが unified diff の行番号形式かを判定する。
// 例: "-274,6 +274,32 @@", "-43,7 +43,7 @@"
func looksLikeUnifiedDiffHeader(header string) bool {
	trimmed := strings.TrimSpace(header)
	if !strings.HasPrefix(trimmed, "-") {
		return false
	}
	return strings.ContainsAny(trimmed, "0123456789") && strings.Contains(trimmed, ",")
}
