package file

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

type countingReadCache struct {
	mu           sync.Mutex
	files        map[string]string
	getFileCalls int
	setFileCalls int
}

func (c *countingReadCache) GetFile(path string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.getFileCalls++
	if c.files == nil {
		return "", false
	}
	content, ok := c.files[path]
	return content, ok
}

func (c *countingReadCache) SetFile(path, content string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setFileCalls++
	if c.files == nil {
		c.files = make(map[string]string)
	}
	c.files[path] = content
}

func (c *countingReadCache) GetDir(string) (string, bool) { return "", false }
func (c *countingReadCache) SetDir(string, string)        {}
func (c *countingReadCache) InvalidateFile(string)        {}
func (c *countingReadCache) InvalidateDir(string)         {}
func (c *countingReadCache) Clear()                       {}
func (c *countingReadCache) GetSearch(string, string) (string, bool) {
	return "", false
}
func (c *countingReadCache) SetSearch(string, string, string, []string) {}
func (c *countingReadCache) ClearSearchCache()                          {}
func (c *countingReadCache) InvalidateSearchCacheForFile(string)        {}

func TestResolveReadDetail_FullBudgetKeepsAutoMode(t *testing.T) {
	t.Parallel()

	mode, errResult := resolveReadDetail("", "true")
	if errResult != "" {
		t.Fatalf("unexpected error: %s", errResult)
	}
	if mode != readDetailAuto {
		t.Fatalf("resolveReadDetail(_, true) = %q, want %q", mode, readDetailAuto)
	}
}

func TestResolveReadBudgetOverride(t *testing.T) {
	t.Parallel()

	if got := resolveReadBudgetOverride("", "true"); got != DefaultFullLines {
		t.Fatalf("resolveReadBudgetOverride(\"\", true) = %d, want %d", got, DefaultFullLines)
	}
	if got := resolveReadBudgetOverride("outline", "true"); got != 0 {
		t.Fatalf("resolveReadBudgetOverride(detail, true) = %d, want 0", got)
	}
}

func TestExecuteReadBatchRequest_DetailFullForcesWholeFileRead(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	lines := make([]string, 2200)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%d", i+1)
	}
	filePath := filepath.Join(tmpDir, "large.txt")
	testutil.CreateTempFile(t, tmpDir, "large.txt", strings.Join(lines, "\n"))

	result := executeReadBatchRequest(common.DefaultOutput(), nil, nil, readRequest{
		RawEntry: filePath,
		FilePath: filePath,
		Source:   readRequestSourcePathWhole,
		Detail:   readDetailFull,
	}, DefaultFullLines)

	if strings.Contains(result.result, "lines total") {
		t.Fatalf("detail=full should avoid outline, got: %s", result.result)
	}
	if !strings.Contains(result.result, "2200: line2200") {
		t.Fatalf("detail=full should include the last line, got: %s", result.result)
	}
}

func TestExecuteReadBatchRequest_DetailOutlineForcesWholeFileOutline(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	lines := make([]string, 120)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%d", i+1)
	}
	filePath := filepath.Join(tmpDir, "small.txt")
	testutil.CreateTempFile(t, tmpDir, "small.txt", strings.Join(lines, "\n"))

	result := executeReadBatchRequest(common.DefaultOutput(), nil, nil, readRequest{
		RawEntry: filePath,
		FilePath: filePath,
		Source:   readRequestSourcePathWhole,
		Detail:   readDetailOutline,
	}, DefaultFullLines)

	if !strings.Contains(result.result, "lines total") {
		t.Fatalf("detail=outline should force outline output, got: %s", result.result)
	}
	if strings.Contains(result.result, "31: line31") {
		t.Fatalf("outline output should not include middle lines, got: %s", result.result)
	}
	if !strings.Contains(result.result, "--- Last lines ---") || !strings.Contains(result.result, "120: line120") {
		t.Fatalf("detail=outline should preserve tail sample, got: %s", result.result)
	}
}

func TestExecuteReadBatchRequest_DetailCompactWholeFileErrors(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "compact.txt")
	testutil.CreateTempFile(t, tmpDir, "compact.txt", generateLines(80))

	result := executeReadBatchRequest(common.DefaultOutput(), nil, nil, readRequest{
		RawEntry: filePath,
		FilePath: filePath,
		Source:   readRequestSourcePathWhole,
		Detail:   readDetailCompact,
	}, DefaultFullLines)

	if !strings.Contains(result.result, `Error: detail="compact" requires locator targets or explicit path ranges`) {
		t.Fatalf("expected explicit compact whole-file error, got: %s", result.result)
	}
}

func TestExecuteReadBatchRequest_DetailOutlineLargeFileAvoidsWholeFileLoad(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	lines := make([]string, 1300)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%d %s", i+1, strings.Repeat("x", 900))
	}
	filePath := filepath.Join(tmpDir, "large-outline.txt")
	testutil.CreateTempFile(t, tmpDir, "large-outline.txt", strings.Join(lines, "\n"))

	cache := &countingReadCache{}
	result := executeReadBatchRequest(common.DefaultOutput(), nil, cache, readRequest{
		RawEntry: filePath,
		FilePath: filePath,
		Source:   readRequestSourcePathWhole,
		Detail:   readDetailOutline,
	}, DefaultFullLines)

	if !strings.Contains(result.result, "lines total") {
		t.Fatalf("detail=outline should still produce outline output, got: %s", result.result)
	}
	if cache.setFileCalls != 0 {
		t.Fatalf("large-file outline should avoid whole-file load caching, got %d SetFile calls", cache.setFileCalls)
	}
}

