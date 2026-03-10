package agent

import (
	"fmt"
	"strings"
	"testing"
)

// ── helpers ──

func makeDiff(fileCount, linesPerFile int) string {
	var b strings.Builder
	for i := range fileCount {
		name := fmt.Sprintf("internal/pkg%d/file%d.go", i, i)
		fmt.Fprintf(&b, "diff --git a/%s b/%s\n", name, name)
		fmt.Fprintf(&b, "index abc1234..def5678 100644\n")
		fmt.Fprintf(&b, "--- a/%s\n", name)
		fmt.Fprintf(&b, "+++ b/%s\n", name)
		fmt.Fprintf(&b, "@@ -10,5 +10,%d @@\n", linesPerFile)
		for j := range linesPerFile {
			if j%2 == 0 {
				fmt.Fprintf(&b, "+added line %d\n", j)
			} else {
				fmt.Fprintf(&b, "-removed line %d\n", j)
			}
		}
	}
	return b.String()
}

func makeStandardLog(count int) string {
	var b strings.Builder
	for i := range count {
		fmt.Fprintf(&b, "commit %040x\n", i)
		fmt.Fprintf(&b, "Author: Test User <test@example.com>\n")
		fmt.Fprintf(&b, "Date:   Mon Mar 10 00:00:00 2026 +0900\n")
		fmt.Fprintf(&b, "\n")
		fmt.Fprintf(&b, "    commit message %d\n\n", i)
	}
	return b.String()
}

func makeOnelineLog(count int) string {
	var b strings.Builder
	for i := range count {
		fmt.Fprintf(&b, "%07x fix issue %d\n", i, i)
	}
	return b.String()
}

func makeBlameLine(hash, author, date string, lineNum int, content string) string {
	return fmt.Sprintf("%s (%s %s %5d) %s", hash, author, date, lineNum, content)
}

func makeBlame(lineCount int) string {
	var b strings.Builder
	authors := []string{"Author1       ", "Author2       ", "Other Author  "}
	for i := 1; i <= lineCount; i++ {
		author := authors[i%len(authors)]
		b.WriteString(makeBlameLine(
			fmt.Sprintf("%07x", i),
			author,
			"2026-03-10 10:00:00 +0900",
			i,
			fmt.Sprintf("line content %d", i),
		))
		b.WriteByte('\n')
	}
	return b.String()
}

func makeGHRunLog(stepCount, linesPerStep int, failStep int) string {
	var b strings.Builder
	for s := range stepCount {
		stepName := fmt.Sprintf("Step %d", s)
		for l := range linesPerStep {
			logLine := fmt.Sprintf("log output line %d", l)
			if s == failStep && l == linesPerStep-1 {
				logLine = "##[error] Process completed with exit code 1"
			}
			fmt.Fprintf(&b, "build\t%s\t%s\n", stepName, logLine)
		}
	}
	return b.String()
}

// ── git diff ──

func TestCompactGitDiff_Short(t *testing.T) {
	diff := makeDiff(2, 3) // ~30行
	got := compactGitDiff(diff)
	if got != diff {
		t.Error("short diff should return as-is")
	}
}

func TestCompactGitDiff_Long(t *testing.T) {
	diff := makeDiff(10, 15) // ~150行
	got := compactGitDiff(diff)

	if !strings.HasPrefix(got, "git diff: 10 files changed") {
		t.Errorf("should start with file count summary, got: %s", got[:60])
	}
	if !strings.Contains(got, "internal/pkg0/file0.go") {
		t.Error("should contain file names")
	}
	if !strings.Contains(got, "Use 'git diff <file>' for specific file") {
		t.Error("should contain re-fetch guide")
	}
	// 追加/削除カウントが含まれる
	if !strings.Contains(got, "+") && !strings.Contains(got, "-") {
		t.Error("should contain add/delete counts")
	}
}

func TestCompactGitDiff_NewFile(t *testing.T) {
	diff := "diff --git a/new.go b/new.go\nnew file mode 100644\n" +
		"--- /dev/null\n+++ b/new.go\n@@ -0,0 +1,5 @@\n" +
		strings.Repeat("+line\n", 5)
	// 短いのでそのまま
	got := compactGitDiff(diff)
	if got != diff {
		t.Error("short new file diff should return as-is")
	}
}

