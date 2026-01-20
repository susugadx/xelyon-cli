//go:build !norepomap
// +build !norepomap

package repomap

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// ================== Bash ==================

func extractBashFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	name := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "word" {
			name = child.Content(content)
			break
		}
	}

	return Symbol{
		Name:      name,
		Kind:      "function",
		Signature: "function " + name + "()",
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== YAML ==================

func isTopLevelYAMLKey(node *sitter.Node) bool {
	// 親がdocument または stream の場合トップレベル
	parent := node.Parent()
	if parent == nil {
		return true
	}
	parentType := parent.Type()
	return parentType == "document" || parentType == "stream" || parentType == "block_mapping"
}

func extractYAMLKey(node *sitter.Node, content []byte, filePath string) Symbol {
	// block_mapping_pair の最初の子がキー
	key := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "flow_node" || child.Type() == "block_node" || strings.Contains(child.Type(), "scalar") {
			key = child.Content(content)
			break
		}
	}

	return Symbol{
		Name:      key,
		Kind:      "key",
		Signature: key + ":",
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== TOML ==================

func extractTOMLTable(node *sitter.Node, content []byte, filePath string) Symbol {
	// [section.name] 形式
	nodeContent := strings.TrimSpace(node.Content(content))
	// 最初の行を取得
	lines := strings.Split(nodeContent, "\n")
	sig := lines[0]
	name := strings.Trim(strings.Trim(sig, "["), "]")

	return Symbol{
		Name:      name,
		Kind:      "table",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractTOMLTableArray(node *sitter.Node, content []byte, filePath string) Symbol {
	// [[array.name]] 形式
	nodeContent := strings.TrimSpace(node.Content(content))
	lines := strings.Split(nodeContent, "\n")
	sig := lines[0]
	name := strings.Trim(strings.Trim(sig, "["), "]")

	return Symbol{
		Name:      name,
		Kind:      "table-array",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== SQL ==================

func extractSQLCreateTable(node *sitter.Node, content []byte, filePath string) Symbol {
	// CREATE TABLE name を抽出
	nodeContent := node.Content(content)
	sig := strings.TrimSpace(strings.Split(nodeContent, "(")[0])
	if len(sig) > 100 {
		sig = sig[:100] + "..."
	}

	// テーブル名を抽出
	name := ""
	parts := strings.Fields(sig)
	for i, p := range parts {
		if strings.ToUpper(p) == "TABLE" && i+1 < len(parts) {
			name = parts[i+1]
			break
		}
	}

	return Symbol{
		Name:      name,
		Kind:      "table",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractSQLCreateFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	nodeContent := node.Content(content)
	// 最初の行を取得
	lines := strings.Split(nodeContent, "\n")
	sig := strings.TrimSpace(lines[0])
	if len(sig) > 100 {
		sig = sig[:100] + "..."
	}

	// 関数名を抽出
	name := ""
	parts := strings.Fields(sig)
	for i, p := range parts {
		if strings.ToUpper(p) == "FUNCTION" && i+1 < len(parts) {
			name = strings.Split(parts[i+1], "(")[0]
			break
		}
	}

	return Symbol{
		Name:      name,
		Kind:      "function",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractSQLCreateView(node *sitter.Node, content []byte, filePath string) Symbol {
	nodeContent := node.Content(content)
	// 最初の行を取得
	lines := strings.Split(nodeContent, "\n")
	sig := strings.TrimSpace(lines[0])
	if len(sig) > 100 {
		sig = sig[:100] + "..."
	}

	// ビュー名を抽出
	name := ""
	parts := strings.Fields(sig)
	for i, p := range parts {
		if strings.ToUpper(p) == "VIEW" && i+1 < len(parts) {
			name = parts[i+1]
			break
		}
	}

	return Symbol{
		Name:      name,
		Kind:      "view",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractSQLCreateIndex(node *sitter.Node, content []byte, filePath string) Symbol {
	nodeContent := node.Content(content)
	sig := strings.TrimSpace(strings.Split(nodeContent, "\n")[0])
	if len(sig) > 100 {
		sig = sig[:100] + "..."
	}

	// インデックス名を抽出
	name := ""
	parts := strings.Fields(sig)
	for i, p := range parts {
		if strings.ToUpper(p) == "INDEX" && i+1 < len(parts) {
			name = parts[i+1]
			break
		}
	}

	return Symbol{
		Name:      name,
		Kind:      "index",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== Dockerfile ==================

func extractDockerFrom(node *sitter.Node, content []byte, filePath string) Symbol {
	nodeContent := strings.TrimSpace(node.Content(content))
	// FROM image:tag AS alias
	name := ""
	parts := strings.Fields(nodeContent)
	if len(parts) >= 2 {
		name = parts[1]
	}

	return Symbol{
		Name:      name,
		Kind:      "from",
		Signature: nodeContent,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractDockerRun(node *sitter.Node, content []byte, filePath string) Symbol {
	nodeContent := strings.TrimSpace(node.Content(content))
	// RUN コマンド（長い場合省略）
	sig := nodeContent
	if len(sig) > 80 {
		sig = sig[:80] + "..."
	}

	return Symbol{
		Name:      "RUN",
		Kind:      "run",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractDockerCmd(node *sitter.Node, content []byte, filePath string) Symbol {
	nodeContent := strings.TrimSpace(node.Content(content))
	// CMD または ENTRYPOINT
	kind := "cmd"
	if strings.HasPrefix(strings.ToUpper(nodeContent), "ENTRYPOINT") {
		kind = "entrypoint"
	}

	return Symbol{
		Name:      strings.ToUpper(kind),
		Kind:      kind,
		Signature: nodeContent,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== Markdown ==================

func extractMarkdownHeading(node *sitter.Node, content []byte, filePath string) Symbol {
	nodeContent := strings.TrimSpace(node.Content(content))
	// # heading → level 1, ## heading → level 2, etc.
	level := 0
	for _, c := range nodeContent {
		if c == '#' {
			level++
		} else {
			break
		}
	}

	heading := strings.TrimLeft(nodeContent, "# ")
	if len(heading) > 80 {
		heading = heading[:80] + "..."
	}

	kind := "h1"
	if level >= 1 && level <= 6 {
		kind = "h" + string(rune('0'+level))
	}

	return Symbol{
		Name:      heading,
		Kind:      kind,
		Signature: nodeContent,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== Makefile - 正規表現ベース ==================

// extractConfigFileSymbols は正規表現ベースでシンボルを抽出（Makefile, Jenkinsfile等）
func extractConfigFileSymbols(filePath string) (*FileSymbols, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	baseName := filepath.Base(filePath)
	ext := strings.ToLower(filepath.Ext(filePath))

	var symbols []Symbol
	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Makefile用正規表現（ターゲット定義: name:）
	makeTargetRe := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_-]*)\s*:`)
	// Jenkinsfile用正規表現（stage('name')）
	jenkinsStageRe := regexp.MustCompile(`stage\s*\(\s*['"]([^'"]+)['"]\s*\)`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		switch {
		case baseName == "Makefile" || ext == ".mk":
			if matches := makeTargetRe.FindStringSubmatch(line); len(matches) > 1 {
				// .PHONY などの特殊ターゲットをスキップ
				target := matches[1]
				if !strings.HasPrefix(target, ".") {
					symbols = append(symbols, Symbol{
						Name:      target,
						Kind:      "target",
						Signature: target + ":",
						FilePath:  filePath,
						Line:      lineNum,
					})
				}
			}
		case baseName == "Jenkinsfile":
			if matches := jenkinsStageRe.FindStringSubmatch(line); len(matches) > 1 {
				symbols = append(symbols, Symbol{
					Name:      matches[1],
					Kind:      "stage",
					Signature: "stage('" + matches[1] + "')",
					FilePath:  filePath,
					Line:      lineNum,
				})
			}
		}
	}

	return &FileSymbols{
		Path:    filePath,
		Symbols: symbols,
	}, nil
}