func TestExecuteReadBatchRequest_DetailOutlineLargeFileKeepsActualTail(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	lines := make([]string, 2200)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%d %s", i+1, strings.Repeat("x", 600))
	}
	filePath := filepath.Join(tmpDir, "large-outline-tail.txt")
	testutil.CreateTempFile(t, tmpDir, "large-outline-tail.txt", strings.Join(lines, "\n"))

	result := executeReadBatchRequest(common.DefaultOutput(), nil, nil, readRequest{
		RawEntry: filePath,
		FilePath: filePath,
		Source:   readRequestSourcePathWhole,
		Detail:   readDetailOutline,
	}, DefaultFullLines)

	if !strings.Contains(result.result, "(2200 lines total.") {
		t.Fatalf("detail=outline should report exact total lines for large files, got: %s", result.result)
	}
	if !strings.Contains(result.result, "2191: line2191") || !strings.Contains(result.result, "2200: line2200") {
		t.Fatalf("detail=outline should preserve actual large-file tail, got: %s", result.result)
	}
	if strings.Contains(result.result, "992: line992") || strings.Contains(result.result, "1001: line1001") {
		t.Fatalf("detail=outline should not report sampled pseudo-tail for large files, got: %s", result.result)
	}
}

func TestExecuteReadBatchRequest_DetailOutlineMediumFileKeepsActualTail(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	lines := make([]string, 1200)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%d", i+1)
	}
	filePath := filepath.Join(tmpDir, "medium-outline.txt")
	testutil.CreateTempFile(t, tmpDir, "medium-outline.txt", strings.Join(lines, "\n"))

	result := executeReadBatchRequest(common.DefaultOutput(), nil, nil, readRequest{
		RawEntry: filePath,
		FilePath: filePath,
		Source:   readRequestSourcePathWhole,
		Detail:   readDetailOutline,
	}, DefaultFullLines)

	if !strings.Contains(result.result, "(1200 lines total.") {
		t.Fatalf("detail=outline should report the exact total lines for medium files, got: %s", result.result)
	}
	if !strings.Contains(result.result, "--- Last lines ---") || !strings.Contains(result.result, "1191: line1191") || !strings.Contains(result.result, "1200: line1200") {
		t.Fatalf("detail=outline should preserve the actual file tail, got: %s", result.result)
	}
	if strings.Contains(result.result, "992: line992") || strings.Contains(result.result, "1001: line1001") {
		t.Fatalf("detail=outline should not use sampled pseudo-tail for medium files, got: %s", result.result)
	}
}

func TestExecuteReadBatchRequest_DetailOutlineLargeSingleLineDoesNotFailOrExpandToWholeFile(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "single-line-minified.js")
	testutil.CreateTempFile(t, tmpDir, "single-line-minified.js", strings.Repeat("x", LargeFileThreshold+1024))

	result := executeReadBatchRequest(common.DefaultOutput(), nil, nil, readRequest{
		RawEntry: filePath,
		FilePath: filePath,
		Source:   readRequestSourcePathWhole,
		Detail:   readDetailOutline,
	}, DefaultFullLines)

	if strings.Contains(result.result, "token too long") || strings.Contains(result.result, "Error reading file:") {
		t.Fatalf("detail=outline should not fail on large single-line files, got: %s", result.result)
	}
	if !strings.Contains(result.result, "lines total") {
		t.Fatalf("detail=outline should still produce outline output, got: %s", result.result)
	}
	if len(result.result) > 10000 {
		t.Fatalf("detail=outline should avoid expanding a huge single line into full output, got %d bytes", len(result.result))
	}
}

func TestExecuteReadBatchRequest_DetailOutlineSmallSingleLineStaysBounded(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "small-single-line-minified.js")
	testutil.CreateTempFile(t, tmpDir, "small-single-line-minified.js", strings.Repeat("x", 900*1024))

	result := executeReadBatchRequest(common.DefaultOutput(), nil, nil, readRequest{
		RawEntry: filePath,
		FilePath: filePath,
		Source:   readRequestSourcePathWhole,
		Detail:   readDetailOutline,
	}, DefaultFullLines)

	if strings.Contains(result.result, "Error reading file:") {
		t.Fatalf("detail=outline should not fail on sub-threshold single-line files, got: %s", result.result)
	}
	if !strings.Contains(result.result, "lines total") || !strings.Contains(result.result, "...") {
		t.Fatalf("detail=outline should stay bounded on sub-threshold single-line files, got: %s", result.result)
	}
	if len(result.result) > 10000 {
		t.Fatalf("detail=outline should stay bounded, got %d bytes", len(result.result))
	}
}

func TestExecuteReadBatchRequest_DetailDoesNotChangeExplicitRangeRead(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	lines := make([]string, 80)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%d", i+1)
	}
	filePath := filepath.Join(tmpDir, "range.txt")
	testutil.CreateTempFile(t, tmpDir, "range.txt", strings.Join(lines, "\n"))

	for _, detail := range []readDetailMode{readDetailFull, readDetailOutline} {
		t.Run(string(detail), func(t *testing.T) {
			result := executeReadBatchRequest(common.DefaultOutput(), nil, nil, readRequest{
				RawEntry:   filePath + ":10-20",
				FilePath:   filePath,
				StartLine:  10,
				EndLine:    20,
				Source:     readRequestSourcePathRange,
				Detail:     detail,
				RangeEntry: filePath + ":10-20",
			}, DefaultFullLines)

			if !strings.Contains(result.result, "10: line10") || !strings.Contains(result.result, "20: line20") {
				t.Fatalf("range output missing requested lines, got: %s", result.result)
			}
			if strings.Contains(result.result, "9: line9") || strings.Contains(result.result, "21: line21") {
				t.Fatalf("range output should stay bounded, got: %s", result.result)
			}
			if strings.Contains(result.result, "lines total") {
				t.Fatalf("range output should not switch to outline, got: %s", result.result)
			}
		})
	}
}
