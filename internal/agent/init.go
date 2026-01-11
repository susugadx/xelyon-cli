package agent

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/repomap"
)

// InitConfig は/initコマンドで収集する情報
type InitConfig struct {
	ProjectName   string
	Overview      string
	TechStack     []string
	CodingRules   string
	RepoMapData   *repomap.RepoMap
	LanguageStats map[string]int // 言語 → ファイル数
}

// handleInitCommand は/initコマンドを処理（XELYON.md生成）
func handleInitCommand(agent *Agent) bool {
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Println("🚀 XELYON.md Generator / プロジェクト設定ファイル生成")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// XELYON.mdが既に存在するか確認
	if _, err := os.Stat("XELYON.md"); err == nil {
		yellow.Println("⚠️  XELYON.md already exists")
		fmt.Print("Overwrite? (y/n): ")
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
	}

	config := &InitConfig{
		TechStack:     []string{},
		LanguageStats: make(map[string]int),
	}

	// Step 1: RepoMapでコード構造を解析
	cyan.Println("📊 Step 1/4: Analyzing codebase...")
	cwd, err := os.Getwd()
	if err != nil {
		red.Printf("Failed to get current directory: %v\n", err)
		return true
	}

	rm := repomap.NewRepoMap(cwd, 0)
	if err := rm.Build(); err != nil {
		yellow.Printf("Warning: Failed to build repo map: %v\n", err)
	} else {
		config.RepoMapData = rm
		// 言語統計を収集
		for _, file := range rm.Files {
			ext := strings.ToLower(filepath.Ext(file.Path))
			lang := extToLanguage(ext)
			if lang != "" {
				config.LanguageStats[lang]++
			}
		}
	}

	// 技術スタックの自動検出
	config.TechStack = detectTechStack(cwd)

	// 検出結果を表示
	if len(config.LanguageStats) > 0 {
		green.Println("   Detected languages:")
		for lang, count := range config.LanguageStats {
			fmt.Printf("     - %s: %d files\n", lang, count)
		}
	}
	if len(config.TechStack) > 0 {
		green.Println("   Detected tech stack:")
		for _, tech := range config.TechStack {
			fmt.Printf("     - %s\n", tech)
		}
	}
	if config.RepoMapData != nil {
		green.Printf("   Symbols found: %d from %d files\n",
			config.RepoMapData.GetSymbolCount(), len(config.RepoMapData.Files))
	}
	fmt.Println()

	// Step 2: プロジェクト名（ディレクトリ名をデフォルト）
	cyan.Println("📝 Step 2/4: Project name")
	defaultName := filepath.Base(cwd)
	fmt.Printf("   Project name [%s]: ", defaultName)
	reader := bufio.NewReader(os.Stdin)
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultName
	}
	config.ProjectName = name
	fmt.Println()

	// Step 3: プロジェクト概要
	cyan.Println("📝 Step 3/4: Project overview")
	yellow.Println("   Enter a brief description of your project (1-3 sentences)")
	yellow.Println("   Press Enter twice to finish")
	fmt.Print("   > ")

	var overviewLines []string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\n\r")
		if line == "" {
			break
		}
		overviewLines = append(overviewLines, line)
		fmt.Print("   > ")
	}
	config.Overview = strings.Join(overviewLines, " ")
	fmt.Println()

	// Step 4: コーディング規約
	cyan.Println("📝 Step 4/4: Coding rules (optional)")
	yellow.Println("   Enter any coding conventions or rules for this project")
	yellow.Println("   Press Enter twice to finish (or just Enter to skip)")
	fmt.Print("   > ")

	var rulesLines []string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\n\r")
		if line == "" {
			break
		}
		rulesLines = append(rulesLines, line)
		fmt.Print("   > ")
	}
	config.CodingRules = strings.Join(rulesLines, "\n")
	fmt.Println()

	// XELYON.md生成
	cyan.Println("📄 Generating XELYON.md...")
	content := generateXELYONMD(config)

	if err := os.WriteFile("XELYON.md", []byte(content), 0644); err != nil {
		red.Printf("Failed to write XELYON.md: %v\n", err)
		return true
	}

	green.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	green.Println("✅ XELYON.md generated successfully!")
	green.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	yellow.Println("Next steps:")
	yellow.Println("  1. Review and edit XELYON.md as needed")
	yellow.Println("  2. Restart xelyon to load the new configuration")
	yellow.Println("  3. XELYON.md will be automatically included in AI context")

	return true
}

// generateXELYONMD はXELYON.mdの内容を生成
func generateXELYONMD(config *InitConfig) string {
	var sb strings.Builder

	// タイトル
	sb.WriteString(fmt.Sprintf("# %s プロジェクト設定\n\n", config.ProjectName))

	// 概要
	sb.WriteString("## 概要\n")
	if config.Overview != "" {
		sb.WriteString(config.Overview + "\n")
	} else {
		sb.WriteString("(プロジェクトの説明をここに記載)\n")
	}
	sb.WriteString("\n")

	// 技術スタック
	sb.WriteString("## 技術スタック\n")
	if len(config.TechStack) > 0 {
		for _, tech := range config.TechStack {
			sb.WriteString(fmt.Sprintf("- **%s**\n", tech))
		}
	} else {
		sb.WriteString("- (使用している技術をここに記載)\n")
	}
	sb.WriteString("\n")

	// ディレクトリ構造
	sb.WriteString("## ディレクトリ構造\n")
	sb.WriteString("```\n")
	sb.WriteString(generateDirectoryTree(config))
	sb.WriteString("```\n\n")

	// コーディング規約
	sb.WriteString("## 開発ルール\n")
	if config.CodingRules != "" {
		sb.WriteString(config.CodingRules + "\n")
	} else {
		sb.WriteString("### コード品質\n")
		sb.WriteString("- (コーディング規約をここに記載)\n")
	}
	sb.WriteString("\n")

	// コードマップ（シンボル一覧）
	if config.RepoMapData != nil && config.RepoMapData.GetSymbolCount() > 0 {
		sb.WriteString("## コードマップ\n")
		sb.WriteString(generateCodeMap(config.RepoMapData))
		sb.WriteString("\n")
	}

	return sb.String()
}

