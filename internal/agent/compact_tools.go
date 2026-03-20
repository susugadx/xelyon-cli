package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ── 閾値定数 ──

const (
	// compactReadFileMinLines は broad read_file 結果を圧縮する最小行数。
	// アウトライン（--- Signatures --- を含む）は既にコンパクトなのでスキップ。
	compactReadFileMinLines = 120

	// compactReadFileHeadLines は圧縮時に残す先頭行数。
	compactReadFileHeadLines = 40

	// compactReadFileTailLines は圧縮時に残す末尾行数。
	compactReadFileTailLines = 20

	// compactInspectSymbolMinLines は inspect_symbol 結果を圧縮する最小行数。
	compactInspectSymbolMinLines = 80

	// compactInspectBodyMaxLines は圧縮時にシンボル本体として残す最大行数。
	compactInspectBodyMaxLines = 25

	// compactListDirMinLines は list_dir 結果を圧縮する最小行数。
	compactListDirMinLines = 60

	// compactListDirKeepLines は list_dir 圧縮時に残す行数（summary + root level）。
	compactListDirKeepLines = 25
)

// guidanceTag は readTracker が追記する guidance のマーカー。
const guidanceTag = "\n\n[GUIDANCE]"

// ── read_file ──

// isTargetedRead は read_file 呼び出しが targeted read（line range 指定）か判定する。
// targeted read は次の str_replace に必要な正確な内容を含むため、圧縮しない。
// batch mode（paths 引数）では、1つでも "path:line-range" 形式が含まれていたら targeted と見なす。
func isTargetedRead(args map[string]string) bool {
	// single read mode
	if _, ok := args["start_line"]; ok {
		return true
	}
	if _, ok := args["end_line"]; ok {
		return true
	}

	// batch read mode: paths に range 指定が1つでもあれば targeted
	if pathsJSON := args["paths"]; pathsJSON != "" {
		return pathsHasRange(pathsJSON)
	}
	return false
}

// pathsHasRange は paths JSON 配列内に "path:N" または "path:N-M" 形式が含まれるか判定する。
func pathsHasRange(pathsJSON string) bool {
	var paths []string
	if err := json.Unmarshal([]byte(pathsJSON), &paths); err != nil {
		return false
	}
	for _, entry := range paths {
		// find the last colon (Windows path safety: C:\...)
		lastColon := strings.LastIndex(entry, ":")
		if lastColon < 0 {
			continue
		}
		suffix := entry[lastColon+1:]
		if suffix == "" {
			continue
		}
		// check if suffix looks like a number or number-range
		if isDigitOrRange(suffix) {
			return true
		}
	}
	return false
}

// isDigitOrRange は文字列が "123" または "10-20" 形式か判定する。
func isDigitOrRange(s string) bool {
	dashSeen := false
	for i, c := range s {
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '-' && !dashSeen && i > 0 && i < len(s)-1 {
			dashSeen = true
			continue
		}
		return false
	}
	return len(s) > 0
}

// extractGuidance は result 末尾の [GUIDANCE] ブロックを分離して返す。
// guidance が見つからない場合は body=result, guidance="" を返す。
func extractGuidance(result string) (body, guidance string) {
	idx := strings.Index(result, guidanceTag)
	if idx < 0 {
		return result, ""
	}
	return result[:idx], result[idx:]
}

// compactReadFile は read_file の結果を history 格納用に圧縮する。
// targeted read（start_line / end_line 指定）は圧縮しない。
// アウトライン形式（"--- Signatures ---" を含む）は既にコンパクトなのでそのまま返す。
// broad read（全文読み込み）のみ head + tail + 行数サマリーに変換する。
// [GUIDANCE] が付いている場合は圧縮後も保持する。
func compactReadFile(args map[string]string, result string) string {
	// targeted read は圧縮しない（str_replace に必要な exact content を保持）
	if isTargetedRead(args) {
		return result
	}

	// [GUIDANCE] を分離して最後に再追記する
	body, guidance := extractGuidance(result)

	compacted := compactReadFileBody(body)
	return compacted + guidance
}

