package dev

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPResponse はHTTPリクエストの結果
type HTTPResponse struct {
	StatusCode int               `json:"status_code"`
	Status     string            `json:"status"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Duration   string            `json:"duration"`
}

// ExecuteHTTPRequest はHTTPリクエストを実行
// method: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS
// url: リクエスト先URL
// headers: リクエストヘッダー（JSON形式）
// body: リクエストボディ
// timeout: タイムアウト秒数（デフォルト: 30）
func ExecuteHTTPRequest(method, url, headers, body string, timeoutSec int) (string, error) {
	if url == "" {
		return "", fmt.Errorf("url is required")
	}

	// メソッドのデフォルトと検証
	method = strings.ToUpper(method)
	if method == "" {
		method = "GET"
	}

	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true,
		"PATCH": true, "HEAD": true, "OPTIONS": true,
	}
	if !validMethods[method] {
		return "", fmt.Errorf("invalid HTTP method: %s", method)
	}

	// URLの検証（基本的なチェック）
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	// タイムアウト設定
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	if timeoutSec > 120 {
		timeoutSec = 120 // 最大2分
	}

	// コンテキスト作成
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// リクエストボディの準備
	var reqBody io.Reader
	if body != "" {
		reqBody = bytes.NewBufferString(body)
	}

	// リクエスト作成
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// デフォルトヘッダー
	req.Header.Set("User-Agent", "xelyon-cli/1.0")

	// カスタムヘッダーをパース
	if headers != "" {
		var headerMap map[string]string
		if err := json.Unmarshal([]byte(headers), &headerMap); err != nil {
			return "", fmt.Errorf("invalid headers JSON: %w", err)
		}
		for k, v := range headerMap {
			req.Header.Set(k, v)
		}
	}

	// Content-Typeが未設定でボディがある場合
	if body != "" && req.Header.Get("Content-Type") == "" {
		// JSONっぽければapplication/json
		if strings.HasPrefix(strings.TrimSpace(body), "{") || strings.HasPrefix(strings.TrimSpace(body), "[") {
			req.Header.Set("Content-Type", "application/json")
		} else {
			req.Header.Set("Content-Type", "text/plain")
		}
	}

	green.Printf("🌐 %s %s\n", method, url)

	// リクエスト実行
	client := &http.Client{
		Timeout: time.Duration(timeoutSec) * time.Second,
	}

	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// レスポンスボディを読み取り
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 最大1MB
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// レスポンスヘッダーを収集
	respHeaders := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			respHeaders[k] = v[0]
		}
	}

	// 結果を整形
	result := HTTPResponse{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    respHeaders,
		Body:       string(respBody),
		Duration:   duration.String(),
	}

	// 読みやすい出力形式
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("HTTP %s\n", result.Status))
	sb.WriteString(fmt.Sprintf("Duration: %s\n", result.Duration))
	sb.WriteString("\n--- Headers ---\n")
	for k, v := range result.Headers {
		sb.WriteString(fmt.Sprintf("%s: %s\n", k, v))
	}
	sb.WriteString("\n--- Body ---\n")

	// JSONならフォーマット
	bodyStr := result.Body
	if strings.Contains(respHeaders["Content-Type"], "application/json") {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, []byte(bodyStr), "", "  "); err == nil {
			bodyStr = prettyJSON.String()
		}
	}

	// ボディが長すぎる場合は切り詰め（ファイル保存方式）
	bodyStr = TruncateWithFile(bodyStr)

	sb.WriteString(bodyStr)

	return sb.String(), nil
}
