package agent

import (
	"fmt"
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func parseImageInputWithWriter(out io.Writer, input string) (text string, image *api.ImageData) {
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
		text = DefaultImagePrompt
	}

	// 画像読み込み
	img, err := api.LoadImage(imagePath)
	if err != nil {
		red.Fprintf(out, "Failed to load image: %v\n", err)
		return input, nil
	}

	green.Fprintf(out, "🖼️  Image loaded: %s (%s)\n", img.Path, api.FormatImageSize(img.Size))
	return text, img
}

// printHeaderToWriter はセッション開始時のグラデーションロゴ + Provider/Model 情報を表示する。
func printHeaderToWriter(out io.Writer, model string, provider api.Provider) {
	_, _ = fmt.Fprint(out, buildGradientHeader())
	dim.Fprintf(out, "  Provider: %s | Model: %s\n", provider.Name(), model)
}

func printModeInfoToWriter(out io.Writer, autoApprove, dryRun bool) {
	var modes []string
	if autoApprove {
		modes = append(modes, "Auto-approve")
	}
	if dryRun {
		modes = append(modes, "Dry-run")
	}

	if len(modes) > 0 {
		yellow.Fprintf(out, "  Mode: %s\n", strings.Join(modes, ", "))
	}
}
