package review

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MultiFileApplier handles applying and rolling back multi-file changes.
type MultiFileApplier struct {
	BackupDir string
}

// NewMultiFileApplier creates a new applier with a backup directory.
func NewMultiFileApplier() *MultiFileApplier {
	homeDir, _ := os.UserHomeDir()
	backupDir := filepath.Join(homeDir, ".xelyon", "backups")
	return &MultiFileApplier{BackupDir: backupDir}
}

// ValidateMultiFileChange validates that all changes can be applied.
func (a *MultiFileApplier) ValidateMultiFileChange(change *MultiFileChange) error {
	if len(change.Changes) == 0 {
		return fmt.Errorf("no changes to apply")
	}

	// Check for duplicate file paths
	seen := make(map[string]bool)
	for _, fc := range change.Changes {
		if seen[fc.FilePath] {
			return fmt.Errorf("duplicate file path: %s", fc.FilePath)
		}
		seen[fc.FilePath] = true
	}

	// Validate each file change
	for i := range change.Changes {
		fc := &change.Changes[i]
		if err := a.validateFileChange(fc); err != nil {
			return fmt.Errorf("file %s: %w", fc.FilePath, err)
		}
	}

	return nil
}

func (a *MultiFileApplier) validateFileChange(fc *FileChange) error {
	if fc.FilePath == "" {
		return fmt.Errorf("empty file path")
	}

	// Check file exists
	info, err := os.Stat(fc.FilePath)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory")
	}

	if len(fc.Patches) == 0 {
		return fmt.Errorf("no patches")
	}

	// Read file content for validation
	content, err := os.ReadFile(fc.FilePath)
	if err != nil {
		return fmt.Errorf("cannot read file: %w", err)
	}

	// Validate each patch
	for i, patch := range fc.Patches {
		if err := a.validatePatch(patch, string(content)); err != nil {
			return fmt.Errorf("patch %d: %w", i+1, err)
		}
	}

	return nil
}

func (a *MultiFileApplier) validatePatch(patch Patch, content string) error {
	if patch.StartLine > 0 && patch.EndLine > 0 {
		// Line-based patch
		lines := strings.Split(content, "\n")
		if patch.StartLine > len(lines) {
			return fmt.Errorf("start line %d exceeds file length %d", patch.StartLine, len(lines))
		}
		if patch.EndLine > len(lines) {
			return fmt.Errorf("end line %d exceeds file length %d", patch.EndLine, len(lines))
		}
		if patch.StartLine > patch.EndLine {
			return fmt.Errorf("start line %d > end line %d", patch.StartLine, patch.EndLine)
		}
	} else {
		// Code-matching patch
		if patch.OldCode == "" {
			return fmt.Errorf("OldCode is empty and no line range specified")
		}
		if !strings.Contains(content, patch.OldCode) {
			return fmt.Errorf("OldCode not found in file")
		}
		// Check for multiple matches
		count := strings.Count(content, patch.OldCode)
		if count > 1 {
			return fmt.Errorf("OldCode matches %d times (ambiguous)", count)
		}
	}

	return nil
}

// ApplyMultiFileChange applies all changes with automatic rollback on failure.
func (a *MultiFileApplier) ApplyMultiFileChange(change *MultiFileChange) error {
	// Validate first
	if err := a.ValidateMultiFileChange(change); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(a.BackupDir, 0755); err != nil {
		return fmt.Errorf("cannot create backup dir: %w", err)
	}

	// Create backups
	for i := range change.Changes {
		fc := &change.Changes[i]
		backupPath, err := a.createBackup(fc.FilePath)
		if err != nil {
			// Rollback any backups already created
			a.rollbackChanges(change.Changes[:i])
			return fmt.Errorf("backup failed for %s: %w", fc.FilePath, err)
		}
		fc.BackupPath = backupPath
	}

	// Apply changes
	for i := range change.Changes {
		fc := &change.Changes[i]
		if err := a.applyFileChange(fc); err != nil {
			// Rollback all changes
			a.rollbackChanges(change.Changes[:i+1])
			return fmt.Errorf("apply failed for %s: %w", fc.FilePath, err)
		}
	}

	return nil
}

func (a *MultiFileApplier) createBackup(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// Generate unique backup filename
	timestamp := time.Now().Format("20060102_150405")
	baseName := filepath.Base(filePath)
	backupName := fmt.Sprintf("%s_%s.bak", baseName, timestamp)
	backupPath := filepath.Join(a.BackupDir, backupName)

	// Ensure unique name
	for i := 1; ; i++ {
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			break
		}
		backupName = fmt.Sprintf("%s_%s_%d.bak", baseName, timestamp, i)
		backupPath = filepath.Join(a.BackupDir, backupName)
	}

	if err := os.WriteFile(backupPath, content, 0600); err != nil {
		return "", err
	}

	return backupPath, nil
}

