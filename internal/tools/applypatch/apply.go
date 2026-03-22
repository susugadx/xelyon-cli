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
	ops := make([]plannedFileOp, 0, len(parsed.Hunks)*2)
	snapshots := make(map[string]fileSnapshot, len(parsed.Hunks)*2)
	result := &ApplyResult{
		Added:    make([]string, 0),
		Modified: make([]string, 0),
		Deleted:  make([]string, 0),
		details:  make([]applyResultDetail, 0, len(parsed.Hunks)),
	}

	for _, hunk := range parsed.Hunks {
		switch hunk.Type {
		case "add":
			absPath, err := common.ValidatePath(hunk.Path)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to write file %s: %w", hunk.Path, err)
			}
			snapshot, err := captureSnapshot(snapshots, absPath)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to write file %s: %w", hunk.Path, err)
			}
			if snapshot.Existed {
				return nil, nil, nil, fmt.Errorf("failed to add file %s: file already exists", hunk.Path)
			}
			ops = append(ops, plannedFileOp{
				Kind:     "write",
				Path:     absPath,
				Contents: []byte(hunk.Contents),
				Perm:     0o644,
			})
			result.Added = append(result.Added, hunk.Path)
			result.details = append(result.details, applyResultDetail{
				Path:       hunk.Path,
				Action:     "created",
				LinesAdded: countContentLines(hunk.Contents),
			})
		case "delete":
			absPath, err := common.ValidatePath(hunk.Path)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to delete file %s: %w", hunk.Path, err)
			}
			snapshot, err := captureSnapshot(snapshots, absPath)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to delete file %s: %w", hunk.Path, err)
			}
			if !snapshot.Existed {
				return nil, nil, nil, fmt.Errorf("failed to delete file %s: %w", hunk.Path, &os.PathError{
					Op:   "remove",
					Path: hunk.Path,
					Err:  os.ErrNotExist,
				})
			}
			ops = append(ops, plannedFileOp{Kind: "delete", Path: absPath})
			result.Deleted = append(result.Deleted, hunk.Path)
			result.details = append(result.details, applyResultDetail{
				Path:         hunk.Path,
				Action:       "deleted",
				LinesRemoved: countContentLines(string(snapshot.Contents)),
			})
		case "update":
			absPath, err := common.ValidatePath(hunk.Path)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to read file to update %s: %w", hunk.Path, err)
			}
			snapshot, err := captureSnapshot(snapshots, absPath)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to read file to update %s: %w", hunk.Path, err)
			}
			if !snapshot.Existed {
				return nil, nil, nil, fmt.Errorf("failed to read file to update %s: %w", hunk.Path, &os.PathError{
					Op:   "open",
					Path: hunk.Path,
					Err:  os.ErrNotExist,
				})
			}

			newContents, err := deriveNewContentsFromChunks(hunk.Path, snapshot.Contents, hunk.Chunks)
			if err != nil {
				return nil, nil, nil, err
			}

			if hunk.MovePath != "" {
				destPath, err := common.ValidatePath(hunk.MovePath)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("failed to move file to %s: %w", hunk.MovePath, err)
				}
				if destPath != absPath {
					if _, err := captureSnapshot(snapshots, destPath); err != nil {
						return nil, nil, nil, fmt.Errorf("failed to move file to %s: %w", hunk.MovePath, err)
					}

					ops = append(ops,
						plannedFileOp{
							Kind:     "write",
							Path:     destPath,
							Contents: []byte(newContents),
							Perm:     snapshot.Perm,
						},
						plannedFileOp{
							Kind: "delete",
							Path: absPath,
						},
					)
					result.Modified = append(result.Modified, hunk.MovePath)
					result.details = append(result.details, applyResultDetail{
						Path:   hunk.MovePath,
						Action: "moved",
					})
					continue
				}
			}

			ops = append(ops, plannedFileOp{
				Kind:     "write",
				Path:     absPath,
				Contents: []byte(newContents),
				Perm:     snapshot.Perm,
			})
			result.Modified = append(result.Modified, hunk.Path)
			result.details = append(result.details, applyResultDetail{
				Path:   hunk.Path,
				Action: "modified",
			})
		default:
			return nil, nil, nil, fmt.Errorf("unsupported hunk type: %s", hunk.Type)
		}
	}

	return ops, snapshots, result, nil
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
	if err := os.WriteFile(path, contents, perm); err != nil {
		return err
	}
	return os.Chmod(path, perm)
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

	for _, chunk := range chunks {
		if chunk.ChangeContext != "" {
			idx, ok := SeekSequence(originalLines, []string{chunk.ChangeContext}, lineIndex, false)
			if !ok {
				return nil, fmt.Errorf("failed to find context '%s' in %s", chunk.ChangeContext, path)
			}
			lineIndex = idx + 1
		}

		if len(chunk.OldLines) == 0 {
			insertionIdx := len(originalLines)
			if len(originalLines) > 0 && originalLines[len(originalLines)-1] == "" {
				insertionIdx = len(originalLines) - 1
			}
			replacements = append(replacements, replacement{
				StartIndex: insertionIdx,
				OldLen:     0,
				NewLines:   append([]string(nil), chunk.NewLines...),
			})
			continue
		}

		pattern := chunk.OldLines
		newSlice := chunk.NewLines
		startIdx, ok := SeekSequence(originalLines, pattern, lineIndex, chunk.IsEndOfFile)

		if !ok && len(pattern) > 0 && pattern[len(pattern)-1] == "" {
			pattern = pattern[:len(pattern)-1]
			if len(newSlice) > 0 && newSlice[len(newSlice)-1] == "" {
				newSlice = newSlice[:len(newSlice)-1]
			}
			startIdx, ok = SeekSequence(originalLines, pattern, lineIndex, chunk.IsEndOfFile)
		}

		if !ok {
			return nil, fmt.Errorf("failed to find expected lines in %s:\n%s", path, strings.Join(chunk.OldLines, "\n"))
		}

		replacements = append(replacements, replacement{
			StartIndex: startIdx,
			OldLen:     len(pattern),
			NewLines:   append([]string(nil), newSlice...),
		})
		lineIndex = startIdx + len(pattern)
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
