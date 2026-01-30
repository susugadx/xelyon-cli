package ui

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// StringSliceEditor は []string 型の編集UI
type StringSliceEditor struct {
	Path    string
	Current []string
}

// NewStringSliceEditor は新しいStringSliceEditorを作成
func NewStringSliceEditor(path string, current []string) *StringSliceEditor {
	return &StringSliceEditor{
		Path:    path,
		Current: current,
	}
}

// Run は []string 編集UIを表示し、編集結果を返す
func (e *StringSliceEditor) Run() ([]string, bool, error) {
	StopGlobalSpinner()
	fmt.Print("\033[?25h") // カーソル表示

	result := make([]string, len(e.Current))
	copy(result, e.Current)

	for {
		// 現在の項目を表示
		fmt.Printf("\n%s┌─ %s ─────────────────────────────────┐%s\n", colorCyan, e.Path, colorReset)
		fmt.Printf("│ Current items:                       │\n")

		if len(result) == 0 {
			fmt.Printf("│   %s(empty)%s                           │\n", colorDim, colorReset)
		} else {
			for i, item := range result {
				fmt.Printf("│   %d. %s\n", i+1, truncateString(item, 35))
			}
		}

		fmt.Println("│                                      │")
		fmt.Println("│ [a] Add item                         │")
		fmt.Println("│ [d] Delete item (enter number)       │")
		fmt.Println("│ [s] Save and back                    │")
		fmt.Println("│ [c] Cancel (discard changes)         │")
		fmt.Printf("%s└──────────────────────────────────────┘%s\n", colorCyan, colorReset)
		fmt.Printf("\n%sChoice:%s ", colorCyan, colorReset)

		input := readLine()
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "a", "add":
			fmt.Printf("Enter new item: ")
			newItem := readLine()
			newItem = strings.TrimSpace(newItem)
			if newItem != "" {
				result = append(result, newItem)
				fmt.Printf("%s✓ Added: %s%s\n", colorGreen, newItem, colorReset)
			}

		case "d", "delete":
			if len(result) == 0 {
				fmt.Printf("%sNo items to delete%s\n", colorDim, colorReset)
				continue
			}
			fmt.Printf("Enter number to delete (1-%d): ", len(result))
			numStr := readLine()
			num, err := strconv.Atoi(strings.TrimSpace(numStr))
			if err != nil || num < 1 || num > len(result) {
				fmt.Printf("%sInvalid number%s\n", colorDim, colorReset)
				continue
			}
			deleted := result[num-1]
			result = append(result[:num-1], result[num:]...)
			fmt.Printf("%s✓ Deleted: %s%s\n", colorGreen, deleted, colorReset)

		case "s", "save":
			return result, true, nil

		case "c", "cancel":
			return nil, false, nil

		default:
			fmt.Printf("%sUnknown command. Use a/d/s/c%s\n", colorDim, colorReset)
		}
	}
}

// StringMapEditor は map[string]string 型の編集UI
type StringMapEditor struct {
	Path    string
	Current map[string]string
}

// NewStringMapEditor は新しいStringMapEditorを作成
func NewStringMapEditor(path string, current map[string]string) *StringMapEditor {
	return &StringMapEditor{
		Path:    path,
		Current: current,
	}
}

