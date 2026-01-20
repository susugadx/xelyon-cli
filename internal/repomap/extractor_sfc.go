//go:build !norepomap
// +build !norepomap

package repomap

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// SFCSection は SFC の各セクション情報
type SFCSection struct {
	Tag       string // "script", "style", "template"
	Content   string
	StartLine int
	Lang      string // "ts", "scss" など
}

// extractSFCSymbols は Vue/Svelte SFC からシンボルを抽出
func extractSFCSymbols(filePath string) (*FileSymbols, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	sections := parseSFCSections(string(content))

	var symbols []Symbol

	for _, section := range sections {
		switch section.Tag {
		case "script":
			// <script> セクションを JS/TS パーサーで解析
			scriptSymbols := extractSFCScriptSymbols(section, filePath, ext)
			symbols = append(symbols, scriptSymbols...)

		case "style":
			// <style> セクションを CSS パーサーで解析
			styleSymbols := extractSFCStyleSymbols(section, filePath)
			symbols = append(symbols, styleSymbols...)
		}
	}

	return &FileSymbols{
		Path:    filePath,
		Symbols: symbols,
	}, nil
}

// parseSFCSections は SFC の内容をセクションごとに分割
func parseSFCSections(content string) []SFCSection {
	var sections []SFCSection

	// <script>, <style>, <template> タグを正規表現で抽出
	// <script lang="ts" setup> のような属性も対応
	scriptRe := regexp.MustCompile(`(?is)<script([^>]*)>(.*?)</script>`)
	styleRe := regexp.MustCompile(`(?is)<style([^>]*)>(.*?)</style>`)

	// Script セクション
	for _, match := range scriptRe.FindAllStringSubmatchIndex(content, -1) {
		attrStart, attrEnd := match[2], match[3]
		contentStart, contentEnd := match[4], match[5]

		attrs := ""
		if attrStart >= 0 && attrEnd >= 0 {
			attrs = content[attrStart:attrEnd]
		}

		sectionContent := ""
		if contentStart >= 0 && contentEnd >= 0 {
			sectionContent = content[contentStart:contentEnd]
		}

		// 開始行を計算
		startLine := strings.Count(content[:contentStart], "\n") + 1

		// lang属性を抽出
		lang := extractSFCLangAttr(attrs, "js")

		sections = append(sections, SFCSection{
			Tag:       "script",
			Content:   sectionContent,
			StartLine: startLine,
			Lang:      lang,
		})
	}

	// Style セクション
	for _, match := range styleRe.FindAllStringSubmatchIndex(content, -1) {
		attrStart, attrEnd := match[2], match[3]
		contentStart, contentEnd := match[4], match[5]

		attrs := ""
		if attrStart >= 0 && attrEnd >= 0 {
			attrs = content[attrStart:attrEnd]
		}

		sectionContent := ""
		if contentStart >= 0 && contentEnd >= 0 {
			sectionContent = content[contentStart:contentEnd]
		}

		startLine := strings.Count(content[:contentStart], "\n") + 1
		lang := extractSFCLangAttr(attrs, "css")

		sections = append(sections, SFCSection{
			Tag:       "style",
			Content:   sectionContent,
			StartLine: startLine,
			Lang:      lang,
		})
	}

	return sections
}

// extractSFCLangAttr は lang="xxx" 属性を抽出
func extractSFCLangAttr(attrs, defaultLang string) string {
	langRe := regexp.MustCompile(`lang\s*=\s*["']([^"']+)["']`)
	if match := langRe.FindStringSubmatch(attrs); len(match) > 1 {
		return match[1]
	}
	return defaultLang
}

// extractSFCScriptSymbols は <script> セクションからシンボルを抽出
func extractSFCScriptSymbols(section SFCSection, filePath, sfcExt string) []Symbol {
	var symbols []Symbol

	// 言語に応じたパーサーを選択
	var lang *sitter.Language
	switch section.Lang {
	case "ts", "typescript":
		lang = SupportedLanguages[".ts"]
	default:
		lang = SupportedLanguages[".js"]
	}

	if lang == nil {
		return symbols
	}

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(lang)

	content := []byte(section.Content)
	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return symbols
	}
	defer tree.Close()

	// スクリプト用の仮想ファイルパス（.ts/.js として解析）
	virtualExt := ".js"
	if section.Lang == "ts" || section.Lang == "typescript" {
		virtualExt = ".ts"
	}
	virtualPath := filePath + virtualExt

	root := tree.RootNode()
	rawSymbols := extractFromNode(root, content, virtualPath)

	// 行番号をオフセットで調整し、実際のファイルパスに戻す
	for _, sym := range rawSymbols {
		sym.Line += section.StartLine - 1
		sym.FilePath = filePath

		// Svelte 固有: $: ラベルはリアクティブステートメント
		if sfcExt == ".svelte" && strings.HasPrefix(sym.Name, "$:") {
			sym.Kind = "reactive"
		}

		symbols = append(symbols, sym)
	}

	return symbols
}

// extractSFCStyleSymbols は <style> セクションからシンボルを抽出
func extractSFCStyleSymbols(section SFCSection, filePath string) []Symbol {
	var symbols []Symbol

	lang := SupportedLanguages[".css"]
	if lang == nil {
		return symbols
	}

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(lang)

	content := []byte(section.Content)
	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return symbols
	}
	defer tree.Close()

	virtualPath := filePath + ".css"
	root := tree.RootNode()
	rawSymbols := extractFromNode(root, content, virtualPath)

	// 行番号をオフセットで調整
	for _, sym := range rawSymbols {
		sym.Line += section.StartLine - 1
		sym.FilePath = filePath
		symbols = append(symbols, sym)
	}

	return symbols
}
