package refactor

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// DetectLargeFiles detects files that exceed the maximum line count.
func DetectLargeFiles(files []string, maxLines int) []RefactorProposal {
	var proposals []RefactorProposal

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		lines := strings.Count(string(content), "\n") + 1
		if lines > maxLines {
			proposals = append(proposals, RefactorProposal{
				ID:           fmt.Sprintf("split-file:%s", file),
				Type:         RefactorSplitFile,
				Description:  fmt.Sprintf("File has %d lines (max: %d). Consider splitting into smaller modules.", lines, maxLines),
				FilePath:     file,
				CurrentLines: lines,
				TargetLines:  maxLines,
				Confidence:   0.8,
				Actionable:   false, // Requires AI to generate split plan
			})
		}
	}

	return proposals
}

// DetectLongFunctions detects functions that exceed the maximum line count.
func DetectLongFunctions(files []string, maxLines int) []RefactorProposal {
	var proposals []RefactorProposal

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		ext := strings.ToLower(filepath.Ext(file))
		funcs := extractFunctions(string(content), ext)

		for _, f := range funcs {
			if f.lines > maxLines {
				proposals = append(proposals, RefactorProposal{
					ID:           fmt.Sprintf("extract-method:%s:%s", file, f.name),
					Type:         RefactorExtractMethod,
					Description:  fmt.Sprintf("Function '%s' has %d lines (max: %d). Consider extracting sub-functions.", f.name, f.lines, maxLines),
					FilePath:     file,
					FunctionName: f.name,
					LineStart:    f.lineStart,
					LineEnd:      f.lineEnd,
					CurrentLines: f.lines,
					TargetLines:  maxLines,
					Confidence:   0.7,
					Actionable:   false, // Requires AI to generate extraction
				})
			}
		}
	}

	return proposals
}

// funcInfo holds information about a function.
type funcInfo struct {
	name      string
	lineStart int
	lineEnd   int
	lines     int
}

// extractFunctions extracts function information from source code.
func extractFunctions(content string, ext string) []funcInfo {
	var funcs []funcInfo
	lines := strings.Split(content, "\n")

	var funcPattern *regexp.Regexp
	switch ext {
	case ".go":
		funcPattern = regexp.MustCompile(`^func\s+(?:\([^)]+\)\s+)?(\w+)\s*\(`)
	case ".js", ".ts", ".jsx", ".tsx":
		funcPattern = regexp.MustCompile(`(?:function\s+(\w+)|(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?(?:function|\([^)]*\)\s*=>|\w+\s*=>))`)
	case ".py":
		funcPattern = regexp.MustCompile(`^def\s+(\w+)\s*\(`)
	case ".java":
		funcPattern = regexp.MustCompile(`(?:public|private|protected)?\s*(?:static\s+)?(?:\w+\s+)?(\w+)\s*\([^)]*\)\s*(?:throws\s+\w+)?\s*\{`)
	case ".rs":
		funcPattern = regexp.MustCompile(`^(?:pub\s+)?fn\s+(\w+)`)
	default:
		return nil
	}

	braceCount := 0
	inFunc := false
	var currentFunc funcInfo

	for i, line := range lines {
		lineNum := i + 1

		if !inFunc {
			matches := funcPattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				// Find the first non-empty capture group
				name := ""
				for _, m := range matches[1:] {
					if m != "" {
						name = m
						break
					}
				}
				if name != "" {
					currentFunc = funcInfo{
						name:      name,
						lineStart: lineNum,
					}
					inFunc = true
					braceCount = strings.Count(line, "{") - strings.Count(line, "}")
					if braceCount <= 0 && strings.Contains(line, "{") {
						// Single-line function
						currentFunc.lineEnd = lineNum
						currentFunc.lines = 1
						funcs = append(funcs, currentFunc)
						inFunc = false
					}
				}
			}
		} else {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount <= 0 {
				currentFunc.lineEnd = lineNum
				currentFunc.lines = currentFunc.lineEnd - currentFunc.lineStart + 1
				funcs = append(funcs, currentFunc)
				inFunc = false
			}
		}
	}

	return funcs
}