// Run は map[string]string 編集UIを表示し、編集結果を返す
func (e *StringMapEditor) Run() (map[string]string, bool, error) {
	StopGlobalSpinner()
	fmt.Print("\033[?25h")

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

		fmt.Printf("\n%s┌─ %s ─────────────────────────────────┐%s\n", colorCyan, e.Path, colorReset)
		fmt.Printf("│ Current entries:                     │\n")

		if len(result) == 0 {
			fmt.Printf("│   %s(empty)%s                           │\n", colorDim, colorReset)
		} else {
			for i, key := range keys {
				val := result[key]
				fmt.Printf("│   %d. %s → %s\n", i+1, key, truncateString(val, 25))
			}
		}

		fmt.Println("│                                      │")
		fmt.Println("│ [a] Add entry                        │")
		fmt.Println("│ [e] Edit entry (enter number)        │")
		fmt.Println("│ [d] Delete entry (enter number)      │")
		fmt.Println("│ [s] Save and back                    │")
		fmt.Println("│ [c] Cancel (discard changes)         │")
		fmt.Printf("%s└──────────────────────────────────────┘%s\n", colorCyan, colorReset)
		fmt.Printf("\n%sChoice:%s ", colorCyan, colorReset)

		input := readLine()
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "a", "add":
			fmt.Printf("Enter key: ")
			key := strings.TrimSpace(readLine())
			if key == "" {
				fmt.Printf("%sKey cannot be empty%s\n", colorDim, colorReset)
				continue
			}
			fmt.Printf("Enter value: ")
			value := strings.TrimSpace(readLine())
			result[key] = value
			fmt.Printf("%s✓ Added: %s → %s%s\n", colorGreen, key, value, colorReset)

		case "e", "edit":
			if len(result) == 0 {
				fmt.Printf("%sNo entries to edit%s\n", colorDim, colorReset)
				continue
			}
			fmt.Printf("Enter number to edit (1-%d): ", len(keys))
			numStr := readLine()
			num, err := strconv.Atoi(strings.TrimSpace(numStr))
			if err != nil || num < 1 || num > len(keys) {
				fmt.Printf("%sInvalid number%s\n", colorDim, colorReset)
				continue
			}
			key := keys[num-1]
			fmt.Printf("Enter new value for '%s' (current: %s): ", key, result[key])
			newValue := strings.TrimSpace(readLine())
			if newValue != "" {
				result[key] = newValue
				fmt.Printf("%s✓ Updated: %s → %s%s\n", colorGreen, key, newValue, colorReset)
			}

		case "d", "delete":
			if len(result) == 0 {
				fmt.Printf("%sNo entries to delete%s\n", colorDim, colorReset)
				continue
			}
			fmt.Printf("Enter number to delete (1-%d): ", len(keys))
			numStr := readLine()
			num, err := strconv.Atoi(strings.TrimSpace(numStr))
			if err != nil || num < 1 || num > len(keys) {
				fmt.Printf("%sInvalid number%s\n", colorDim, colorReset)
				continue
			}
			key := keys[num-1]
			delete(result, key)
			fmt.Printf("%s✓ Deleted: %s%s\n", colorGreen, key, colorReset)

		case "s", "save":
			return result, true, nil

		case "c", "cancel":
			return nil, false, nil

		default:
			fmt.Printf("%sUnknown command. Use a/e/d/s/c%s\n", colorDim, colorReset)
		}
	}
}

// StructMapEditor は map[string]struct 型の編集UI
// ProviderModels や LSP Servers などに使用
type StructMapEditor struct {
	Path      string
	FieldType config.ConfigFieldType
}

// NewStructMapEditor は新しいStructMapEditorを作成
func NewStructMapEditor(path string, fieldType config.ConfigFieldType) *StructMapEditor {
	return &StructMapEditor{
		Path:      path,
		FieldType: fieldType,
	}
}

// Run は struct map 編集UIを表示
// 注: この実装はシンプル化のため、provider_models と lsp.servers に特化
func (e *StructMapEditor) Run(cfg *config.Config) (bool, error) {
	StopGlobalSpinner()
	fmt.Print("\033[?25h")

	switch e.Path {
	case "provider_models":
		return e.runProviderModels(cfg)
	case "lsp.servers":
		return e.runLSPServers(cfg)
	default:
		fmt.Printf("%sStructMap editing not supported for: %s%s\n", colorDim, e.Path, colorReset)
		return false, nil
	}
}

