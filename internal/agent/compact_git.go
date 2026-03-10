package agent

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const (
	// git/CI系閾値
	gitDiffMaxLines   = 100
	gitLogMaxEntries  = 10
	gitLogKeepEntries = 5
	gitShowMaxLines   = 80
	gitBlameMaxLines  = 50
	ghRunLogMaxLines  = 100
	ghRunLogFailTail  = 20
)

// ── コマンド判定 ──

// isGitDiff は git diff コマンドか判定する。--stat 付きは既にサマリー形式のため除外。
func isGitDiff(command string) bool {
	cmd := strings.TrimSpace(command)
	return strings.HasPrefix(cmd, "git diff") && !strings.Contains(cmd, "--stat") &&
		!strings.Contains(cmd, "--name-only") && !strings.Contains(cmd, "--name-status")
}

// isGitLog は git log コマンドか判定する。
func isGitLog(command string) bool {
	return strings.HasPrefix(strings.TrimSpace(command), "git log")
}

// isGitShow は git show コマンドか判定する。
func isGitShow(command string) bool {
	return strings.HasPrefix(strings.TrimSpace(command), "git show")
}

// isGitBlame は git blame コマンドか判定する。
func isGitBlame(command string) bool {
	return strings.HasPrefix(strings.TrimSpace(command), "git blame")
}

// isGHRunLog は gh run view --log コマンドか判定する。
func isGHRunLog(command string) bool {
	cmd := strings.TrimSpace(command)
	return strings.Contains(cmd, "gh run view") && strings.Contains(cmd, "--log")
}

// ── git diff ──

type diffFileStat struct {
	path    string
	added   int
	deleted int
	minLine int // 変更領域の最小行番号（0=不明）
	maxLine int // 変更領域の最大行番号
	newFile bool
}

// compactGitDiff は git diff の出力を構造化サマリーに変換する。
// gitDiffMaxLines 行以下はそのまま返す。
func compactGitDiff(result string) string {
	lines := strings.Split(result, "\n")
	if len(lines) <= gitDiffMaxLines {
		return result
	}

	files := parseDiffStats(lines)
	if len(files) == 0 {
		return result
	}

	var b strings.Builder
	fmt.Fprintf(&b, "git diff: %d files changed\n", len(files))
	for _, f := range files {
		fmt.Fprintf(&b, "  %s (+%d -%d", f.path, f.added, f.deleted)
		if f.newFile {
			b.WriteString(", new file")
		} else if f.minLine > 0 {
			fmt.Fprintf(&b, ", L%d-%d", f.minLine, f.maxLine)
		}
		b.WriteString(")\n")
	}
	b.WriteString("Use 'git diff <file>' for specific file")
	return b.String()
}

// parseDiffStats は unified diff 出力からファイル別の変更統計を抽出する。
func parseDiffStats(lines []string) []diffFileStat {
	var files []diffFileStat
	var cur *diffFileStat

	flush := func() {
		if cur != nil {
			files = append(files, *cur)
			cur = nil
		}
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			flush()
			cur = &diffFileStat{path: extractDiffPath(line)}
			continue
		}
		if cur == nil {
			continue
		}
		if strings.HasPrefix(line, "new file mode") {
			cur.newFile = true
			continue
		}
		if strings.HasPrefix(line, "@@ ") {
			newStart, newCount := parseHunkHeader(line)
			if newStart > 0 {
				end := newStart + newCount - 1
				if end < newStart {
					end = newStart
				}
				if cur.minLine == 0 || newStart < cur.minLine {
					cur.minLine = newStart
				}
				if end > cur.maxLine {
					cur.maxLine = end
				}
			}
			continue
		}
		if len(line) > 0 && line[0] == '+' && !strings.HasPrefix(line, "+++") {
			cur.added++
		} else if len(line) > 0 && line[0] == '-' && !strings.HasPrefix(line, "---") {
			cur.deleted++
		}
	}
	flush()
	return files
}

// extractDiffPath は "diff --git a/<path> b/<path>" からファイルパスを抽出する。
func extractDiffPath(line string) string {
	if parts := strings.SplitN(line, " b/", 2); len(parts) == 2 {
		return parts[1]
	}
	if after, found := strings.CutPrefix(line, "diff --git a/"); found {
		if sp := strings.Index(after, " "); sp >= 0 {
			return after[:sp]
		}
		return after
	}
	return line
}

// parseHunkHeader は "@@ -old,len +new,len @@" から new file の start, count を抽出する。
func parseHunkHeader(line string) (newStart, newCount int) {
	idx := strings.Index(line, " +")
	if idx < 0 {
		return 0, 0
	}
	rest := line[idx+2:]
	end := strings.Index(rest, " ")
	if end < 0 {
		return 0, 0
	}
	chunk := rest[:end]

	if comma := strings.IndexByte(chunk, ','); comma >= 0 {
		newStart, _ = strconv.Atoi(chunk[:comma])
		newCount, _ = strconv.Atoi(chunk[comma+1:])
	} else {
		newStart, _ = strconv.Atoi(chunk)
		newCount = 1
	}
	return
}