// generateDirectoryTree はディレクトリ構造を生成
func generateDirectoryTree(config *InitConfig) string {
	if config.RepoMapData == nil || len(config.RepoMapData.Files) == 0 {
		return "(ディレクトリ構造をここに記載)\n"
	}

	// ディレクトリを収集
	dirs := make(map[string]bool)
	for _, file := range config.RepoMapData.Files {
		relPath, _ := filepath.Rel(config.RepoMapData.RootPath, file.Path)
		dir := filepath.Dir(relPath)
		if dir != "." {
			// 親ディレクトリも追加
			parts := strings.Split(dir, string(filepath.Separator))
			for i := range parts {
				parentDir := strings.Join(parts[:i+1], string(filepath.Separator))
				dirs[parentDir] = true
			}
		}
	}

	// ソート
	dirList := make([]string, 0, len(dirs))
	for dir := range dirs {
		dirList = append(dirList, dir)
	}
	sort.Strings(dirList)

	var sb strings.Builder
	sb.WriteString(config.ProjectName + "/\n")

	// 主要ディレクトリのみ表示（深さ2まで）
	for _, dir := range dirList {
		depth := strings.Count(dir, string(filepath.Separator))
		if depth > 1 {
			continue
		}
		indent := strings.Repeat("│   ", depth)
		name := filepath.Base(dir)
		sb.WriteString(fmt.Sprintf("%s├── %s/\n", indent, name))
	}

	return sb.String()
}

// generateCodeMap はコードマップを生成（主要なシンボルのみ）
func generateCodeMap(rm *repomap.RepoMap) string {
	var sb strings.Builder

	// ファイルをパスでソート
	files := make([]*repomap.FileSymbols, len(rm.Files))
	copy(files, rm.Files)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	// 最大20ファイルまで表示
	maxFiles := 20
	count := 0

	for _, file := range files {
		if count >= maxFiles {
			sb.WriteString(fmt.Sprintf("\n... and %d more files\n", len(files)-maxFiles))
			break
		}

		relPath, _ := filepath.Rel(rm.RootPath, file.Path)
		sb.WriteString(fmt.Sprintf("### %s\n", relPath))

		// 各ファイルから最大10シンボルまで
		maxSymbols := 10
		for i, sym := range file.Symbols {
			if i >= maxSymbols {
				sb.WriteString(fmt.Sprintf("  ... and %d more symbols\n", len(file.Symbols)-maxSymbols))
				break
			}
			sb.WriteString(fmt.Sprintf("- `%s` (line %d)\n", sym.Signature, sym.Line))
		}
		sb.WriteString("\n")
		count++
	}

	return sb.String()
}

// detectTechStack はプロジェクトの技術スタックを検出
func detectTechStack(path string) []string {
	stack := []string{}

	// Go
	if fileExists(filepath.Join(path, "go.mod")) {
		stack = append(stack, "Go")
	}

	// Node.js / JavaScript / TypeScript
	if fileExists(filepath.Join(path, "package.json")) {
		stack = append(stack, "Node.js")
		if fileExists(filepath.Join(path, "tsconfig.json")) {
			stack = append(stack, "TypeScript")
		}
	}

	// Python
	if fileExists(filepath.Join(path, "requirements.txt")) ||
		fileExists(filepath.Join(path, "pyproject.toml")) ||
		fileExists(filepath.Join(path, "setup.py")) {
		stack = append(stack, "Python")
	}

	// Rust
	if fileExists(filepath.Join(path, "Cargo.toml")) {
		stack = append(stack, "Rust")
	}

	// Docker
	if fileExists(filepath.Join(path, "Dockerfile")) ||
		fileExists(filepath.Join(path, "docker-compose.yml")) ||
		fileExists(filepath.Join(path, "docker-compose.yaml")) {
		stack = append(stack, "Docker")
	}

	// フレームワーク検出
	if fileExists(filepath.Join(path, "next.config.js")) ||
		fileExists(filepath.Join(path, "next.config.mjs")) {
		stack = append(stack, "Next.js")
	}
	if fileExists(filepath.Join(path, "vite.config.js")) ||
		fileExists(filepath.Join(path, "vite.config.ts")) {
		stack = append(stack, "Vite")
	}
	if fileExists(filepath.Join(path, "angular.json")) {
		stack = append(stack, "Angular")
	}

	return stack
}

// extToLanguage は拡張子を言語名に変換
func extToLanguage(ext string) string {
	switch ext {
	case ".go":
		return "Go"
	case ".js", ".mjs", ".cjs":
		return "JavaScript"
	case ".ts", ".tsx":
		return "TypeScript"
	case ".jsx":
		return "React/JSX"
	case ".py":
		return "Python"
	case ".rs":
		return "Rust"
	case ".java":
		return "Java"
	case ".kt", ".kts":
		return "Kotlin"
	case ".rb":
		return "Ruby"
	case ".php":
		return "PHP"
	case ".c", ".h":
		return "C"
	case ".cpp", ".hpp", ".cc", ".cxx":
		return "C++"
	case ".cs":
		return "C#"
	case ".swift":
		return "Swift"
	case ".vue":
		return "Vue"
	case ".svelte":
		return "Svelte"
	default:
		return ""
	}
}

// fileExists はファイルの存在を確認（init.go用）
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