func (a *MultiFileApplier) applyFileChange(fc *FileChange) error {
	content, err := os.ReadFile(fc.FilePath)
	if err != nil {
		return err
	}

	newContent := string(content)

	// Apply patches (in reverse order for line-based to avoid offset issues)
	// First, separate line-based and code-based patches
	var linePatches, codePatches []Patch
	for _, p := range fc.Patches {
		if p.StartLine > 0 && p.EndLine > 0 {
			linePatches = append(linePatches, p)
		} else {
			codePatches = append(codePatches, p)
		}
	}

	// Apply code-based patches first (order doesn't matter as much)
	for _, patch := range codePatches {
		newContent = strings.Replace(newContent, patch.OldCode, patch.NewCode, 1)
	}

	// Apply line-based patches in reverse order
	if len(linePatches) > 0 {
		lines := strings.Split(newContent, "\n")
		// Sort by StartLine descending
		for i := len(linePatches) - 1; i >= 0; i-- {
			patch := linePatches[i]
			lines = applyLinePatch(lines, patch)
		}
		newContent = strings.Join(lines, "\n")
	}

	// Get original file permissions
	info, err := os.Stat(fc.FilePath)
	if err != nil {
		return err
	}

	return os.WriteFile(fc.FilePath, []byte(newContent), info.Mode())
}

func applyLinePatch(lines []string, patch Patch) []string {
	// Convert to 0-indexed
	start := patch.StartLine - 1
	end := patch.EndLine

	// Replace lines
	newLines := strings.Split(patch.NewCode, "\n")

	result := make([]string, 0, len(lines)-end+start+len(newLines))
	result = append(result, lines[:start]...)
	result = append(result, newLines...)
	if end < len(lines) {
		result = append(result, lines[end:]...)
	}

	return result
}

func (a *MultiFileApplier) rollbackChanges(changes []FileChange) {
	for i := len(changes) - 1; i >= 0; i-- {
		fc := changes[i]
		if fc.BackupPath != "" {
			_ = a.restoreBackup(fc.BackupPath, fc.FilePath)
		}
	}
}

func (a *MultiFileApplier) restoreBackup(backupPath, originalPath string) error {
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}

	// Get original permissions if file exists
	mode := os.FileMode(0644)
	if info, err := os.Stat(originalPath); err == nil {
		mode = info.Mode()
	}

	return os.WriteFile(originalPath, content, mode)
}

// RollbackMultiFileChange rolls back a previously applied change.
func (a *MultiFileApplier) RollbackMultiFileChange(change *MultiFileChange) error {
	var errs []string
	for _, fc := range change.Changes {
		if fc.BackupPath == "" {
			errs = append(errs, fmt.Sprintf("%s: no backup path", fc.FilePath))
			continue
		}
		if err := a.restoreBackup(fc.BackupPath, fc.FilePath); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", fc.FilePath, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("rollback errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// CleanupBackups removes backup files for a change.
func (a *MultiFileApplier) CleanupBackups(change *MultiFileChange) {
	for _, fc := range change.Changes {
		if fc.BackupPath != "" {
			_ = os.Remove(fc.BackupPath)
		}
	}
}

// ConvertFixProposalsToMultiFileChange converts FixProposals to a MultiFileChange.
func ConvertFixProposalsToMultiFileChange(proposals []FixProposal, description string) *MultiFileChange {
	// Group by file
	fileChanges := make(map[string]*FileChange)

	for _, p := range proposals {
		if !p.Actionable || p.FilePath == "" {
			continue
		}

		fc, ok := fileChanges[p.FilePath]
		if !ok {
			fc = &FileChange{FilePath: p.FilePath}
			fileChanges[p.FilePath] = fc
		}

		fc.Patches = append(fc.Patches, Patch{
			OldCode: p.OldCode,
			NewCode: p.NewCode,
		})
	}

	// Convert map to slice
	changes := make([]FileChange, 0, len(fileChanges))
	for _, fc := range fileChanges {
		changes = append(changes, *fc)
	}

	return &MultiFileChange{
		ID:          fmt.Sprintf("fix_%d", time.Now().Unix()),
		Description: description,
		Changes:     changes,
	}
}