// ── git log ──

type commitEntry struct {
	hash string
	msg  string
}

// compactGitLog は git log の出力を最新N件サマリーに変換する。
// gitLogMaxEntries 件以下はそのまま返す。
func compactGitLog(result string) string {
	lines := strings.Split(result, "\n")

	// フォーマット検出: 先頭行が "commit " → 標準形式、それ以外 → oneline
	if len(lines) > 0 && strings.HasPrefix(lines[0], "commit ") {
		return compactGitLogStandard(result, lines)
	}
	return compactGitLogOneline(result, lines)
}

// compactGitLogStandard は標準形式の git log を圧縮する。
func compactGitLogStandard(result string, lines []string) string {
	var entries []commitEntry
	var curHash string

	for _, line := range lines {
		if strings.HasPrefix(line, "commit ") {
			curHash = strings.TrimSpace(line[7:])
			// "commit <hash> (HEAD -> main)" のような付加情報を除去
			if sp := strings.IndexByte(curHash, ' '); sp > 0 {
				curHash = curHash[:sp]
			}
			entries = append(entries, commitEntry{hash: curHash})
			continue
		}
		if len(entries) == 0 {
			continue
		}
		last := &entries[len(entries)-1]
		if last.msg != "" {
			continue
		}
		// Author/Date/Merge 行はスキップ
		if strings.HasPrefix(line, "Author:") || strings.HasPrefix(line, "Date:") ||
			strings.HasPrefix(line, "Merge:") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			last.msg = trimmed
		}
	}

	if len(entries) <= gitLogMaxEntries {
		return result
	}

	return formatLogSummary(entries)
}

// compactGitLogOneline は --oneline 形式の git log を圧縮する。
func compactGitLogOneline(result string, lines []string) string {
	var nonEmpty []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmpty = append(nonEmpty, line)
		}
	}

	if len(nonEmpty) <= gitLogMaxEntries {
		return result
	}

	var b strings.Builder
	fmt.Fprintf(&b, "git log: showing %d of %d commits\n", gitLogKeepEntries, len(nonEmpty))
	for i := range min(gitLogKeepEntries, len(nonEmpty)) {
		fmt.Fprintf(&b, "  %s\n", nonEmpty[i])
	}
	b.WriteString("Use 'git show <hash>' for commit details")
	return b.String()
}

// formatLogSummary はコミットエントリ群からサマリー文字列を生成する。
func formatLogSummary(entries []commitEntry) string {
	var b strings.Builder
	fmt.Fprintf(&b, "git log: showing %d of %d commits\n", gitLogKeepEntries, len(entries))
	for i := range min(gitLogKeepEntries, len(entries)) {
		h := entries[i].hash
		if len(h) > 7 {
			h = h[:7]
		}
		fmt.Fprintf(&b, "  %s %s\n", h, entries[i].msg)
	}
	b.WriteString("Use 'git show <hash>' for commit details")
	return b.String()
}

// ── git show ──

// compactGitShow は git show の出力を構造化サマリーに変換する。
// gitShowMaxLines 行以下はそのまま返す。
// ヘッダー（commit/Author/Date/メッセージ）を保持し、diff部分をファイル一覧に圧縮する。
func compactGitShow(result string) string {
	lines := strings.Split(result, "\n")
	if len(lines) <= gitShowMaxLines {
		return result
	}

	// ヘッダーとdiffに分離
	var header []string
	var diffLines []string
	inDiff := false

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			inDiff = true
		}
		if inDiff {
			diffLines = append(diffLines, line)
		} else {
			header = append(header, line)
		}
	}

	var b strings.Builder
	for _, line := range header {
		b.WriteString(line)
		b.WriteByte('\n')
	}

	if len(diffLines) > 0 {
		files := parseDiffStats(diffLines)
		fmt.Fprintf(&b, "%d files changed:\n", len(files))
		for _, f := range files {
			fmt.Fprintf(&b, "  %s (+%d -%d)\n", f.path, f.added, f.deleted)
		}

		// ヘッダーからコミットハッシュを抽出
		hash := extractCommitHash(header)
		if hash != "" {
			fmt.Fprintf(&b, "Use 'git diff %s -- <file>' for specific file", hash)
		}
	}

	return b.String()
}

// extractCommitHash はヘッダー行群からコミットハッシュ（先頭7文字）を抽出する。
func extractCommitHash(header []string) string {
	for _, line := range header {
		if strings.HasPrefix(line, "commit ") {
			hash := strings.TrimSpace(line[7:])
			if sp := strings.IndexByte(hash, ' '); sp > 0 {
				hash = hash[:sp]
			}
			if len(hash) > 7 {
				hash = hash[:7]
			}
			return hash
		}
	}
	return ""
}

// ── git blame ──

type blameAuthorStat struct {
	name  string
	lines int
}

