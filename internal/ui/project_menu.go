package ui

import (
	"github.com/susugadx/xelyon-cli/internal/config"
)

// ProjectMenu は xelyon.yaml の対話式編集メニュー
type ProjectMenu struct {
	PC      *config.ProjectConfig
	changed bool
	Runtime *Runtime
}

// NewProjectMenu は新しい ProjectMenu を作成
func NewProjectMenu(pc *config.ProjectConfig) *ProjectMenu {
	return NewProjectMenuWithRuntime(pc, DefaultRuntime())
}

// NewProjectMenuWithRuntime は UI runtime を指定して新しい ProjectMenu を作成する。
func NewProjectMenuWithRuntime(pc *config.ProjectConfig, runtime *Runtime) *ProjectMenu {
	return &ProjectMenu{PC: pc, Runtime: runtimeOrDefault(runtime)}
}
