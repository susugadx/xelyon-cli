package search

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ExecuteSearchFile searches for files by name pattern
func ExecuteSearchFile(pattern string, path string) string {
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

	green.Printf("📁 Searching for files matching '%s' in %s\n", pattern, path)

	// find search (exclude .git)
	cmd := exec.Command("find", path, "-type", "f", "-name", pattern, "-not", "-path", "*/.git/*")

	var result string

	if showProgress {
		// ストリーミングモードで進捗表示
		result = executeFileSearchWithProgress(cmd)
	} else {
		// 従来モード
		output, err := cmd.CombinedOutput()
		result = string(output)
		if err != nil {
			return fmt.Sprintf("Error: %v\n%s", err, result)
		}
	}

	if strings.TrimSpace(result) == "" {
		noMatchResult := fmt.Sprintf("No files found matching '%s'", pattern)
		if tools.GlobalToolCache != nil {
			tools.GlobalToolCache.SetSearch(pattern, path, noMatchResult)
		}
		return noMatchResult
	}

	// Truncate long results
	lines := strings.Split(result, "\n")
	totalFiles := len(lines)
	if lines[len(lines)-1] == "" {
		totalFiles-- // 末尾の空行を除外
	}
	if len(lines) > 30 {
		result = strings.Join(lines[:30], "\n") + fmt.Sprintf("\n... (%d more files)", totalFiles-30)
	}

	if tools.GlobalToolCache != nil {
		tools.GlobalToolCache.SetSearch(pattern, path, result)
	}
	return result
}

// executeFileSearchWithProgress はファイル検索を実行しながら進捗を表示
func executeFileSearchWithProgress(cmd *exec.Cmd) string {
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
			// stderrは無視（findの警告など）
		}
	}()

	wg.Wait()
	cmd.Wait()
	progress.stop()

	return strings.Join(outputLines, "\n")
}
