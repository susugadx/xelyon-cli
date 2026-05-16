package navigation

import (
	"bufio"
	"io"
	"strings"
)

// collectReferenceSearchResult は ripgrep の標準出力を読み取り、参照一覧を構築する。
func collectReferenceSearchResult(reader io.Reader, symbol string, filter ReferenceFilter) referenceSearchResult {
	result := referenceSearchResult{}
	if reader == nil {
		result.Incomplete = true
		return result
	}

	cache := newReferenceParseCache()
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, ripgrepScannerInitialBufferSize), ripgrepScannerMaxBufferSize)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parsed, ok := parseRipgrepReferenceLine(line)
		if !ok {
			continue
		}
		if filter != nil && !filter(referenceForPreClassificationFilter(parsed)) {
			continue
		}

		classification := classifyParsedReferenceLine(parsed, symbol, cache)
		classification = applySnippetCompletionHints(parsed.Snippet, symbol, classification)
		ref := buildReferenceFromParsedLine(parsed, classification)
		result.Refs = append(result.Refs, *ref)

		// 上限+1件目を検出したら truncated=true にし、先頭上限件のみ保持して早期停止。
		if len(result.Refs) > maxRipgrepResults {
			result.Truncated = true
			result.Refs = result.Refs[:maxRipgrepResults]
			result.StopRequested = true
			break
		}
	}

	if err := scanner.Err(); err != nil {
		result.Incomplete = true
	}

	return result
}

// runReferenceSearch は参照ストリームの読み取りと終了待機をまとめて処理する。
func runReferenceSearch(reader io.Reader, symbol string, cancel func(), wait func() error, referenceFilter ReferenceFilter) ([]Reference, bool, bool) {
	result := collectReferenceSearchResult(reader, symbol, referenceFilter)
	if result.StopRequested && cancel != nil {
		cancel()
	}
	if wait != nil {
		if err := wait(); err != nil && !result.StopRequested {
			result.Incomplete = true
		}
	}
	return result.Refs, result.Truncated, result.Incomplete
}

func referenceForPreClassificationFilter(parsed parsedRipgrepReferenceLine) Reference {
	return Reference{
		File:         parsed.RelPath,
		ResolvedPath: cleanNavigationResolvedPath(parsed.AbsPath),
		Line:         parsed.Line,
		Snippet:      parsed.Snippet,
		IsTest:       parsed.IsTest,
	}
}
