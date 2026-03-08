package common

import (
	"sync"
	"sync/atomic"
)

var quietModeCount int32

// EnterQuietMode は quiet mode の参照カウントを 1 増やす。
func EnterQuietMode() {
	atomic.AddInt32(&quietModeCount, 1)
}

// ExitQuietMode は quiet mode の参照カウントを 1 減らす。
// カウントが 0 のときは何もしない。
func ExitQuietMode() {
	for {
		current := atomic.LoadInt32(&quietModeCount)
		if current == 0 {
			return
		}
		if atomic.CompareAndSwapInt32(&quietModeCount, current, current-1) {
			return
		}
	}
}

// PushQuietMode は quiet mode を開始し、元に戻す関数を返す。
func PushQuietMode() func() {
	EnterQuietMode()
	var once sync.Once
	return func() {
		once.Do(ExitQuietMode)
	}
}

// SetQuietMode は旧 API 互換ラッパー。
// absolute setter ではなく、enabled=true で Enter、false で Exit を行う。
// 新規コードでは PushQuietMode / EnterQuietMode / ExitQuietMode を使う。
func SetQuietMode(enabled bool) {
	if enabled {
		EnterQuietMode()
		return
	}
	ExitQuietMode()
}

// IsQuietMode は現在 1 つ以上 quiet 実行中か返す。
func IsQuietMode() bool {
	return atomic.LoadInt32(&quietModeCount) > 0
}