func (e *StructMapEditor) runProviderModels(cfg *config.Config) (bool, error) {
	for {
		// プロバイダーリスト
		providers := make([]string, 0, len(cfg.ProviderModels))
		for p := range cfg.ProviderModels {
			providers = append(providers, p)
		}
		sort.Strings(providers)

		fmt.Printf("\n%s┌─ provider_models ────────────────────┐%s\n", colorCyan, colorReset)
		fmt.Printf("│ Configured providers:                │\n")

		if len(providers) == 0 {
			fmt.Printf("│   %s(empty)%s                           │\n", colorDim, colorReset)
		} else {
			for i, p := range providers {
				model := cfg.ProviderModels[p].DefaultModel
				fmt.Printf("│   %d. %s: %s\n", i+1, p, truncateString(model, 25))
			}
		}

		fmt.Println("│                                      │")
		fmt.Println("│ [a] Add provider                     │")
		fmt.Println("│ [1-9] Edit provider                  │")
		fmt.Println("│ [d] Delete provider                  │")
		fmt.Println("│ [s] Save and back                    │")
		fmt.Println("│ [c] Cancel                           │")
		fmt.Printf("%s└──────────────────────────────────────┘%s\n", colorCyan, colorReset)
		fmt.Printf("\n%sChoice:%s ", colorCyan, colorReset)

		input := readLine()
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "a", "add":
			fmt.Printf("Enter provider name: ")
			name := strings.TrimSpace(readLine())
			if name == "" {
				continue
			}
			fmt.Printf("Enter default model: ")
			model := strings.TrimSpace(readLine())
			if cfg.ProviderModels == nil {
				cfg.ProviderModels = make(map[string]config.ProviderModelConfig)
			}
			cfg.ProviderModels[name] = config.ProviderModelConfig{DefaultModel: model}
			fmt.Printf("%s✓ Added: %s%s\n", colorGreen, name, colorReset)

		case "d", "delete":
			if len(providers) == 0 {
				continue
			}
			fmt.Printf("Enter number to delete (1-%d): ", len(providers))
			numStr := readLine()
			num, err := strconv.Atoi(strings.TrimSpace(numStr))
			if err != nil || num < 1 || num > len(providers) {
				continue
			}
			name := providers[num-1]
			delete(cfg.ProviderModels, name)
			fmt.Printf("%s✓ Deleted: %s%s\n", colorGreen, name, colorReset)

		case "s", "save":
			return true, nil

		case "c", "cancel":
			return false, nil

		default:
			// 数字入力の場合
			if num, err := strconv.Atoi(input); err == nil && num >= 1 && num <= len(providers) {
				name := providers[num-1]
				pConfig := cfg.ProviderModels[name]
				fmt.Printf("\nEditing %s (current: %s)\n", name, pConfig.DefaultModel)
				fmt.Printf("Enter new default model: ")
				model := strings.TrimSpace(readLine())
				if model != "" {
					pConfig.DefaultModel = model
					cfg.ProviderModels[name] = pConfig
					fmt.Printf("%s✓ Updated: %s%s\n", colorGreen, name, colorReset)
				}
			}
		}
	}
}

