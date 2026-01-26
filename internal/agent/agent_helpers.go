package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/version"
)

// parseImageInput は入力から画像パスを抽出
// 形式: "image:/path/to/file.png こんにちは" または "こんにちは image:/path/to/file.png"
func parseImageInput(input string) (text string, image *api.ImageData) {
	// image:プレフィックスを探す
	imagePrefix := "image:"

	// 正規表現的な簡易パース
	parts := strings.Fields(input)
	var textParts []string
	var imagePath string

	for _, part := range parts {
		if strings.HasPrefix(part, imagePrefix) {
			imagePath = strings.TrimPrefix(part, imagePrefix)
		} else {
			textParts = append(textParts, part)
		}
	}

	// 画像パスがない場合
	if imagePath == "" {
		return input, nil
	}

	// テキスト部分を結合
	text = strings.Join(textParts, " ")
	if text == "" {
		text = "Please analyze this image." // デフォルトメッセージ
	}

	// 画像読み込み
	img, err := api.LoadImage(imagePath)
	if err != nil {
		red.Printf("Failed to load image: %v\n", err)
		return input, nil
	}

	green.Printf("🖼️  Image loaded: %s (%s)\n", img.Path, api.FormatImageSize(img.Size))
	return text, img
}

// ANSI color codes for gradient (blue -> cyan)
const (
	colorBlue1 = "\033[38;5;27m" // Deep blue
	colorBlue2 = "\033[38;5;33m" // Blue
	colorCyan1 = "\033[38;5;39m" // Light blue
	colorCyan2 = "\033[38;5;45m" // Cyan
	colorCyan3 = "\033[38;5;51m" // Bright cyan
	colorReset = "\033[0m"
	colorDim   = "\033[2m"
)

// printHeader はセッション開始時のヘッダーを表示
func printHeader(model string, provider api.Provider) {
	// ASCII logo with info on the right side
	// Logo lines paired with info text
	type lineInfo struct {
		color string
		logo  string
		info  string
	}

	lines := []lineInfo{
		{colorBlue1, `██╗  ██╗`, ""},
		{colorBlue1, `╚██╗██╔╝`, fmt.Sprintf("%sXELYON%s v%s", colorCyan2, colorReset, version.GetVersion())},
		{colorBlue2, ` ╚███╔╝ `, fmt.Sprintf("%sAI-powered coding assistant%s", colorDim, colorReset)},
		{colorCyan1, ` ██╔██╗ `, ""},
		{colorCyan2, `██╔╝ ██╗`, fmt.Sprintf("Provider: %s", provider.Name())},
		{colorCyan3, `╚═╝  ╚═╝`, fmt.Sprintf("Model: %s", modelDisplayName(model))},
	}

	// Print logo with info
	fmt.Println()
	for _, l := range lines {
		if l.info == "" {
			fmt.Printf("  %s%s%s\n", l.color, l.logo, colorReset)
		} else {
			fmt.Printf("  %s%s%s   %s\n", l.color, l.logo, colorReset, l.info)
		}
	}
	fmt.Println()
}

// printModeInfo はモード情報を表示
func printModeInfo(autoApprove, dryRun bool) {
	var modes []string
	if autoApprove {
		modes = append(modes, "Auto-approve")
	}
	if dryRun {
		modes = append(modes, "Dry-run")
	}

	// 特殊モードのときだけ表示
	if len(modes) > 0 {
		yellow.Printf("  Mode: %s\n\n", strings.Join(modes, ", "))
	}

	cyan.Println("  ─────────────────────────────────────────")
	yellow.Println("  Type /help for commands, /exit to quit")
}

// modelDisplayName はモデル名を表示用にフォーマット
func modelDisplayName(model string) string {
	switch model {
	case "deepseek-chat":
		return "DeepSeek V3 (balanced)"
	case "deepseek-coder":
		return "DeepSeek Coder (code-focused)"
	case "deepseek-reasoner":
		return "DeepSeek R1 (reasoning)"
	case "claude":
		return "Claude (Vertex AI)"
	default:
		return model
	}
}

// loadProjectConfig はXELYON.mdをロード
func loadProjectConfig() string {
	dir, err := os.Getwd()
	if err != nil {
		yellow.Printf("Warning: Could not get current directory: %v\n", err)
		return "" // Cannot locate XELYON.md without working directory
	}
	for {
		path := filepath.Join(dir, "XELYON.md")
		if content, err := os.ReadFile(path); err == nil {
			return string(content)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
