package serper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// SearchRequest は Serper API へのリクエスト構造
type SearchRequest struct {
	Q  string `json:"q"`
	Gl string `json:"gl,omitempty"` // 地域コード (optional)
	Hl string `json:"hl,omitempty"` // 言語コード (optional)
}

// SearchResult は検索結果の1件
type SearchResult struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	Snippet string `json:"snippet"`
}

// Response は Serper API のレスポンス構造
type Response struct {
	Organic []SearchResult `json:"organic"`
}

// getURL returns the Serper API URL (allows override for testing)
func getURL() string {
	if url := os.Getenv("SERPER_API_URL"); url != "" {
		return url
	}
	return "https://google.serper.dev/search"
}

// WebSearch は Serper API を使って Web 検索を実行し、上位5件の結果を返す
func WebSearch(query string) (string, error) {
	// 環境変数から API キーを取得
	apiKey := os.Getenv("SERPER_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("SERPER_API_KEY environment variable is not set.\n\nTo use web search, get your free API key at https://serper.dev (2,500 queries/month free)\nThen set it in your environment:\n  export SERPER_API_KEY=your_api_key_here\n\nOr add it to your .env file:\n  echo \"SERPER_API_KEY=your_api_key_here\" >> .env\n\nFor more details, see: https://github.com/susugadx/xelyon-cli/blob/main/docs/config.md#web検索serper-api")
	}

	// リクエストボディを作成
	reqBody := SearchRequest{
		Q:  query,
		Gl: "jp", // 日本の検索結果
		Hl: "ja", // 日本語
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// HTTP リクエストを作成
	req, err := http.NewRequest("POST", getURL(), bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// ヘッダーを設定
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", apiKey)

	// HTTP クライアントで実行
	client := &http.Client{Timeout: config.SerperHTTPTimeout}
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
	var serperResp Response
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
