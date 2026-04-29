package tui

import "github.com/susugadx/xelyon-cli/internal/config"

// ProjectSavedMsg は /project 画面の保存完了を表す。
type ProjectSavedMsg struct {
	Error    error
	Snapshot *config.ProjectConfig
	ScreenID int
	SaveSeq  int
}

// ProjectTemplateCreatedMsg は /project 画面からの template 作成完了を表す。
type ProjectTemplateCreatedMsg struct {
	Error    error
	Config   *config.ProjectConfig
	ScreenID int
}
