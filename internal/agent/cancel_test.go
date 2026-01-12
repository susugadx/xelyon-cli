package agent

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestContextCanceledDetection は context.Canceled の検出をテスト
func TestContextCanceledDetection(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		isCanceled bool
	}{
		{
			name:       "context.Canceled",
			err:        context.Canceled,
			isCanceled: true,
		},
		{
			name:       "wrapped context.Canceled",
			err:        &wrappedError{err: context.Canceled},
			isCanceled: true,
		},
		{
			name:       "context.DeadlineExceeded",
			err:        context.DeadlineExceeded,
			isCanceled: false,
		},
		{
			name:       "other error",
			err:        errors.New("some other error"),
			isCanceled: false,
		},
		{
			name:       "nil error",
			err:        nil,
			isCanceled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errors.Is(tt.err, context.Canceled)
			if result != tt.isCanceled {
				t.Errorf("errors.Is(%v, context.Canceled) = %v, want %v", tt.err, result, tt.isCanceled)
			}
		})
	}
}

// wrappedError はエラーラッピングのテスト用
type wrappedError struct {
	err error
}

func (e *wrappedError) Error() string {
	return e.err.Error()
}

func (e *wrappedError) Unwrap() error {
	return e.err
}

// TestInterruptTimeoutLogic は3秒以内の2回Ctrl+Cで終了するロジックをテスト
func TestInterruptTimeoutLogic(t *testing.T) {
	tests := []struct {
		name          string
		firstTime     time.Time
		secondTime    time.Time
		shouldExit    bool
	}{
		{
			name:       "2秒以内に2回",
			firstTime:  time.Now(),
			secondTime: time.Now().Add(2 * time.Second),
			shouldExit: true,
		},
		{
			name:       "3秒ちょうどで2回",
			firstTime:  time.Now(),
			secondTime: time.Now().Add(3 * time.Second),
			shouldExit: false, // 3秒以上経過
		},
		{
			name:       "5秒後に2回",
			firstTime:  time.Now(),
			secondTime: time.Now().Add(5 * time.Second),
			shouldExit: false,
		},
		{
			name:       "0.5秒以内に2回",
			firstTime:  time.Now(),
			secondTime: time.Now().Add(500 * time.Millisecond),
			shouldExit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// シミュレーション: secondTime - firstTime < 3秒 なら終了
			shouldExit := tt.secondTime.Sub(tt.firstTime) < 3*time.Second
			if shouldExit != tt.shouldExit {
				t.Errorf("shouldExit = %v, want %v (diff: %v)", shouldExit, tt.shouldExit, tt.secondTime.Sub(tt.firstTime))
			}
		})
	}
}

// TestAgentCancelFuncField は Agent.cancelFunc フィールドの存在をテスト
func TestAgentCancelFuncField(t *testing.T) {
	// Agent構造体にcancelFuncフィールドが存在することを確認
	agent := &Agent{}

	// cancelFunc は初期状態で nil
	if agent.cancelFunc != nil {
		t.Error("Expected cancelFunc to be nil initially")
	}

	// cancelFunc を設定
	ctx, cancel := context.WithCancel(context.Background())
	agent.cancelFunc = cancel

	if agent.cancelFunc == nil {
		t.Error("Expected cancelFunc to be set")
	}

	// キャンセル実行
	agent.cancelFunc()

	// コンテキストがキャンセルされたことを確認
	select {
	case <-ctx.Done():
		// 期待通り
	default:
		t.Error("Expected context to be cancelled")
	}
}
