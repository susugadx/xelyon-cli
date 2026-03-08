package ui

import (
	"fmt"
	"sort"
	"strconv"
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

// Run は []string 編集UIを表示し、編集結果を返す
func (e *StringSliceEditor) Run() ([]string, bool, error) {
	runtime := runtimeOrDefault(e.Runtime)
	runtime.StopSpinner()
	runtime.ResetTerminalState()
	promptIO := runtime.PromptIO()
	out := promptIO.Out

	result := make([]string, len(e.Current))
	copy(result, e.Current)

	for {
		// 現在の項目を表示
		_, _ = fmt.Fprintf(out, "\n%s── %s ───────────────────────────────────%s\n\n", colorCyan, e.Path, colorReset)
		_, _ = fmt.Fprintln(out, "  Current items:")

		if len(result) == 0 {
			_, _ = fmt.Fprintf(out, "    %s(empty)%s\n", colorDim, colorReset)
		} else {
			for i, item := range result {
				_, _ = fmt.Fprintf(out, "    %d. %s\n", i+1, truncateString(item, 40))
			}
		}

		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintln(out, "  [a] Add item")
		_, _ = fmt.Fprintln(out, "  [d] Delete item (enter number)")
		_, _ = fmt.Fprintln(out, "  [s] Save and back")
		_, _ = fmt.Fprintln(out, "  [c] Cancel (discard changes)")
		_, _ = fmt.Fprintf(out, "\n%sChoice:%s ", colorCyan, colorReset)

		input := readLineWithIO(&promptIO)
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "a", "add":
			_, _ = fmt.Fprint(out, "Enter new item: ")
			newItem := readLineWithIO(&promptIO)
			newItem = strings.TrimSpace(newItem)
			if newItem != "" {
				result = append(result, newItem)
				_, _ = fmt.Fprintf(out, "%s✓ Added: %s%s\n", colorGreen, newItem, colorReset)
			}

		case "d", "delete":
			if len(result) == 0 {
				_, _ = fmt.Fprintf(out, "%sNo items to delete%s\n", colorDim, colorReset)
				continue
			}
			_, _ = fmt.Fprintf(out, "Enter number to delete (1-%d): ", len(result))
			numStr := readLineWithIO(&promptIO)
			num, err := strconv.Atoi(strings.TrimSpace(numStr))
			if err != nil || num < 1 || num > len(result) {
				_, _ = fmt.Fprintf(out, "%sInvalid number%s\n", colorDim, colorReset)
				continue
			}
			deleted := result[num-1]
			result = append(result[:num-1], result[num:]...)
			_, _ = fmt.Fprintf(out, "%s✓ Deleted: %s%s\n", colorGreen, deleted, colorReset)

		case "s", "save":
			return result, true, nil

		case "c", "cancel":
			return nil, false, nil

		default:
			_, _ = fmt.Fprintf(out, "%sUnknown command. Use a/d/s/c%s\n", colorDim, colorReset)
		}
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

// Run は map[string]string 編集UIを表示し、編集結果を返す
func (e *StringMapEditor) Run() (map[string]string, bool, error) {
	runtime := runtimeOrDefault(e.Runtime)
	runtime.StopSpinner()
	runtime.ResetTerminalState()
	promptIO := runtime.PromptIO()
	out := promptIO.Out

	result := make(map[string]string)
	for k, v := range e.Current {
		result[k] = v
	}

	for {
		// キーをソートして表示
		keys := make([]string, 0, len(result))
		for k := range result {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		_, _ = fmt.Fprintf(out, "\n%s── %s ───────────────────────────────────%s\n\n", colorCyan, e.Path, colorReset)
		_, _ = fmt.Fprintln(out, "  Current entries:")

		if len(result) == 0 {
			_, _ = fmt.Fprintf(out, "    %s(empty)%s\n", colorDim, colorReset)
		} else {
			for i, key := range keys {
				val := result[key]
				_, _ = fmt.Fprintf(out, "    %d. %s → %s\n", i+1, key, truncateString(val, 25))
			}
		}

		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintln(out, "  [a] Add entry")
		_, _ = fmt.Fprintln(out, "  [e] Edit entry (enter number)")
		_, _ = fmt.Fprintln(out, "  [d] Delete entry (enter number)")
		_, _ = fmt.Fprintln(out, "  [s] Save and back")
		_, _ = fmt.Fprintln(out, "  [c] Cancel (discard changes)")
		_, _ = fmt.Fprintf(out, "\n%sChoice:%s ", colorCyan, colorReset)

		input := readLineWithIO(&promptIO)
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "a", "add":
			_, _ = fmt.Fprint(out, "Enter key: ")
			key := strings.TrimSpace(readLineWithIO(&promptIO))
			if key == "" {
				_, _ = fmt.Fprintf(out, "%sKey cannot be empty%s\n", colorDim, colorReset)
				continue
			}
			_, _ = fmt.Fprint(out, "Enter value: ")
			value := strings.TrimSpace(readLineWithIO(&promptIO))
			result[key] = value
			_, _ = fmt.Fprintf(out, "%s✓ Added: %s → %s%s\n", colorGreen, key, value, colorReset)

		case "e", "edit":
			if len(result) == 0 {
				_, _ = fmt.Fprintf(out, "%sNo entries to edit%s\n", colorDim, colorReset)
				continue
			}
			_, _ = fmt.Fprintf(out, "Enter number to edit (1-%d): ", len(keys))
			numStr := readLineWithIO(&promptIO)
			num, err := strconv.Atoi(strings.TrimSpace(numStr))
			if err != nil || num < 1 || num > len(keys) {
				_, _ = fmt.Fprintf(out, "%sInvalid number%s\n", colorDim, colorReset)
				continue
			}
			key := keys[num-1]
			_, _ = fmt.Fprintf(out, "Enter new value for '%s' (current: %s): ", key, result[key])
			newValue := strings.TrimSpace(readLineWithIO(&promptIO))
			if newValue != "" {
				result[key] = newValue
				_, _ = fmt.Fprintf(out, "%s✓ Updated: %s → %s%s\n", colorGreen, key, newValue, colorReset)
			}

		case "d", "delete":
			if len(result) == 0 {
				_, _ = fmt.Fprintf(out, "%sNo entries to delete%s\n", colorDim, colorReset)
				continue
			}
			_, _ = fmt.Fprintf(out, "Enter number to delete (1-%d): ", len(keys))
			numStr := readLineWithIO(&promptIO)
			num, err := strconv.Atoi(strings.TrimSpace(numStr))
			if err != nil || num < 1 || num > len(keys) {
				_, _ = fmt.Fprintf(out, "%sInvalid number%s\n", colorDim, colorReset)
				continue
			}
			key := keys[num-1]
			delete(result, key)
			_, _ = fmt.Fprintf(out, "%s✓ Deleted: %s%s\n", colorGreen, key, colorReset)

		case "s", "save":
			return result, true, nil

		case "c", "cancel":
			return nil, false, nil

		default:
			_, _ = fmt.Fprintf(out, "%sUnknown command. Use a/e/d/s/c%s\n", colorDim, colorReset)
		}
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
	runtime := runtimeOrDefault(e.Runtime)
	runtime.StopSpinner()
	runtime.ResetTerminalState()

	switch e.Path {
	case "provider_models":
		return e.runProviderModels(cfg)
	case "lsp.servers":
		return e.runLSPServers(cfg)
	default:
		_, _ = fmt.Fprintf(runtime.Output(), "%sStructMap editing not supported for: %s%s\n", colorDim, e.Path, colorReset)
		return false, nil
	}
}

func (e *StructMapEditor) runProviderModels(cfg *config.Config) (bool, error) {
	promptIO := runtimeOrDefault(e.Runtime).PromptIO()
	out := promptIO.Out

	for {
		// プロバイダーリスト
		providers := make([]string, 0, len(cfg.ProviderModels))
		for p := range cfg.ProviderModels {
			providers = append(providers, p)
		}
		sort.Strings(providers)

		_, _ = fmt.Fprintf(out, "\n%s── provider_models ──────────────────────%s\n\n", colorCyan, colorReset)
		_, _ = fmt.Fprintln(out, "  Configured providers:")

		if len(providers) == 0 {
			_, _ = fmt.Fprintf(out, "    %s(empty)%s\n", colorDim, colorReset)
		} else {
			for i, p := range providers {
				model := cfg.ProviderModels[p].DefaultModel
				_, _ = fmt.Fprintf(out, "    %d. %s: %s\n", i+1, p, truncateString(model, 25))
			}
		}

		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintln(out, "  [a] Add provider")
		_, _ = fmt.Fprintln(out, "  [1-9] Edit provider")
		_, _ = fmt.Fprintln(out, "  [d] Delete provider")
		_, _ = fmt.Fprintln(out, "  [s] Save and back")
		_, _ = fmt.Fprintln(out, "  [c] Cancel")
		_, _ = fmt.Fprintf(out, "\n%sChoice:%s ", colorCyan, colorReset)

		input := readLineWithIO(&promptIO)
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "a", "add":
			_, _ = fmt.Fprint(out, "Enter provider name: ")
			name := strings.TrimSpace(readLineWithIO(&promptIO))
			if name == "" {
				continue
			}
			_, _ = fmt.Fprint(out, "Enter default model: ")
			model := strings.TrimSpace(readLineWithIO(&promptIO))
			if cfg.ProviderModels == nil {
				cfg.ProviderModels = make(map[string]config.ProviderModelConfig)
			}
			cfg.ProviderModels[name] = config.ProviderModelConfig{DefaultModel: model}
			_, _ = fmt.Fprintf(out, "%s✓ Added: %s%s\n", colorGreen, name, colorReset)

		case "d", "delete":
			if len(providers) == 0 {
				continue
			}
			_, _ = fmt.Fprintf(out, "Enter number to delete (1-%d): ", len(providers))
			numStr := readLineWithIO(&promptIO)
			num, err := strconv.Atoi(strings.TrimSpace(numStr))
			if err != nil || num < 1 || num > len(providers) {
				continue
			}
			name := providers[num-1]
			delete(cfg.ProviderModels, name)
			_, _ = fmt.Fprintf(out, "%s✓ Deleted: %s%s\n", colorGreen, name, colorReset)

		case "s", "save":
			return true, nil

		case "c", "cancel":
			return false, nil

		default:
			// 数字入力の場合
			if num, err := strconv.Atoi(input); err == nil && num >= 1 && num <= len(providers) {
				name := providers[num-1]
				pConfig := cfg.ProviderModels[name]
				_, _ = fmt.Fprintf(out, "\nEditing %s (current: %s)\n", name, pConfig.DefaultModel)
				_, _ = fmt.Fprint(out, "Enter new default model: ")
				model := strings.TrimSpace(readLineWithIO(&promptIO))
				if model != "" {
					pConfig.DefaultModel = model
					cfg.ProviderModels[name] = pConfig
					_, _ = fmt.Fprintf(out, "%s✓ Updated: %s%s\n", colorGreen, name, colorReset)
				}
			}
		}
	}
}

func (e *StructMapEditor) runLSPServers(cfg *config.Config) (bool, error) {
	promptIO := runtimeOrDefault(e.Runtime).PromptIO()
	out := promptIO.Out

	for {
		// サーバーリスト
		servers := make([]string, 0, len(cfg.LSP.Servers))
		for s := range cfg.LSP.Servers {
			servers = append(servers, s)
		}
		sort.Strings(servers)

		_, _ = fmt.Fprintf(out, "\n%s── lsp.servers ──────────────────────────%s\n\n", colorCyan, colorReset)
		_, _ = fmt.Fprintln(out, "  Configured LSP servers:")

		if len(servers) == 0 {
			_, _ = fmt.Fprintf(out, "    %s(using defaults)%s\n", colorDim, colorReset)
		} else {
			for i, s := range servers {
				sConfig := cfg.LSP.Servers[s]
				status := ""
				if sConfig.Disabled {
					status = " (disabled)"
				}
				_, _ = fmt.Fprintf(out, "    %d. %s: %s%s\n", i+1, s, truncateString(sConfig.Command, 20), status)
			}
		}

		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintln(out, "  [a] Add server")
		_, _ = fmt.Fprintln(out, "  [1-9] Edit server")
		_, _ = fmt.Fprintln(out, "  [d] Delete server")
		_, _ = fmt.Fprintln(out, "  [s] Save and back")
		_, _ = fmt.Fprintln(out, "  [c] Cancel")
		_, _ = fmt.Fprintf(out, "\n%sChoice:%s ", colorCyan, colorReset)

		input := readLineWithIO(&promptIO)
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "a", "add":
			_, _ = fmt.Fprint(out, "Enter language (e.g., python, rust): ")
			lang := strings.TrimSpace(readLineWithIO(&promptIO))
			if lang == "" {
				continue
			}
			_, _ = fmt.Fprint(out, "Enter command (e.g., pyright-langserver): ")
			cmd := strings.TrimSpace(readLineWithIO(&promptIO))
			if cfg.LSP.Servers == nil {
				cfg.LSP.Servers = make(map[string]config.LSPServerConfig)
			}
			cfg.LSP.Servers[lang] = config.LSPServerConfig{Command: cmd}
			_, _ = fmt.Fprintf(out, "%s✓ Added: %s%s\n", colorGreen, lang, colorReset)

		case "d", "delete":
			if len(servers) == 0 {
				continue
			}
			_, _ = fmt.Fprintf(out, "Enter number to delete (1-%d): ", len(servers))
			numStr := readLineWithIO(&promptIO)
			num, err := strconv.Atoi(strings.TrimSpace(numStr))
			if err != nil || num < 1 || num > len(servers) {
				continue
			}
			lang := servers[num-1]
			delete(cfg.LSP.Servers, lang)
			_, _ = fmt.Fprintf(out, "%s✓ Deleted: %s%s\n", colorGreen, lang, colorReset)

		case "s", "save":
			return true, nil

		case "c", "cancel":
			return false, nil

		default:
			// 数字入力の場合
			if num, err := strconv.Atoi(input); err == nil && num >= 1 && num <= len(servers) {
				lang := servers[num-1]
				sConfig := cfg.LSP.Servers[lang]

				_, _ = fmt.Fprintf(out, "\nEditing %s:\n", lang)
				_, _ = fmt.Fprintf(out, "  [1] command: %s\n", sConfig.Command)
				_, _ = fmt.Fprintf(out, "  [2] disabled: %v\n", sConfig.Disabled)
				_, _ = fmt.Fprintln(out, "  [b] Back")
				_, _ = fmt.Fprint(out, "\nChoice: ")

				subInput := strings.TrimSpace(readLineWithIO(&promptIO))
				switch subInput {
				case "1":
					_, _ = fmt.Fprint(out, "Enter new command: ")
					cmd := strings.TrimSpace(readLineWithIO(&promptIO))
					if cmd != "" {
						sConfig.Command = cmd
						cfg.LSP.Servers[lang] = sConfig
						_, _ = fmt.Fprintf(out, "%s✓ Updated command%s\n", colorGreen, colorReset)
					}
				case "2":
					sConfig.Disabled = !sConfig.Disabled
					cfg.LSP.Servers[lang] = sConfig
					_, _ = fmt.Fprintf(out, "%s✓ Disabled = %v%s\n", colorGreen, sConfig.Disabled, colorReset)
				}
			}
		}
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
