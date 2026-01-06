package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// SerperSearchRequest は Serper API へのリクエスト構造
type SerperSearchRequest struct {
	Q  string `json:"q"`
	Gl string `json:"gl,omitempty"` // 地域コード (optional)
	Hl string `json:"hl,omitempty"` // 言語コード (optional)
}

// SerperSearchResult は検索結果の1件
type SerperSearchResult struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	Snippet string `json:"snippet"`
}

// SerperResponse は Serper API のレスポンス構造
type SerperResponse struct {
	Organic []SerperSearchResult `json:"organic"`
}

// WebSearch は Serper API を使って Web 検索を実行し、上位5件の結果を返す
func WebSearch(query string) (string, error) {
	// 環境変数から API キーを取得
	apiKey := os.Getenv("SERPER_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("SERPER_API_KEY environment variable is not set")
	}

	// リクエストボディを作成
	reqBody := SerperSearchRequest{
		Q:  query,
		Gl: "jp", // 日本の検索結果
		Hl: "ja", // 日本語
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// HTTP リクエストを作成
	req, err := http.NewRequest("POST", "https://google.serper.dev/search", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// ヘッダーを設定
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", apiKey)

	// HTTP クライアントで実行
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// レスポンスボディを読み込み
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// ステータスコードチェック
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// JSON をパース
	var serperResp SerperResponse
	if err := json.Unmarshal(body, &serperResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 結果を整形（上位5件）
	if len(serperResp.Organic) == 0 {
		return "No results found.", nil
	}

	var result string
	maxResults := 5
	if len(serperResp.Organic) < maxResults {
		maxResults = len(serperResp.Organic)
	}

	result += fmt.Sprintf("Found %d results:\n\n", len(serperResp.Organic))

	for i := 0; i < maxResults; i++ {
		item := serperResp.Organic[i]
		result += fmt.Sprintf("%d. %s\n", i+1, item.Title)
		result += fmt.Sprintf("   URL: %s\n", item.Link)
		result += fmt.Sprintf("   %s\n\n", item.Snippet)
	}

	return result, nil
}
