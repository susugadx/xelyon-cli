package search

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/filefilter"
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

type parsedSearchLine struct {
	FilePath string
	LineNum  int
	Line     string
	IsMatch  bool
	Type     MatchType
}

type searchResultCollector struct {
	fileMap         map[string]*SearchResult
	fileOrder       []string
	totalMatches    int
	maxTotalMatches int
}

func newSearchResultCollector(maxTotalMatches int) *searchResultCollector {
	return &searchResultCollector{
		fileMap:         make(map[string]*SearchResult),
		maxTotalMatches: maxTotalMatches,
	}
}

func (c *searchResultCollector) ensureFile(filePath string) {
	if _, exists := c.fileMap[filePath]; exists {
		return
	}
	c.fileMap[filePath] = &SearchResult{FilePath: filePath}
	c.fileOrder = append(c.fileOrder, filePath)
}

func (c *searchResultCollector) addLine(line parsedSearchLine) bool {
	c.ensureFile(line.FilePath)
	sr := c.fileMap[line.FilePath]
	sr.Matches = append(sr.Matches, Match{
		LineNum: line.LineNum,
		Line:    line.Line,
		IsMatch: line.IsMatch,
		Type:    line.Type,
	})
	if !line.IsMatch {
		return false
	}

	sr.MatchCount++
	c.totalMatches++
	return c.maxTotalMatches > 0 && c.totalMatches >= c.maxTotalMatches
}

func (c *searchResultCollector) results() []SearchResult {
	results := make([]SearchResult, 0, len(c.fileOrder))
	for _, fp := range c.fileOrder {
		results = append(results, *c.fileMap[fp])
	}
	return results
}

func parseRipgrepJSON(output string, maxTotalMatches int) []SearchResult {
	if output == "" {
		return nil
	}

	collector := newSearchResultCollector(maxTotalMatches)
	var currentFile string

	scanSearchOutputLines(output, func(line string) bool {
		entry, ok := decodeRipgrepJSONEntry(line)
		if !ok {
			return false
		}
		var shouldStop bool
		currentFile, shouldStop = appendRipgrepEntry(entry, currentFile, collector)
		return shouldStop
	})

	return collector.results()
}

func appendRipgrepEntry(entry rgJSONLine, currentFile string, collector *searchResultCollector) (string, bool) {
	switch entry.Type {
	case "begin":
		if data, ok := decodeRipgrepBeginData(entry); ok {
			currentFile = normalizeSearchResultFilePath(data.Path.Text)
			collector.ensureFile(currentFile)
		}
		return currentFile, false
	case "match":
		if data, ok := decodeRipgrepMatchData(entry); ok {
			record := buildRipgrepParsedLine(currentFile, data, true)
			return currentFile, collector.addLine(record)
		}
		return currentFile, false
	case "context":
		if data, ok := decodeRipgrepMatchData(entry); ok {
			collector.addLine(buildRipgrepParsedLine(currentFile, data, false))
		}
		return currentFile, false
	default:
		return currentFile, false
	}
}

func decodeRipgrepJSONEntry(line string) (rgJSONLine, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return rgJSONLine{}, false
	}
	var entry rgJSONLine
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		return rgJSONLine{}, false
	}
	return entry, true
}

func decodeRipgrepBeginData(entry rgJSONLine) (rgBeginData, bool) {
	var data rgBeginData
	if err := json.Unmarshal(entry.Data, &data); err != nil {
		return rgBeginData{}, false
	}
	return data, true
}

func decodeRipgrepMatchData(entry rgJSONLine) (rgMatchData, bool) {
	var data rgMatchData
	if err := json.Unmarshal(entry.Data, &data); err != nil {
		return rgMatchData{}, false
	}
	return data, true
}

func buildRipgrepParsedLine(currentFile string, data rgMatchData, isMatch bool) parsedSearchLine {
	filePath := currentFile
	if data.Path.Text != "" {
		filePath = normalizeSearchResultFilePath(data.Path.Text)
	}
	lineText := strings.TrimRight(data.Lines.Text, "\n")
	matchType := MatchTypeRef
	if isMatch {
		matchType = classifyMatch(lineText)
	}
	return parsedSearchLine{
		FilePath: filePath,
		LineNum:  data.LineNumber,
		Line:     lineText,
		IsMatch:  isMatch,
		Type:     matchType,
	}
}

func parseGrepOutput(output string, maxTotalMatches int) []SearchResult {
	if output == "" {
		return nil
	}

	collector := newSearchResultCollector(maxTotalMatches)
	scanSearchOutputLines(output, func(line string) bool {
		return appendGrepParsedLine(line, collector)
	})

	return collector.results()
}

func scanSearchOutputLines(output string, consume func(line string) bool) {
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if consume(line) {
			return
		}
	}
}

func appendGrepParsedLine(line string, collector *searchResultCollector) bool {
	if line == "--" {
		return false
	}
	record, ok := decodeGrepParsedLine(line)
	if !ok {
		return false
	}
	return collector.addLine(record)
}

func decodeGrepParsedLine(line string) (parsedSearchLine, bool) {
	filePath, lineNum, content, isMatch := parseGrepLine(line)
	if filePath == "" {
		return parsedSearchLine{}, false
	}
	matchType := MatchTypeRef
	if isMatch {
		matchType = classifyMatch(content)
	}
	return parsedSearchLine{
		FilePath: filePath,
		LineNum:  lineNum,
		Line:     content,
		IsMatch:  isMatch,
		Type:     matchType,
	}, true
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

		filePath = normalizeSearchResultFilePath(line[:pos])
		lineNum = n
		content = line[numEnd+len(sep):]
		return filePath, lineNum, content, true
	}
}

func normalizeSearchResultFilePath(filePath string) string {
	return filefilter.CleanPath(filePath)
}
