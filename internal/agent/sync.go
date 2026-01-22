package agent

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/repomap"
)

// SyncState は現在のXELYON.mdの状態を表す
type SyncState struct {
	ProjectName  string
	Overview     string
	TechStack    []string
	CodingRules  string
	CodeMapFiles []string // コードマップに含まれるファイル一覧
}

// SyncDiff は変更の差分を表す
type SyncDiff struct {
	AddedTech    []string
	RemovedTech  []string
	AddedFiles   []string
	RemovedFiles []string
	NewSymbols   int
}

// handleSyncCommand は/syncコマンドを処理（XELYON.mdの同期）
func handleSyncCommand(agent *Agent) bool {
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Println("🔄 XELYON.md Sync / プロジェクト設定同期")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// XELYON.mdが存在するか確認
	if !fileExists("XELYON.md") {
		yellow.Println("⚠️  XELYON.md not found")
		yellow.Println("   Run /init first to create XELYON.md")
		return true
	}

	// Step 1: 現在のXELYON.mdをパース
	cyan.Println("📊 Step 1/3: Parsing current XELYON.md...")
	currentState, err := parseXELYONMD("XELYON.md")
	if err != nil {
		red.Printf("Failed to parse XELYON.md: %v\n", err)
		return true
	}
	green.Printf("   Project: %s\n", currentState.ProjectName)
	green.Printf("   Tech stack: %v\n", currentState.TechStack)
	green.Printf("   Code map files: %d\n", len(currentState.CodeMapFiles))
	fmt.Println()

	// Step 2: 現在のコードベースを解析
	cyan.Println("📊 Step 2/3: Analyzing current codebase...")
	cwd, err := os.Getwd()
	if err != nil {
		red.Printf("Failed to get current directory: %v\n", err)
		return true
	}

	rm := repomap.NewRepoMap(cwd, 0)
	if err := rm.Build(); err != nil {
		red.Printf("Failed to build repo map: %v\n", err)
		return true
	}

	newTechStack := detectTechStack(cwd)
	newFiles := make([]string, 0, len(rm.Files))
	for _, f := range rm.Files {
		relPath, _ := filepath.Rel(rm.RootPath, f.Path)
		newFiles = append(newFiles, relPath)
	}
	sort.Strings(newFiles)

	green.Printf("   Detected tech: %v\n", newTechStack)
	green.Printf("   Files found: %d\n", len(newFiles))
	green.Printf("   Symbols found: %d\n", rm.GetSymbolCount())
	fmt.Println()

	// Step 3: 差分を検出
	cyan.Println("📊 Step 3/3: Detecting changes...")
	diff := detectDiff(currentState, newTechStack, newFiles, rm.GetSymbolCount())

	if !diff.HasChanges() {
		green.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		green.Println("✅ XELYON.md is up to date!")
		green.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		return true
	}

	// 変更を表示
	fmt.Println()
	yellow.Println("📝 Changes detected:")
	displayDiff(diff)
	fmt.Println()

	// 確認
	fmt.Print("Apply these changes? (y/n): ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		red.Printf("Failed to read input: %v\n", err)
		return true
	}
	input = strings.TrimSpace(strings.ToLower(input))
	if input != "y" && input != "yes" {
		yellow.Println("Cancelled")
		return true
	}

	// XELYON.mdを更新
	cyan.Println("📄 Updating XELYON.md...")

	// 新しい設定を生成
	config := &InitConfig{
		ProjectName:   currentState.ProjectName,
		Overview:      currentState.Overview,
		TechStack:     newTechStack,
		CodingRules:   currentState.CodingRules,
		RepoMapData:   rm,
		LanguageStats: make(map[string]int),
	}

	// 言語統計を収集
	for _, file := range rm.Files {
		ext := strings.ToLower(filepath.Ext(file.Path))
		lang := extToLanguage(ext)
		if lang != "" {
			config.LanguageStats[lang]++
		}
	}

	content := generateXELYONMD(config)
	if err := os.WriteFile("XELYON.md", []byte(content), 0644); err != nil {
		red.Printf("Failed to write XELYON.md: %v\n", err)
		return true
	}

	green.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	green.Println("✅ XELYON.md synced successfully!")
	green.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return true
}