func TestCompactGitDiff_NewFileLong(t *testing.T) {
	lines := make([]string, 0, 120)
	lines = append(lines, "diff --git a/new.go b/new.go")
	lines = append(lines, "new file mode 100644")
	lines = append(lines, "--- /dev/null")
	lines = append(lines, "+++ b/new.go")
	lines = append(lines, "@@ -0,0 +1,110 @@")
	for range 110 {
		lines = append(lines, "+new line")
	}
	diff := strings.Join(lines, "\n")
	got := compactGitDiff(diff)
	if !strings.Contains(got, "new file") {
		t.Error("should indicate new file")
	}
}

// ── git log ──

func TestCompactGitLog_StandardShort(t *testing.T) {
	log := makeStandardLog(5)
	got := compactGitLog(log)
	if got != log {
		t.Error("short standard log should return as-is")
	}
}

func TestCompactGitLog_StandardLong(t *testing.T) {
	log := makeStandardLog(20)
	got := compactGitLog(log)

	if !strings.Contains(got, "showing 5 of 20 commits") {
		t.Errorf("should show commit count summary, got:\n%s", got)
	}
	if !strings.Contains(got, "commit message 0") {
		t.Error("should contain first commit message")
	}
	if !strings.Contains(got, "Use 'git show <hash>' for commit details") {
		t.Error("should contain re-fetch guide")
	}
}

func TestCompactGitLog_OnelineShort(t *testing.T) {
	log := makeOnelineLog(8)
	got := compactGitLog(log)
	if got != log {
		t.Error("short oneline log should return as-is")
	}
}

func TestCompactGitLog_OnelineLong(t *testing.T) {
	log := makeOnelineLog(30)
	got := compactGitLog(log)

	if !strings.Contains(got, "showing 5 of 30 commits") {
		t.Errorf("should show commit count summary, got:\n%s", got)
	}
	if !strings.Contains(got, "0000000 fix issue 0") {
		t.Error("should contain first commit")
	}
}

// ── git show ──

func TestCompactGitShow_Short(t *testing.T) {
	show := "commit abc1234\nAuthor: Test <test@test.com>\nDate:   Mon\n\n    msg\n\n" +
		"diff --git a/f.go b/f.go\n@@ -1,2 +1,3 @@\n+line\n"
	got := compactGitShow(show)
	if got != show {
		t.Error("short show should return as-is")
	}
}

func TestCompactGitShow_Long(t *testing.T) {
	header := "commit abc1234567890abcdef1234567890abcdef12345678\n" +
		"Author: Test <test@test.com>\n" +
		"Date:   Mon Mar 10 00:00:00 2026 +0900\n\n" +
		"    fix important bug\n\n"
	diff := makeDiff(5, 20) // ~125行
	show := header + diff

	got := compactGitShow(show)
	if !strings.Contains(got, "fix important bug") {
		t.Error("should preserve commit message")
	}
	if !strings.Contains(got, "5 files changed") {
		t.Error("should contain diff summary")
	}
	if !strings.Contains(got, "Use 'git diff abc1234 -- <file>'") {
		t.Errorf("should contain re-fetch guide, got:\n%s", got)
	}
}

// ── git blame ──

func TestCompactGitBlame_Short(t *testing.T) {
	blame := makeBlame(30)
	got := compactGitBlame(blame)
	if got != blame {
		t.Error("short blame should return as-is")
	}
}

func TestCompactGitBlame_Long(t *testing.T) {
	blame := makeBlame(100)
	got := compactGitBlame(blame)

	if !strings.Contains(got, "git blame summary (100 lines)") {
		t.Errorf("should contain line count, got:\n%s", got)
	}
	// 著者名が含まれる
	if !strings.Contains(got, "Author1") {
		t.Error("should contain author names")
	}
	if !strings.Contains(got, "Use 'git blame -L") {
		t.Error("should contain re-fetch guide")
	}
}

// ── gh run view --log ──

func TestCompactGHRunLog_Short(t *testing.T) {
	log := makeGHRunLog(3, 10, -1)
	got := compactGHRunLog(log)
	if got != log {
		t.Error("short gh run log should return as-is")
	}
}

func TestCompactGHRunLog_Long(t *testing.T) {
	log := makeGHRunLog(5, 30, 3) // 150行, step 3 fails
	got := compactGHRunLog(log)

	if !strings.Contains(got, "gh run:") {
		t.Errorf("should contain step summary, got:\n%s", got)
	}
	if !strings.Contains(got, "✗ Step 3") {
		t.Errorf("should mark failed step, got:\n%s", got)
	}
	if !strings.Contains(got, "##[error]") {
		t.Error("should include failed step tail")
	}
	if !strings.Contains(got, "Use 'gh run view --log --job <id>'") {
		t.Error("should contain re-fetch guide")
	}
}

