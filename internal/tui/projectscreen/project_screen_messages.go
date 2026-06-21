package projectscreen

import "github.com/susugadx/xelyon-cli/internal/config"

// TemplateResult は /project template 作成コマンドの結果を表す。
type TemplateResult struct {
	Error    error
	Config   *config.ProjectConfig
	ScreenID int
}

// InstallTemplateResult は template 作成結果を screen state に反映する。
func (ps *Screen) InstallTemplateResult(msg TemplateResult) bool {
	if msg.ScreenID != ps.screenID {
		return false
	}
	if msg.Error != nil {
		ps.saveStatus = projectStatusFailed
		ps.saveError = msg.Error.Error()
		ps.message = ""
		return true
	}
	replacement := New(msg.Config, ps.screenID)
	*ps = *replacement
	ps.message = "template created"
	return true
}