// compactReadFileBody は read_file の本体部分（guidance 除去済み）を圧縮する。
func compactReadFileBody(result string) string {
	// アウトラインは既にコンパクト
	if strings.Contains(result, "--- Signatures ---") {
		return result
	}

	lines := strings.Split(result, "\n")
	if len(lines) <= compactReadFileMinLines {
		return result
	}

	// ガイドメッセージ（末尾の "(N lines total. Use ...)" ）を保持
	guide := ""
	for i := len(lines) - 1; i >= len(lines)-3 && i >= 0; i-- {
		if strings.HasPrefix(lines[i], "(") && strings.Contains(lines[i], "lines total") {
			guide = lines[i]
			lines = lines[:i]
			break
		}
	}

	var b strings.Builder
	// head
	head := compactReadFileHeadLines
	if head > len(lines) {
		head = len(lines)
	}
	for i := 0; i < head; i++ {
		b.WriteString(lines[i])
		b.WriteByte('\n')
	}

	omitted := len(lines) - head - compactReadFileTailLines
	if omitted < 0 {
		omitted = 0
	}
	if omitted > 0 {
		fmt.Fprintf(&b, "\n[... %d lines omitted ...]\n\n", omitted)
	}

	// tail
	tailStart := len(lines) - compactReadFileTailLines
	if tailStart < head {
		tailStart = head
	}
	for i := tailStart; i < len(lines); i++ {
		b.WriteString(lines[i])
		b.WriteByte('\n')
	}

	if guide != "" {
		b.WriteString(guide)
		b.WriteByte('\n')
	}

	return b.String()
}

// ── inspect_symbol ──

// compactInspectSymbol は inspect_symbol の結果を history 格納用に圧縮する。
// ヘッダー（── kind name (L-L) in file ──）と body の先頭部分を保持し、
// Callers/References/Tests のカウントサマリーのみ残す。
func compactInspectSymbol(result string) string {
	lines := strings.Split(result, "\n")
	if len(lines) <= compactInspectSymbolMinLines {
		return result
	}

	// "Multiple symbols matched" はそのまま返す（既にコンパクト）
	if strings.HasPrefix(result, "Multiple symbols matched") {
		return result
	}

	var b strings.Builder
	type section struct {
		name  string
		count int
	}
	var sections []section

	bodyLines := 0
	bodyDone := false
	inSection := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// ヘッダー行（── で始まる）
		if strings.HasPrefix(trimmed, "──") {
			b.WriteString(line)
			b.WriteByte('\n')
			continue
		}

		// セクション検出
		if strings.HasPrefix(trimmed, "Callers") {
			bodyDone = true
			inSection = "Callers"
			sections = append(sections, section{name: "Callers"})
			b.WriteString(line)
			b.WriteByte('\n')
			continue
		}
		if strings.HasPrefix(trimmed, "References") {
			bodyDone = true
			inSection = "References"
			sections = append(sections, section{name: "References"})
			b.WriteString(line)
			b.WriteByte('\n')
			continue
		}
		if strings.HasPrefix(trimmed, "Related tests") {
			bodyDone = true
			inSection = "Related tests"
			sections = append(sections, section{name: "Related tests"})
			b.WriteString(line)
			b.WriteByte('\n')
			continue
		}

		// Warning / Note 行
		if strings.HasPrefix(trimmed, "Warning:") || strings.HasPrefix(trimmed, "Note:") {
			b.WriteString(line)
			b.WriteByte('\n')
			continue
		}

		// body 部分（セクション開始前）
		if !bodyDone {
			bodyLines++
			if bodyLines <= compactInspectBodyMaxLines {
				b.WriteString(line)
				b.WriteByte('\n')
			}
			// bodyLines > compactInspectBodyMaxLines の行は省略（最後にまとめてマーカー出力）
			continue
		}

		// セクション内の詳細行: "  - " で始まるものをカウント
		if inSection != "" && strings.HasPrefix(trimmed, "- ") {
			if len(sections) > 0 {
				sections[len(sections)-1].count++
			}
			continue
		}

		// "(+ more ...)" 行はそのまま保持
		if strings.HasPrefix(trimmed, "(+ more") {
			b.WriteString(line)
			b.WriteByte('\n')
			continue
		}
	}

	// body が切り詰められた場合
	if bodyLines > compactInspectBodyMaxLines {
		fmt.Fprintf(&b, "  [... %d more body lines omitted ...]\n", bodyLines-compactInspectBodyMaxLines)
	}

	// セクション詳細の省略通知
	for _, sec := range sections {
		if sec.count > 0 {
			fmt.Fprintf(&b, "  [%s: %d entries shown in header, details omitted]\n", sec.name, sec.count)
		}
	}

	return b.String()
}

// ── list_dir ──

// compactListDir は list_dir の結果を history 格納用に圧縮する。
// summary 行と root レベルの dirs/files を保持し、深いサブツリーの詳細を省略する。
func compactListDir(result string) string {
	lines := strings.Split(result, "\n")
	if len(lines) <= compactListDirMinLines {
		return result
	}

	var b strings.Builder
	kept := 0
	for _, line := range lines {
		if kept >= compactListDirKeepLines {
			break
		}
		b.WriteString(line)
		b.WriteByte('\n')
		kept++
	}

	omitted := len(lines) - kept
	if omitted > 0 {
		fmt.Fprintf(&b, "[... %d subtree lines omitted. Use list_dir with specific path for details]\n", omitted)
	}

	return b.String()
}