func (e *StructMapEditor) runLSPServers(cfg *config.Config) (bool, error) {
	for {
		// サーバーリスト
		servers := make([]string, 0, len(cfg.LSP.Servers))
		for s := range cfg.LSP.Servers {
			servers = append(servers, s)
		}
		sort.Strings(servers)

		fmt.Printf("\n%s┌─ lsp.servers ────────────────────────┐%s\n", colorCyan, colorReset)
		fmt.Printf("│ Configured LSP servers:              │\n")

		if len(servers) == 0 {
			fmt.Printf("│   %s(using defaults)%s                  │\n", colorDim, colorReset)
		} else {
			for i, s := range servers {
				sConfig := cfg.LSP.Servers[s]
				status := ""
				if sConfig.Disabled {
					status = " (disabled)"
				}
				fmt.Printf("│   %d. %s: %s%s\n", i+1, s, truncateString(sConfig.Command, 20), status)
			}
		}

		fmt.Println("│                                      │")
		fmt.Println("│ [a] Add server                       │")
		fmt.Println("│ [1-9] Edit server                    │")
		fmt.Println("│ [d] Delete server                    │")
		fmt.Println("│ [s] Save and back                    │")
		fmt.Println("│ [c] Cancel                           │")
		fmt.Printf("%s└──────────────────────────────────────┘%s\n", colorCyan, colorReset)
		fmt.Printf("\n%sChoice:%s ", colorCyan, colorReset)

		input := readLine()
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "a", "add":
			fmt.Printf("Enter language (e.g., python, rust): ")
			lang := strings.TrimSpace(readLine())
			if lang == "" {
				continue
			}
			fmt.Printf("Enter command (e.g., pyright-langserver): ")
			cmd := strings.TrimSpace(readLine())
			if cfg.LSP.Servers == nil {
				cfg.LSP.Servers = make(map[string]config.LSPServerConfig)
			}
			cfg.LSP.Servers[lang] = config.LSPServerConfig{Command: cmd}
			fmt.Printf("%s✓ Added: %s%s\n", colorGreen, lang, colorReset)

		case "d", "delete":
			if len(servers) == 0 {
				continue
			}
			fmt.Printf("Enter number to delete (1-%d): ", len(servers))
			numStr := readLine()
			num, err := strconv.Atoi(strings.TrimSpace(numStr))
			if err != nil || num < 1 || num > len(servers) {
				continue
			}
			lang := servers[num-1]
			delete(cfg.LSP.Servers, lang)
			fmt.Printf("%s✓ Deleted: %s%s\n", colorGreen, lang, colorReset)

		case "s", "save":
			return true, nil

		case "c", "cancel":
			return false, nil

		default:
			// 数字入力の場合
			if num, err := strconv.Atoi(input); err == nil && num >= 1 && num <= len(servers) {
				lang := servers[num-1]
				sConfig := cfg.LSP.Servers[lang]

				fmt.Printf("\nEditing %s:\n", lang)
				fmt.Printf("  [1] command: %s\n", sConfig.Command)
				fmt.Printf("  [2] disabled: %v\n", sConfig.Disabled)
				fmt.Printf("  [b] Back\n")
				fmt.Printf("\nChoice: ")

				subInput := strings.TrimSpace(readLine())
				switch subInput {
				case "1":
					fmt.Printf("Enter new command: ")
					cmd := strings.TrimSpace(readLine())
					if cmd != "" {
						sConfig.Command = cmd
						cfg.LSP.Servers[lang] = sConfig
						fmt.Printf("%s✓ Updated command%s\n", colorGreen, colorReset)
					}
				case "2":
					sConfig.Disabled = !sConfig.Disabled
					cfg.LSP.Servers[lang] = sConfig
					fmt.Printf("%s✓ Disabled = %v%s\n", colorGreen, sConfig.Disabled, colorReset)
				}
			}
		}
	}
}

// cleanBracketedPaste は Bracketed Paste のエスケープシーケンスを除去
func cleanBracketedPaste(input string) string {
	// Bracketed Paste Mode のエスケープシーケンス
	// \x1b[200~ = 開始, \x1b[201~ = 終了
	input = strings.ReplaceAll(input, "\x1b[200~", "")
	input = strings.ReplaceAll(input, "\x1b[201~", "")
	return strings.TrimSpace(input)
}

// readLine は標準入力から1行読み取る
func readLine() string {
	var line string
	if reader := GetGlobalReader(); reader != nil {
		l, err := reader.ReadSimpleLine()
		if err == nil {
			line = l
		}
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			line = scanner.Text()
		}
	}

	// Bracketed Paste シーケンスを除去
	return cleanBracketedPaste(line)
}
