package common

import "sync/atomic"

var quietMode int32

// SetQuietMode はツール内部の補助的な標準出力を抑制するか設定する。
func SetQuietMode(enabled bool) {
	if enabled {
		atomic.StoreInt32(&quietMode, 1)
		return
	}
	atomic.StoreInt32(&quietMode, 0)
}

// IsQuietMode はツール内部の補助的な標準出力を抑制中か返す。
func IsQuietMode() bool {
	return atomic.LoadInt32(&quietMode) == 1
}
