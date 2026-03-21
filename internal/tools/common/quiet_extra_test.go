package common

import (
	"sync/atomic"
	"testing"
)

func TestSetQuietMode_Enable(t *testing.T) {
	// テスト前の状態を保存・復元
	saved := atomic.LoadInt32(&quietModeCount)
	defer atomic.StoreInt32(&quietModeCount, saved)

	atomic.StoreInt32(&quietModeCount, 0)

	SetQuietMode(true)
	if !IsQuietMode() {
		t.Error("expected IsQuietMode() == true after SetQuietMode(true)")
	}
}

func TestSetQuietMode_Disable(t *testing.T) {
	saved := atomic.LoadInt32(&quietModeCount)
	defer atomic.StoreInt32(&quietModeCount, saved)

	// まず有効にしてから無効にする
	atomic.StoreInt32(&quietModeCount, 1)

	SetQuietMode(false)
	if IsQuietMode() {
		t.Error("expected IsQuietMode() == false after SetQuietMode(false)")
	}
}

func TestSetQuietMode_EnableDisable(t *testing.T) {
	saved := atomic.LoadInt32(&quietModeCount)
	defer atomic.StoreInt32(&quietModeCount, saved)

	atomic.StoreInt32(&quietModeCount, 0)

	// 有効化
	SetQuietMode(true)
	if !IsQuietMode() {
		t.Error("expected quiet mode enabled after SetQuietMode(true)")
	}

	// 無効化
	SetQuietMode(false)
	if IsQuietMode() {
		t.Error("expected quiet mode disabled after SetQuietMode(false)")
	}

	// 再度有効化
	SetQuietMode(true)
	if !IsQuietMode() {
		t.Error("expected quiet mode re-enabled after second SetQuietMode(true)")
	}

	// 再度無効化
	SetQuietMode(false)
	if IsQuietMode() {
		t.Error("expected quiet mode disabled after second SetQuietMode(false)")
	}
}
