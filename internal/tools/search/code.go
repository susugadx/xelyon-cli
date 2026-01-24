package search

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// Colors from common package
var green = common.Green

// searchProgress はリアルタイム検索進捗を管理
type searchProgress struct {
	matchCount int
	mu         sync.Mutex
	stopCh     chan struct{}
	done       bool
}

func newSearchProgress() *searchProgress {
	return &searchProgress{
		stopCh: make(chan struct{}),
	}
}

func (sp *searchProgress) increment() {
	sp.mu.Lock()
	sp.matchCount++
	sp.mu.Unlock()
}

func (sp *searchProgress) getCount() int {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	return sp.matchCount
}

func (sp *searchProgress) stop() {
	sp.mu.Lock()
	if !sp.done {
		sp.done = true
		close(sp.stopCh)
	}
	sp.mu.Unlock()
}

// startProgressDisplay は500msごとに進捗を表示
func (sp *searchProgress) startProgressDisplay() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sp.stopCh:
			// 進捗表示をクリア
			fmt.Print("\r\033[K")
			return
		case <-ticker.C:
			count := sp.getCount()
			fmt.Printf("\r🔍 Searching... %d matches found", count)
		}
	}
}

// ExecuteSearchCode searches for a pattern in code files
func ExecuteSearchCode(pattern string, path string) string {
	if pattern == "" {
		return "Error: pattern is required"
	}

	// Cache check
	if tools.GlobalToolCache != nil {
		if cached, ok := tools.GlobalToolCache.GetSearch(pattern, path); ok {
			return cached
		}
	}

	// 設定読み込み
	cfg, _ := config.LoadConfig()
	showProgress := cfg != nil && cfg.Streaming.ShowSearchProgress

	green.Printf("🔍 Searching for '%s' in %s\n", pattern, path)

	// grep search (-r: recursive, -n: line numbers, -I: exclude binary)
	cmd := exec.Command("grep", "-rn", "-I", "--include=*.go", "--include=*.js", "--include=*.ts", "--include=*.py", "--include=*.md", "--include=*.json", "--include=*.yaml", "--include=*.yml", pattern, path)

	var result string

	if showProgress {
		// ストリーミングモードで進捗表示
		result = executeWithProgress(cmd)
	} else {
		// 従来モード
		output, err := cmd.CombinedOutput()
		result = string(output)
		if err != nil && result == "" {
			noMatchResult := fmt.Sprintf("No matches found for '%s'", pattern)
			if tools.GlobalToolCache != nil {
				tools.GlobalToolCache.SetSearch(pattern, path, noMatchResult)
			}
			return noMatchResult
		}
	}

	if strings.TrimSpace(result) == "" {
		noMatchResult := fmt.Sprintf("No matches found for '%s'", pattern)
		if tools.GlobalToolCache != nil {
			tools.GlobalToolCache.SetSearch(pattern, path, noMatchResult)
		}
		return noMatchResult
	}

	// Truncate long results
	lines := strings.Split(result, "\n")
	totalMatches := len(lines)
	if lines[len(lines)-1] == "" {
		totalMatches-- // 末尾の空行を除外
	}
	if len(lines) > 50 {
		result = strings.Join(lines[:50], "\n") + fmt.Sprintf("\n... (%d more matches)", totalMatches-50)
	}

	if tools.GlobalToolCache != nil {
		tools.GlobalToolCache.SetSearch(pattern, path, result)
	}
	return result
}

// executeWithProgress はコマンドを実行しながら進捗を表示
func executeWithProgress(cmd *exec.Cmd) string {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		output, _ := cmd.CombinedOutput()
		return string(output)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		output, _ := cmd.CombinedOutput()
		return string(output)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Sprintf("Error starting command: %v", err)
	}

	progress := newSearchProgress()
	go progress.startProgressDisplay()

	var outputLines []string
	var wg sync.WaitGroup

	// stdout読み取り
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			outputLines = append(outputLines, line)
			progress.increment()
		}
	}()

	// stderr読み取り（エラーメッセージ用）
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			// stderrは無視（grepの警告など）
		}
	}()

	wg.Wait()
	_ = cmd.Wait() // エラーは無視（プロセス終了済み）
	progress.stop()

	return strings.Join(outputLines, "\n")
}
