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

// printHeader はセッション開始時のヘッダーを表示
func printHeader(model string, provider api.Provider) {
	cyan.Println("╔═══════════════════════════════════════════╗")
	cyan.Printf("║  🚀 XELYON CLI v%-25s║\n", version.GetVersion())
	cyan.Println("║  AI-powered coding assistant              ║")
	cyan.Println("╚═══════════════════════════════════════════╝")
	green.Printf("🌐 Provider: %s\n", provider.Name())
	fmt.Printf("   Model: %s\n", modelDisplayName(model))
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

	if len(modes) > 0 {
		yellow.Printf("   Mode: %s\n", strings.Join(modes, ", "))
	} else {
		fmt.Println("   Mode: Normal")
	}
	cyan.Println("───────────────────────────────────────────")
	yellow.Println("Type /help for commands, /exit to quit")
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
