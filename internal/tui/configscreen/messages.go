package configscreen

import "github.com/susugadx/xelyon-cli/internal/config"

// OpenMsg は config screen への遷移を要求する Msg。
type OpenMsg struct{}

// SavedMsg は config 保存完了を通知する Msg。
type SavedMsg struct {
	Error    error
	Snapshot *config.Config
}