func TestCompactGHRunLog_NoStructure(t *testing.T) {
	// タブ区切りでないログ
	var lines []string
	for i := range 150 {
		lines = append(lines, fmt.Sprintf("plain log line %d", i))
	}
	log := strings.Join(lines, "\n")
	got := compactGHRunLog(log)

	if !strings.Contains(got, "150 lines total") {
		t.Errorf("should fall back to line count, got:\n%s", got)
	}
}

// ── compactBash routing ──

func TestCompactBash_GitDiff(t *testing.T) {
	diff := makeDiff(10, 15)
	got := compactBash("git diff HEAD~3", diff)
	if !strings.Contains(got, "git diff: 10 files changed") {
		t.Error("compactBash should route git diff to compactGitDiff")
	}
}

func TestCompactBash_GitDiffStat(t *testing.T) {
	// --stat は既にサマリー形式なのでそのまま通す
	output := strings.Repeat(" file.go | 10 +++---\n", 60)
	got := compactBash("git diff --stat", output)
	if got != output {
		t.Error("git diff --stat should return as-is")
	}
}

func TestCompactBash_GitDiffNameOnly(t *testing.T) {
	output := strings.Repeat("internal/agent/file.go\n", 60)
	got := compactBash("git diff --name-only", output)
	if got != output {
		t.Error("git diff --name-only should return as-is")
	}
}

func TestCompactBash_GitLog(t *testing.T) {
	log := makeStandardLog(20)
	got := compactBash("git log", log)
	if !strings.Contains(got, "showing 5 of 20") {
		t.Error("compactBash should route git log to compactGitLog")
	}
}

func TestCompactBash_GitShow(t *testing.T) {
	header := "commit abc1234567890abcdef1234567890abcdef12345678\n" +
		"Author: Test <test@test.com>\n" +
		"Date:   Mon Mar 10 00:00:00 2026 +0900\n\n" +
		"    fix bug\n\n"
	diff := makeDiff(5, 20)
	got := compactBash("git show abc1234", header+diff)
	if !strings.Contains(got, "5 files changed") {
		t.Error("compactBash should route git show to compactGitShow")
	}
}

// ── パーサーユニットテスト ──

func TestExtractDiffPath(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"diff --git a/file.go b/file.go", "file.go"},
		{"diff --git a/internal/agent/compact.go b/internal/agent/compact.go", "internal/agent/compact.go"},
	}
	for _, tt := range tests {
		got := extractDiffPath(tt.line)
		if got != tt.want {
			t.Errorf("extractDiffPath(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestParseHunkHeader(t *testing.T) {
	tests := []struct {
		line      string
		wantStart int
		wantCount int
	}{
		{"@@ -10,5 +10,8 @@ func example()", 10, 8},
		{"@@ -0,0 +1,25 @@", 1, 25},
		{"@@ -1 +1 @@", 1, 1},
		{"@@ -100,50 +200,30 @@ type Foo struct", 200, 30},
	}
	for _, tt := range tests {
		s, c := parseHunkHeader(tt.line)
		if s != tt.wantStart || c != tt.wantCount {
			t.Errorf("parseHunkHeader(%q) = (%d,%d), want (%d,%d)", tt.line, s, c, tt.wantStart, tt.wantCount)
		}
	}
}

func TestExtractBlameAuthor(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"abc1234 (John Doe  2026-03-10 10:00:00 +0900  1) content", "John Doe"},
		{"abc1234 (sugai     2026-03-10 10:00:00 +0900 42) content", "sugai"},
		{"not a blame line", ""},
		{"abc1234 (A 2026-03-10 10:00:00 +0900 1) x", "A"},
	}
	for _, tt := range tests {
		got := extractBlameAuthor(tt.line)
		if got != tt.want {
			t.Errorf("extractBlameAuthor(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestIsGitDiff(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{"git diff", true},
		{"git diff HEAD~3", true},
		{"git diff main...feature", true},
		{"git diff --stat", false},
		{"git diff --name-only", false},
		{"git diff --name-status", false},
		{"  git diff  ", true},
		{"git log", false},
	}
	for _, tt := range tests {
		if got := isGitDiff(tt.cmd); got != tt.want {
			t.Errorf("isGitDiff(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}

func TestIsGHRunLog(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{"gh run view --log", true},
		{"gh run view 12345 --log", true},
		{"gh run view --log --job 123", true},
		{"gh run view", false},
		{"gh run list", false},
	}
	for _, tt := range tests {
		if got := isGHRunLog(tt.cmd); got != tt.want {
			t.Errorf("isGHRunLog(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}
