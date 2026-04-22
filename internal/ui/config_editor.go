package ui

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// StringSliceEditor は []string 型の編集UI
type StringSliceEditor struct {
	Path    string
	Current []string
	Runtime *Runtime
}

// NewStringSliceEditor は新しいStringSliceEditorを作成
func NewStringSliceEditor(path string, current []string) *StringSliceEditor {
	return NewStringSliceEditorWithRuntime(path, current, DefaultRuntime())
}

// NewStringSliceEditorWithRuntime は UI runtime を指定して新しい StringSliceEditor を作成する。
func NewStringSliceEditorWithRuntime(path string, current []string, runtime *Runtime) *StringSliceEditor {
	return &StringSliceEditor{
		Path:    path,
		Current: current,
		Runtime: runtimeOrDefault(runtime),
	}
}

// StringMapEditor は map[string]string 型の編集UI
type StringMapEditor struct {
	Path    string
	Current map[string]string
	Runtime *Runtime
}

// NewStringMapEditor は新しいStringMapEditorを作成
func NewStringMapEditor(path string, current map[string]string) *StringMapEditor {
	return NewStringMapEditorWithRuntime(path, current, DefaultRuntime())
}

// NewStringMapEditorWithRuntime は UI runtime を指定して新しい StringMapEditor を作成する。
func NewStringMapEditorWithRuntime(path string, current map[string]string, runtime *Runtime) *StringMapEditor {
	return &StringMapEditor{
		Path:    path,
		Current: current,
		Runtime: runtimeOrDefault(runtime),
	}
}

// StructMapEditor は map[string]struct 型の編集UI
// ProviderModels や LSP Servers などに使用
type StructMapEditor struct {
	Path      string
	FieldType config.ConfigFieldType
	Runtime   *Runtime
}

// NewStructMapEditor は新しいStructMapEditorを作成
func NewStructMapEditor(path string, fieldType config.ConfigFieldType) *StructMapEditor {
	return NewStructMapEditorWithRuntime(path, fieldType, DefaultRuntime())
}

// NewStructMapEditorWithRuntime は UI runtime を指定して新しい StructMapEditor を作成する。
func NewStructMapEditorWithRuntime(path string, fieldType config.ConfigFieldType, runtime *Runtime) *StructMapEditor {
	return &StructMapEditor{
		Path:      path,
		FieldType: fieldType,
		Runtime:   runtimeOrDefault(runtime),
	}
}

// Run は struct map 編集UIを表示
// 注: この実装はシンプル化のため、provider_models と lsp.servers に特化
func (e *StructMapEditor) Run(cfg *config.Config) (bool, error) {
	ctx := newConfigPromptContext(e.Runtime)

	switch e.Path {
	case "provider_models":
		return e.runProviderModels(cfg, ctx.promptIO)
	case "lsp.servers":
		return e.runLSPServers(cfg, ctx.promptIO)
	default:
		_, _ = fmt.Fprintf(ctx.out, "%sStructMap editing not supported for: %s%s\n", colorDim, e.Path, colorReset)
		return false, nil
	}
}

func readLineWithIO(promptIO *PromptIO) string {
	if promptIO == nil {
		promptIO = &PromptIO{}
	}
	line, err := promptIO.ReadSimpleLine()
	if err != nil {
		return ""
	}

	// Bracketed Paste シーケンスを除去 + TrimSpace
	return strings.TrimSpace(StripBracketedPaste(line))
}