// DetectDuplicateCode detects duplicate code blocks.
func DetectDuplicateCode(files []string, minLines int) []RefactorProposal {
	var proposals []RefactorProposal

	// Build a map of code block hashes
	type blockInfo struct {
		file      string
		lineStart int
		lineEnd   int
		content   string
	}

	blockHashes := make(map[string][]blockInfo)

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")

		// Sliding window to find duplicate blocks
		for i := 0; i <= len(lines)-minLines; i++ {
			block := strings.Join(lines[i:i+minLines], "\n")
			normalized := normalizeCode(block)

			// Skip empty or trivial blocks
			if len(strings.TrimSpace(normalized)) < 50 {
				continue
			}

			hash := fmt.Sprintf("%x", md5.Sum([]byte(normalized)))
			blockHashes[hash] = append(blockHashes[hash], blockInfo{
				file:      file,
				lineStart: i + 1,
				lineEnd:   i + minLines,
				content:   block,
			})
		}
	}

	// Find duplicates
	seen := make(map[string]bool)
	for hash, blocks := range blockHashes {
		if len(blocks) < 2 {
			continue
		}

		// Create a unique key to avoid duplicate proposals
		key := fmt.Sprintf("%s:%d-%s:%d", blocks[0].file, blocks[0].lineStart, blocks[1].file, blocks[1].lineStart)
		if seen[key] {
			continue
		}
		seen[key] = true

		// Build description
		locations := make([]string, 0, len(blocks))
		for _, b := range blocks {
			locations = append(locations, fmt.Sprintf("%s:%d-%d", filepath.Base(b.file), b.lineStart, b.lineEnd))
		}

		proposals = append(proposals, RefactorProposal{
			ID:           fmt.Sprintf("dry:%s", hash[:8]),
			Type:         RefactorDRY,
			Description:  fmt.Sprintf("Duplicate code block found in %d locations: %s. Consider extracting to a shared function.", len(blocks), strings.Join(locations, ", ")),
			FilePath:     blocks[0].file,
			LineStart:    blocks[0].lineStart,
			LineEnd:      blocks[0].lineEnd,
			CurrentLines: minLines,
			Confidence:   0.6,
			Actionable:   false, // Requires manual or AI intervention
		})
	}

	return proposals
}

// normalizeCode normalizes code for comparison (removes whitespace variations).
func normalizeCode(code string) string {
	// Remove leading/trailing whitespace from each line
	lines := strings.Split(code, "\n")
	var normalized []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return strings.Join(normalized, "\n")
}

// DetectPoorNaming detects poor naming conventions.
func DetectPoorNaming(files []string) []RefactorProposal {
	var proposals []RefactorProposal

	// Patterns for poor naming
	singleLetterVar := regexp.MustCompile(`\b([a-z])\s*:?=`)
	genericNames := regexp.MustCompile(`\b(data|temp|tmp|result|ret|val|value|item|obj|thing)\b`)

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		ext := strings.ToLower(filepath.Ext(file))
		// Only check Go files for now
		if ext != ".go" {
			continue
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			lineNum := i + 1

			// Skip comments
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
				continue
			}

			// Check for single-letter variables (except loop counters i, j, k)
			matches := singleLetterVar.FindAllStringSubmatch(line, -1)
			for _, m := range matches {
				if len(m) > 1 {
					varName := m[1]
					// Skip common loop counters and error variables
					if varName == "i" || varName == "j" || varName == "k" || varName == "n" || varName == "e" {
						continue
					}
					proposals = append(proposals, RefactorProposal{
						ID:          fmt.Sprintf("rename:%s:%d:%s", file, lineNum, varName),
						Type:        RefactorRename,
						Description: fmt.Sprintf("Single-letter variable '%s' could have a more descriptive name.", varName),
						FilePath:    file,
						LineStart:   lineNum,
						Confidence:  0.5,
						Actionable:  false,
					})
				}
			}

			// Check for generic names
			if genericNames.MatchString(line) {
				matches := genericNames.FindAllString(line, -1)
				for _, name := range matches {
					// Skip if it's part of a longer identifier
					if isPartOfIdentifier(line, name) {
						continue
					}
					proposals = append(proposals, RefactorProposal{
						ID:          fmt.Sprintf("rename:%s:%d:%s", file, lineNum, name),
						Type:        RefactorRename,
						Description: fmt.Sprintf("Generic variable name '%s' could be more descriptive.", name),
						FilePath:    file,
						LineStart:   lineNum,
						Confidence:  0.4,
						Actionable:  false,
					})
				}
			}
		}
	}

	return proposals
}

// isPartOfIdentifier checks if a word is part of a larger identifier.
func isPartOfIdentifier(line, word string) bool {
	idx := strings.Index(line, word)
	if idx < 0 {
		return false
	}

	// Check character before
	if idx > 0 {
		before := rune(line[idx-1])
		if unicode.IsLetter(before) || unicode.IsDigit(before) || before == '_' {
			return true
		}
	}

	// Check character after
	endIdx := idx + len(word)
	if endIdx < len(line) {
		after := rune(line[endIdx])
		if unicode.IsLetter(after) || unicode.IsDigit(after) || after == '_' {
			return true
		}
	}

	return false
}
