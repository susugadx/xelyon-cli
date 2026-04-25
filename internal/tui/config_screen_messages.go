package tui

import "github.com/susugadx/xelyon-cli/internal/tui/configscreen"

// OpenConfigScreenMsg は config screen への遷移を要求する Msg。
type OpenConfigScreenMsg = configscreen.OpenMsg

// ConfigSavedMsg は config 保存完了を通知する Msg。
type ConfigSavedMsg = configscreen.SavedMsg
