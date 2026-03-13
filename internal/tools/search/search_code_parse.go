package search

import (
	"encoding/json"
	"strconv"
	"strings"
)

// rgJSONLine は rg --json 出力の1行
type rgJSONLine struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type rgMatchData struct {
	Path       rgPath `json:"path"`
	LineNumber int    `json:"line_number"`
	Lines      rgText `json:"lines"`
}

type rgPath struct {
	Text string `json:"text"`
}

type rgText struct {
	Text string `json:"text"`
}

type rgBeginData struct {
	Path rgPath `json:"path"`
}

func parseRipgrepJSON(output string, maxTotalMatches int) []SearchResult {
	if output == "" {
		return nil
	}

	fileMap := make(map[string]*SearchResult)
	var fileOrder []string
	var currentFile string
	totalMatches := 0

	lines := strings.Split(strings.TrimSpace(output), "\n")
outer:
	for _, line := range lines {
		if line == "" {
			continue
		}

		var entry rgJSONLine
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		switch entry.Type {
		case "begin":
			var data rgBeginData
			if err := json.Unmarshal(entry.Data, &data); err == nil {
				currentFile = data.Path.Text
				if _, exists := fileMap[currentFile]; !exists {
					fileMap[currentFile] = &SearchResult{FilePath: currentFile}
					fileOrder = append(fileOrder, currentFile)
				}
			}

		case "match":
			var data rgMatchData
			if err := json.Unmarshal(entry.Data, &data); err == nil {
				filePath := currentFile
				if data.Path.Text != "" {
					filePath = data.Path.Text
				}
				if _, exists := fileMap[filePath]; !exists {
					fileMap[filePath] = &SearchResult{FilePath: filePath}
					fileOrder = append(fileOrder, filePath)
				}
				sr := fileMap[filePath]
				lineText := strings.TrimRight(data.Lines.Text, "\n")
				sr.Matches = append(sr.Matches, Match{
					LineNum: data.LineNumber,
					Line:    lineText,
					IsMatch: true,
					Type:    classifyMatch(lineText),
				})
				sr.MatchCount++
				totalMatches++
				if totalMatches >= maxTotalMatches {
					break outer
				}
			}

		case "context":
			var data rgMatchData
			if err := json.Unmarshal(entry.Data, &data); err == nil {
				filePath := currentFile
				if data.Path.Text != "" {
					filePath = data.Path.Text
				}
				if _, exists := fileMap[filePath]; !exists {
					fileMap[filePath] = &SearchResult{FilePath: filePath}
					fileOrder = append(fileOrder, filePath)
				}
				sr := fileMap[filePath]
				sr.Matches = append(sr.Matches, Match{
					LineNum: data.LineNumber,
					Line:    strings.TrimRight(data.Lines.Text, "\n"),
					IsMatch: false,
					Type:    MatchTypeRef,
				})
			}

			// "end", "summary" -> 無視
		}
	}

	var results []SearchResult
	for _, fp := range fileOrder {
		results = append(results, *fileMap[fp])
	}
	return results
}

func parseGrepOutput(output string, maxTotalMatches int) []SearchResult {
	if output == "" {
		return nil
	}

	fileMap := make(map[string]*SearchResult)
	var fileOrder []string
	totalMatches := 0

	lines := strings.Split(strings.TrimSpace(output), "\n")
outer:
	for _, line := range lines {
		if line == "--" {
			continue
		}

		filePath, lineNum, content, isMatch := parseGrepLine(line)
		if filePath == "" {
			continue
		}

		if _, exists := fileMap[filePath]; !exists {
			fileMap[filePath] = &SearchResult{FilePath: filePath}
			fileOrder = append(fileOrder, filePath)
		}

		sr := fileMap[filePath]
		matchType := MatchTypeRef
		if isMatch {
			matchType = classifyMatch(content)
		}
		sr.Matches = append(sr.Matches, Match{
			LineNum: lineNum,
			Line:    content,
			IsMatch: isMatch,
			Type:    matchType,
		})
		if isMatch {
			sr.MatchCount++
			totalMatches++
			if totalMatches >= maxTotalMatches {
				break outer
			}
		}
	}

	var results []SearchResult
	for _, fp := range fileOrder {
		results = append(results, *fileMap[fp])
	}
	return results
}

// parseGrepLine は grep の1行をパースする
// マッチ行: file.go:13:  content   (区切り :)
// コンテキスト行: file.go-12-content  (区切り -)
func parseGrepLine(line string) (filePath string, lineNum int, content string, isMatch bool) {
	if fp, ln, c, ok := tryParseGrepMatch(line, ":"); ok {
		return fp, ln, c, true
	}
	if fp, ln, c, ok := tryParseGrepMatch(line, "-"); ok {
		return fp, ln, c, false
	}
	return "", 0, "", false
}

// tryParseGrepMatch は指定セパレータで grep 行をパースする
func tryParseGrepMatch(line, sep string) (filePath string, lineNum int, content string, ok bool) {
	idx := 0
	for {
		pos := strings.Index(line[idx:], sep)
		if pos < 0 {
			return "", 0, "", false
		}
		pos += idx

		numStart := pos + len(sep)
		numEnd := numStart
		for numEnd < len(line) && line[numEnd] >= '0' && line[numEnd] <= '9' {
			numEnd++
		}
		if numEnd == numStart {
			idx = pos + len(sep)
			continue
		}

		if numEnd+len(sep) > len(line) || line[numEnd:numEnd+len(sep)] != sep {
			idx = pos + len(sep)
			continue
		}

		n, err := strconv.Atoi(line[numStart:numEnd])
		if err != nil || n <= 0 {
			idx = pos + len(sep)
			continue
		}

		filePath = line[:pos]
		lineNum = n
		content = line[numEnd+len(sep):]
		return filePath, lineNum, content, true
	}
}
