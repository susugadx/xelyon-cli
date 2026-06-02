package evidence

import reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"

var (
	// ErrHostReadOnlyBlocked は host_readonly policy で command 実行が拒否されたことを示す。
	ErrHostReadOnlyBlocked = reviewprobe.ErrHostReadOnlyBlocked
)
