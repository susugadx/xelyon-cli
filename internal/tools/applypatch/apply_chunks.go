package applypatch

import (
	"sort"
	"strings"
)

type replacement struct {
	StartIndex int
	OldLen     int
	NewLines   []string
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
