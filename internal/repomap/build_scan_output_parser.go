package repomap

import (
	"path/filepath"
	"strconv"
	"strings"
)

func parseSymbolScanOutput(output string, seen map[string]map[int]struct{}) map[string][]Symbol {
	results := make(map[string][]Symbol)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}

		path := filepath.Clean(filepath.ToSlash(parts[0]))
		lineNum, convErr := strconv.Atoi(parts[1])
		if convErr != nil {
			continue
		}
		content := parts[2]
		if !matchesSymbolPattern(path, content) {
			continue
		}

		if seen[path] == nil {
			seen[path] = make(map[int]struct{})
		}
		if _, ok := seen[path][lineNum]; ok {
			continue
		}
		seen[path][lineNum] = struct{}{}

		signature := normalizeSignature(content)
		name, kind, exported := signatureMetadataForPath(path, signature)
		results[path] = append(results[path], Symbol{
			Name:      name,
			Kind:      kind,
			Line:      lineNum,
			Signature: signature,
			Exported:  exported,
		})
	}
	return results
}
