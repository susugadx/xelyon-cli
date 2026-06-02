package evidence

import (
	"sort"
	"strings"
)

const (
	reviewDiffEvidenceSourceUnstaged = "unstaged"
	reviewDiffEvidenceSourceStaged   = "staged"
)

func buildReviewChangedFiles(unstagedEntries, stagedEntries []reviewNameStatusEntry) []ReviewChangedFile {
	byPath := make(map[string]*ReviewChangedFile)
	applyReviewNameStatusEntries(byPath, unstagedEntries, false)
	applyReviewNameStatusEntries(byPath, stagedEntries, true)

	files := make([]ReviewChangedFile, 0, len(byPath))
	for _, file := range byPath {
		files = append(files, *file)
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].Path != files[j].Path {
			return files[i].Path < files[j].Path
		}
		if files[i].OldPath != files[j].OldPath {
			return files[i].OldPath < files[j].OldPath
		}
		return files[i].Status < files[j].Status
	})
	return files
}

func applyReviewNameStatusEntries(byPath map[string]*ReviewChangedFile, entries []reviewNameStatusEntry, staged bool) {
	for _, entry := range entries {
		file := byPath[entry.Path]
		if file == nil {
			file = &ReviewChangedFile{
				Path:    entry.Path,
				OldPath: entry.OldPath,
				Status:  entry.Status,
			}
			byPath[entry.Path] = file
		}
		if file.OldPath == "" {
			file.OldPath = entry.OldPath
		}
		file.Status = mergeReviewChangedFileStatus(file.Status, entry.Status)
		if staged {
			file.Staged = true
		} else {
			file.Unstaged = true
		}
	}
}

func mergeReviewChangedFileStatus(existing, next string) string {
	if existing == "" || existing == next {
		return next
	}
	for _, part := range strings.Split(existing, "/") {
		if part == next {
			return existing
		}
	}
	return existing + "/" + next
}
