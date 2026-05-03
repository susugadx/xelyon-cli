package openai

import (
	"crypto/sha256"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/prompt"
)

var projectConfigSectionRe = regexp.MustCompile(`(?s)<!-- PROJECT_CONFIG_START -->.*?<!-- PROJECT_CONFIG_END -->`)

// BuildPromptCacheKey はプロジェクト・モデル・プロンプトに基づく動的キャッシュキーを生成する。
// フォーマット: "xelyon:v2:{cwd_hash}:{model}:{core_hash}:{project_hash}"
//
// cwd_hash: cwd の SHA-256 先頭8文字
// core_hash: Project Map を除いた主要 prompt 部分の SHA-256 先頭8文字
// project_hash: 条件付き project config 部分の SHA-256 先頭8文字
func BuildPromptCacheKey(model, systemPrompt string) string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "unknown"
	}
	return buildPromptCacheKeyWithCwd(cwd, model, systemPrompt)
}

// buildPromptCacheKeyWithCwd はテスト用に cwd を引数で受け取るバージョン。
func buildPromptCacheKeyWithCwd(cwd, model, systemPrompt string) string {
	cwdHash := shortHash(cwd)
	corePrompt, projectPrompt := normalizePromptSections(systemPrompt)
	return fmt.Sprintf("xelyon:v2:%s:%s:%s:%s", cwdHash, model, shortHash(corePrompt), shortHash(projectPrompt))
}

// shortHash は入力文字列の SHA-256 先頭8文字を返す。
func shortHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:4])
}

func normalizePromptSections(systemPrompt string) (string, string) {
	withoutProjectMap := stripProjectMapForCacheKey(systemPrompt)
	projectSection := projectConfigSectionRe.FindString(withoutProjectMap)
	corePrompt := projectConfigSectionRe.ReplaceAllString(withoutProjectMap, "")
	return collapseWhitespace(corePrompt), collapseWhitespace(projectSection)
}

func stripProjectMapForCacheKey(systemPrompt string) string {
	return prompt.StripProjectMapSectionCompat(systemPrompt)
}

func collapseWhitespace(s string) string {
	fields := strings.Fields(strings.TrimSpace(s))
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
}