// compactGitBlame は git blame の出力を著者別サマリーに変換する。
// gitBlameMaxLines 行以下はそのまま返す。
func compactGitBlame(result string) string {
	lines := strings.Split(result, "\n")
	// 末尾の空行を除外してカウント
	nonEmpty := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmpty++
		}
	}
	if nonEmpty <= gitBlameMaxLines {
		return result
	}

	// 著者ごとの行数を集計
	authorCount := make(map[string]int)
	for _, line := range lines {
		author := extractBlameAuthor(line)
		if author != "" {
			authorCount[author]++
		}
	}
	if len(authorCount) == 0 {
		return result
	}

	// 行数降順でソート
	stats := make([]blameAuthorStat, 0, len(authorCount))
	for name, count := range authorCount {
		stats = append(stats, blameAuthorStat{name: name, lines: count})
	}
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].lines > stats[j].lines
	})

	var b strings.Builder
	fmt.Fprintf(&b, "git blame summary (%d lines):\n", nonEmpty)
	for _, s := range stats {
		pct := s.lines * 100 / nonEmpty
		fmt.Fprintf(&b, "  %-20s %4d lines (%d%%)\n", s.name, s.lines, pct)
	}
	b.WriteString("Use 'git blame -L <start>,<end> <file>' for specific range")
	return b.String()
}

// extractBlameAuthor は git blame の行から著者名を抽出する。
// 形式: "<hash> (<author> <YYYY-MM-DD HH:MM:SS <tz> <linenum>) <content>"
func extractBlameAuthor(line string) string {
	openParen := strings.IndexByte(line, '(')
	closeParen := strings.IndexByte(line, ')')
	if openParen < 0 || closeParen <= openParen {
		return ""
	}
	inner := line[openParen+1 : closeParen]

	// 日付パターン (YYYY-) を探して著者名を分離
	for i := 0; i <= len(inner)-5; i++ {
		if inner[i] >= '1' && inner[i] <= '2' &&
			inner[i+1] >= '0' && inner[i+1] <= '9' &&
			inner[i+2] >= '0' && inner[i+2] <= '9' &&
			inner[i+3] >= '0' && inner[i+3] <= '9' &&
			inner[i+4] == '-' {
			return strings.TrimSpace(inner[:i])
		}
	}
	return ""
}

// ── gh run view --log ──

type ghRunStep struct {
	name   string
	lines  []string
	failed bool
}

// compactGHRunLog は gh run view --log の出力をステップ一覧に変換する。
// ghRunLogMaxLines 行以下はそのまま返す。
func compactGHRunLog(result string) string {
	lines := strings.Split(result, "\n")
	if len(lines) <= ghRunLogMaxLines {
		return result
	}

	steps := parseGHRunSteps(lines)
	if len(steps) == 0 {
		// 構造を検出できない場合はtailで返す
		return fmt.Sprintf("gh run log: %d lines total\n%s",
			len(lines), tailLines(result, ghRunLogFailTail))
	}

	var b strings.Builder
	passed, failed := 0, 0
	var failedStep *ghRunStep
	for i := range steps {
		if steps[i].failed {
			failed++
			failedStep = &steps[i]
		} else {
			passed++
		}
	}

	fmt.Fprintf(&b, "gh run: %d steps (%d passed, %d failed)\n", len(steps), passed, failed)
	for _, s := range steps {
		status := "✓"
		if s.failed {
			status = "✗"
		}
		fmt.Fprintf(&b, "  %s %s (%d lines)\n", status, s.name, len(s.lines))
	}

	if failedStep != nil && len(failedStep.lines) > 0 {
		n := min(ghRunLogFailTail, len(failedStep.lines))
		start := len(failedStep.lines) - n
		fmt.Fprintf(&b, "\n[last %d lines of '%s']:\n", n, failedStep.name)
		for _, line := range failedStep.lines[start:] {
			fmt.Fprintf(&b, "  %s\n", line)
		}
	}

	b.WriteString("Use 'gh run view --log --job <id>' for full job log")
	return b.String()
}

// parseGHRunSteps はタブ区切り形式のログからステップ群をパースする。
// 形式: "<job>\t<step>\t<log_line>"
func parseGHRunSteps(lines []string) []ghRunStep {
	var steps []ghRunStep
	stepMap := make(map[string]int)

	for _, line := range lines {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			continue
		}
		stepName := strings.TrimSpace(parts[1])
		if stepName == "" {
			continue
		}
		var logLine string
		if len(parts) >= 3 {
			logLine = parts[2]
		}

		idx, exists := stepMap[stepName]
		if !exists {
			idx = len(steps)
			stepMap[stepName] = idx
			steps = append(steps, ghRunStep{name: stepName})
		}
		steps[idx].lines = append(steps[idx].lines, logLine)

		if isFailureLine(logLine) {
			steps[idx].failed = true
		}
	}

	return steps
}

// isFailureLine はログ行が失敗を示すか判定する。
func isFailureLine(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "##[error]") ||
		strings.Contains(lower, "process completed with exit code") &&
			!strings.Contains(lower, "exit code 0")
}
