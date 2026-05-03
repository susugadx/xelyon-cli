package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type instructionFileLoadPolicy struct {
	RequireGitTracked    bool
	IncludeGitignored    bool
	GitRoot              string
	Budget               *instructionByteBudget
	AllowReadWhenUnknown bool
	RootBoundary         instructionPathBoundary
	ExpandImports        bool
	GitTrackedLookup     func(fullPath string) (tracked bool, known bool)
}

type instructionImportTraversalState struct {
	ImportStack []string
}

type instructionFileLoadOptions struct {
	FullPath     string
	DisplayLabel string
	Scope        string
	Strength     InstructionStrength
	Policy       instructionFileLoadPolicy
	Traversal    instructionImportTraversalState
}

func (opts instructionFileLoadOptions) withImportPath(importPath string) instructionFileLoadOptions {
	child := opts
	child.FullPath = importPath
	if strings.TrimSpace(importPath) != "" {
		child.DisplayLabel = importPath
	}
	child.Traversal.ImportStack = append(append([]string{}, opts.Traversal.ImportStack...), importPath)
	return child
}

type instructionLoadSkipReason string

const (
	instructionLoadSkipNone              instructionLoadSkipReason = ""
	instructionLoadSkipEmptyPath         instructionLoadSkipReason = "empty_path"
	instructionLoadSkipNotFound          instructionLoadSkipReason = "not_found"
	instructionLoadSkipOutsideRoot       instructionLoadSkipReason = "outside_root"
	instructionLoadSkipUntracked         instructionLoadSkipReason = "untracked_or_gitignored"
	instructionLoadSkipGitStatusUnknown  instructionLoadSkipReason = "git_status_unknown"
	instructionLoadSkipReadError         instructionLoadSkipReason = "read_error"
	instructionLoadSkipNoContentInBudget instructionLoadSkipReason = "no_content_in_budget"
)

type instructionLoadResult struct {
	File       InstructionFile
	Loaded     bool
	Warning    string
	Warnings   []ProjectInstructionWarning
	SkipReason instructionLoadSkipReason
}

func loadInstructionFile(opts instructionFileLoadOptions) instructionLoadResult {
	data, gitTracked, rejected, ok := loadInstructionSource(opts)
	if !ok {
		return rejected
	}

	var importWarnings []ProjectInstructionWarning
	if opts.Policy.ExpandImports {
		expanded := expandInstructionImports(opts, data)
		data = expanded.Content
		importWarnings = expanded.Warnings
	}

	result := buildInstructionLoadResult(opts, data, gitTracked)
	result.Warnings = append(result.Warnings, importWarnings...)
	return result
}

func loadInstructionSource(opts instructionFileLoadOptions) (data []byte, gitTracked bool, rejected instructionLoadResult, ok bool) {
	if rejected, ok := validateInstructionFilePath(opts); !ok {
		return nil, false, rejected, false
	}

	gitTracked, rejected, ok = resolveInstructionGitTracking(opts)
	if !ok {
		return nil, false, rejected, false
	}

	data, rejected, ok = readInstructionFileContent(opts)
	if !ok {
		return nil, false, rejected, false
	}
	return data, gitTracked, instructionLoadResult{}, true
}

func validateInstructionFilePath(opts instructionFileLoadOptions) (instructionLoadResult, bool) {
	if strings.TrimSpace(opts.FullPath) == "" {
		return instructionLoadResult{SkipReason: instructionLoadSkipEmptyPath}, false
	}
	if _, err := os.Stat(opts.FullPath); err != nil {
		return instructionLoadResult{SkipReason: instructionLoadSkipNotFound}, false
	}
	if !opts.Policy.RootBoundary.ContainsPath(opts.FullPath) {
		return instructionLoadResult{
			Warning:    "Skipped guidance outside workspace root boundary: " + opts.DisplayLabel,
			SkipReason: instructionLoadSkipOutsideRoot,
		}, false
	}
	return instructionLoadResult{}, true
}

func resolveInstructionGitTracking(opts instructionFileLoadOptions) (gitTracked bool, rejected instructionLoadResult, ok bool) {
	if !opts.Policy.RequireGitTracked {
		return false, instructionLoadResult{}, true
	}

	trackedLookup := opts.Policy.GitTrackedLookup
	if trackedLookup == nil {
		trackedLookup = func(fullPath string) (bool, bool) {
			return isGitTrackedInstructionFile(opts.Policy.GitRoot, fullPath)
		}
	}
	tracked, known := trackedLookup(opts.FullPath)
	if known {
		if !tracked {
			if !opts.Policy.IncludeGitignored {
				return false, instructionLoadResult{
					SkipReason: instructionLoadSkipUntracked,
				}, false
			}
			return false, instructionLoadResult{}, true
		}
		return true, instructionLoadResult{}, true
	}

	if !opts.Policy.AllowReadWhenUnknown {
		return false, instructionLoadResult{
			Warning:    "Skipped guidance because git tracking status is unknown: " + opts.DisplayLabel,
			SkipReason: instructionLoadSkipGitStatusUnknown,
		}, false
	}
	return false, instructionLoadResult{}, true
}

func readInstructionFileContent(opts instructionFileLoadOptions) ([]byte, instructionLoadResult, bool) {
	data, err := os.ReadFile(opts.FullPath)
	if err != nil {
		return nil, instructionLoadResult{
			Warning:    "Skipped guidance due read error: " + opts.DisplayLabel,
			SkipReason: instructionLoadSkipReadError,
		}, false
	}
	return data, instructionLoadResult{}, true
}

func buildInstructionLoadResult(opts instructionFileLoadOptions, data []byte, gitTracked bool) instructionLoadResult {
	content, truncated, consumed := applyInstructionContentLimits(data, opts.Policy.Budget)
	if consumed <= 0 && strings.TrimSpace(content) == "" {
		return instructionLoadResult{SkipReason: instructionLoadSkipNoContentInBudget}
	}

	return instructionLoadResult{
		File: InstructionFile{
			Path:       opts.FullPath,
			Label:      opts.DisplayLabel,
			Scope:      opts.Scope,
			Strength:   opts.Strength,
			Content:    content,
			Truncated:  truncated,
			GitTracked: gitTracked,
		},
		Loaded: true,
	}
}

func isLocalGuidanceFile(path string) bool {
	name := strings.ToLower(filepath.Base(strings.TrimSpace(path)))
	if name == "" {
		return false
	}
	if name == "claude.local.md" || name == "agents.local.md" {
		return true
	}
	return strings.HasSuffix(name, ".local.md")
}

func expandUserPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		if path == "~" {
			return home
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func isGitTrackedInstructionFile(gitRoot, fullPath string) (tracked bool, known bool) {
	if strings.TrimSpace(gitRoot) == "" || strings.TrimSpace(fullPath) == "" {
		return false, false
	}
	rel, err := filepath.Rel(gitRoot, fullPath)
	if err != nil {
		return false, false
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return false, false
	}
	cmd := exec.Command("git", "-C", gitRoot, "ls-files", "--error-unmatch", "--", rel)
	if err := cmd.Run(); err != nil {
		return false, true
	}
	return true, true
}