// parseXELYONMD はXELYON.mdをパースして現在の状態を取得
func parseXELYONMD(path string) (*SyncState, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	state := &SyncState{
		TechStack:    []string{},
		CodeMapFiles: []string{},
	}

	lines := strings.Split(string(content), "\n")
	currentSection := ""

	for _, line := range lines {
		// プロジェクト名（最初の # タイトル）
		if strings.HasPrefix(line, "# ") && state.ProjectName == "" {
			// "# xxx プロジェクト設定" から xxx を抽出
			title := strings.TrimPrefix(line, "# ")
			title = strings.TrimSuffix(title, " プロジェクト設定")
			state.ProjectName = title
			continue
		}

		// セクション検出
		if strings.HasPrefix(line, "## ") {
			currentSection = strings.TrimPrefix(line, "## ")
			continue
		}

		// 概要セクション
		if currentSection == "概要" && !strings.HasPrefix(line, "##") && strings.TrimSpace(line) != "" {
			if state.Overview == "" {
				state.Overview = strings.TrimSpace(line)
			}
		}

		// 技術スタックセクション
		if currentSection == "技術スタック" && strings.HasPrefix(line, "- **") {
			// "- **Go**" から "Go" を抽出
			re := regexp.MustCompile(`\*\*([^*]+)\*\*`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				state.TechStack = append(state.TechStack, matches[1])
			}
		}

		// 開発ルールセクション
		if currentSection == "開発ルール" && !strings.HasPrefix(line, "##") {
			if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "### ") {
				if state.CodingRules == "" {
					state.CodingRules = strings.TrimSpace(line)
				} else {
					state.CodingRules += "\n" + strings.TrimSpace(line)
				}
			}
		}

		// コードマップセクション（### path/to/file.go）
		if currentSection == "コードマップ" && strings.HasPrefix(line, "### ") {
			filePath := strings.TrimPrefix(line, "### ")
			state.CodeMapFiles = append(state.CodeMapFiles, filePath)
		}
	}

	return state, nil
}

// detectDiff は変更の差分を検出
func detectDiff(current *SyncState, newTech []string, newFiles []string, newSymbolCount int) *SyncDiff {
	diff := &SyncDiff{
		AddedTech:    []string{},
		RemovedTech:  []string{},
		AddedFiles:   []string{},
		RemovedFiles: []string{},
		NewSymbols:   newSymbolCount,
	}

	// 技術スタックの差分
	currentTechMap := make(map[string]bool)
	for _, t := range current.TechStack {
		currentTechMap[t] = true
	}
	newTechMap := make(map[string]bool)
	for _, t := range newTech {
		newTechMap[t] = true
	}

	for _, t := range newTech {
		if !currentTechMap[t] {
			diff.AddedTech = append(diff.AddedTech, t)
		}
	}
	for _, t := range current.TechStack {
		if !newTechMap[t] {
			diff.RemovedTech = append(diff.RemovedTech, t)
		}
	}

	// ファイルの差分
	currentFilesMap := make(map[string]bool)
	for _, f := range current.CodeMapFiles {
		currentFilesMap[f] = true
	}
	newFilesMap := make(map[string]bool)
	for _, f := range newFiles {
		newFilesMap[f] = true
	}

	for _, f := range newFiles {
		if !currentFilesMap[f] {
			diff.AddedFiles = append(diff.AddedFiles, f)
		}
	}
	for _, f := range current.CodeMapFiles {
		if !newFilesMap[f] {
			diff.RemovedFiles = append(diff.RemovedFiles, f)
		}
	}

	return diff
}

// HasChanges は変更があるかどうかを返す
func (d *SyncDiff) HasChanges() bool {
	return len(d.AddedTech) > 0 ||
		len(d.RemovedTech) > 0 ||
		len(d.AddedFiles) > 0 ||
		len(d.RemovedFiles) > 0
}

// displayDiff は差分を表示
func displayDiff(diff *SyncDiff) {
	if len(diff.AddedTech) > 0 {
		green.Println("   + Tech stack added:")
		for _, t := range diff.AddedTech {
			fmt.Printf("     + %s\n", t)
		}
	}

	if len(diff.RemovedTech) > 0 {
		red.Println("   - Tech stack removed:")
		for _, t := range diff.RemovedTech {
			fmt.Printf("     - %s\n", t)
		}
	}

	// ファイル変更は最大件数まで表示
	maxDisplay := config.SyncFileChangesMax

	if len(diff.AddedFiles) > 0 {
		green.Printf("   + Files added: %d\n", len(diff.AddedFiles))
		for i, f := range diff.AddedFiles {
			if i >= maxDisplay {
				fmt.Printf("     ... and %d more\n", len(diff.AddedFiles)-maxDisplay)
				break
			}
			fmt.Printf("     + %s\n", f)
		}
	}

	if len(diff.RemovedFiles) > 0 {
		red.Printf("   - Files removed: %d\n", len(diff.RemovedFiles))
		for i, f := range diff.RemovedFiles {
			if i >= maxDisplay {
				fmt.Printf("     ... and %d more\n", len(diff.RemovedFiles)-maxDisplay)
				break
			}
			fmt.Printf("     - %s\n", f)
		}
	}
}
