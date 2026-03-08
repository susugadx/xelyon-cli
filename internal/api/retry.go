package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	maxRetries        = 3
	initialBackoff    = 20 * time.Second
	maxBackoff        = 60 * time.Second
	backoffMultiplier = 2
)

// DoWithRetry は HTTP リクエストを実行し、429 Rate Limit の場合にリトライする。
// 最大3回リトライ。Retry-After ヘッダーがあればその秒数を待機、なければ指数バックオフ（20s, 40s, 60s）。
func DoWithRetry(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
	// リクエストボディを保存（リトライ時にリセットするため）
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusTooManyRequests || attempt == maxRetries {
			return resp, nil
		}

		// 429: リトライ待機時間を決定
		waitDuration := RetryAfterDuration(resp, attempt)
		resp.Body.Close()

		showRetryMessage(ctx, waitDuration, attempt+1, maxRetries)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(waitDuration):
		}
	}

	return nil, fmt.Errorf("unexpected: exceeded max retries")
}

// RetryAfterDuration は Retry-After ヘッダーまたは指数バックオフから待機時間を決定
func RetryAfterDuration(resp *http.Response, attempt int) time.Duration {
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		// 秒数として解釈
		if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
		// HTTP-date 形式
		if retryTime, err := http.ParseTime(retryAfter); err == nil {
			if d := time.Until(retryTime); d > 0 {
				return d
			}
		}
	}

	// 指数バックオフ: 20s, 40s, 60s
	backoff := initialBackoff
	for i := 0; i < attempt; i++ {
		backoff *= backoffMultiplier
	}
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	return backoff
}

// showRetryMessage はリトライ待機中のメッセージを表示
func showRetryMessage(ctx context.Context, wait time.Duration, attempt, max int) {
	msg := fmt.Sprintf("⚠️  Rate limited (429), retrying in %ds... (%d/%d)", int(wait.Seconds()), attempt, max)

	// スピナーが動いていればステータスを更新、そうでなければ stderr に出力
	runtime := uiRuntimeFromContext(ctx)
	spinner := runtime.CurrentSpinner()
	if spinner != nil && spinner.IsActive() {
		spinner.SetStatus(msg)
	} else {
		fmt.Fprintln(runtime.ErrorOutput(), msg)
	}
}
