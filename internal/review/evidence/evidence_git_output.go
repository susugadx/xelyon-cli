package evidence

import (
	"path/filepath"
	"strconv"
	"strings"
)

type reviewNameStatusEntry struct {
	Status  string
	OldPath string
	Path    string
}

func parseReviewEvidenceNULPaths(output string, truncated bool) []string {
	fields := splitReviewEvidenceNULFields(output, truncated)
	paths := make([]string, 0, len(fields))
	for _, field := range fields {
		if field == "" {
			continue
		}
		paths = append(paths, field)
	}
	return paths
}

func parseReviewNameStatusEntries(output string, truncated bool) []reviewNameStatusEntry {
	fields := splitReviewEvidenceNULFields(output, truncated)
	entries := make([]reviewNameStatusEntry, 0, len(fields)/2)
	for i := 0; i < len(fields); {
		status := strings.TrimSpace(fields[i])
		i++
		if status == "" {
			continue
		}

		if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
			if i+1 >= len(fields) {
				break
			}
			oldPath := normalizeReviewEvidenceDisplayPath(fields[i])
			path := normalizeReviewEvidenceDisplayPath(fields[i+1])
			i += 2
			if path == "" || path == "." {
				continue
			}
			entries = append(entries, reviewNameStatusEntry{
				Status:  status,
				OldPath: oldPath,
				Path:    path,
			})
			continue
		}

		if i >= len(fields) {
			break
		}
		path := normalizeReviewEvidenceDisplayPath(fields[i])
		i++
		if path == "" || path == "." {
			continue
		}
		entries = append(entries, reviewNameStatusEntry{
			Status: status,
			Path:   path,
		})
	}
	return entries
}

func formatReviewNameStatusEntries(entries []reviewNameStatusEntry) string {
	if len(entries) == 0 {
		return ""
	}

	var out strings.Builder
	for _, entry := range entries {
		out.WriteString(entry.Status)
		out.WriteByte('\t')
		if entry.OldPath != "" {
			out.WriteString(formatReviewNameStatusDisplayPath(entry.OldPath))
			out.WriteByte('\t')
		}
		out.WriteString(formatReviewNameStatusDisplayPath(entry.Path))
		out.WriteByte('\n')
	}
	return out.String()
}

func formatReviewNameStatusDisplayPath(path string) string {
	for i := 0; i < len(path); i++ {
		if path[i] < 0x20 || path[i] == 0x7f {
			return strconv.Quote(path)
		}
	}
	return path
}

func splitReviewEvidenceNULFields(output string, truncated bool) []string {
	if output == "" {
		return nil
	}
	if truncated && !strings.HasSuffix(output, "\x00") {
		lastNUL := strings.LastIndex(output, "\x00")
		if lastNUL < 0 {
			output = ""
		} else {
			output = output[:lastNUL+1]
		}
	}

	if output == "" {
		return nil
	}

	fields := strings.Split(output, "\x00")
	if fields[len(fields)-1] == "" {
		fields = fields[:len(fields)-1]
	}
	return fields
}

func normalizeReviewEvidenceDisplayPath(candidate string) string {
	if candidate == "" {
		return ""
	}
	cleaned := filepath.Clean(filepath.FromSlash(candidate))
	return filepath.ToSlash(cleaned)
}
