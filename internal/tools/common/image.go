package common

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// MaxImageSize: 10MB
	MaxImageSize = 10 * 1024 * 1024
)

// ImageData は画像データを表す
type ImageData struct {
	Path      string // 元のファイルパス
	MediaType string // "image/png", "image/jpeg" 等
	Base64    string // Base64エンコードされた画像データ
	Size      int64  // ファイルサイズ（バイト）
}

// LoadImage は画像ファイルを読み込んでBase64エンコードする
func LoadImage(path string) (*ImageData, error) {
	// ファイルの存在確認
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("image file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to stat image file: %w", err)
	}

	// ディレクトリでないことを確認
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not an image file: %s", path)
	}

	// ファイルサイズチェック
	if info.Size() > MaxImageSize {
		return nil, fmt.Errorf("image file too large: %d bytes (max: %d bytes)", info.Size(), MaxImageSize)
	}

	// MIMEタイプを判定（拡張子ベース）
	mediaType, err := getMediaType(path)
	if err != nil {
		return nil, err
	}

	// ファイル読み込み
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %w", err)
	}

	// Base64エンコード
	encoded := base64.StdEncoding.EncodeToString(data)

	return &ImageData{
		Path:      path,
		MediaType: mediaType,
		Base64:    encoded,
		Size:      info.Size(),
	}, nil
}

// getMediaType は拡張子からMIMEタイプを判定
func getMediaType(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".png":
		return "image/png", nil
	case ".jpg", ".jpeg":
		return "image/jpeg", nil
	case ".gif":
		return "image/gif", nil
	case ".webp":
		return "image/webp", nil
	default:
		return "", fmt.Errorf("unsupported image format: %s (supported: png, jpg, jpeg, gif, webp)", ext)
	}
}

// FormatSize はバイト数を人間が読みやすい形式に変換
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
