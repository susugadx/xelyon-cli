package uiconfig

import (
	"errors"
	"fmt"
	"io"

	"github.com/susugadx/xelyon-cli/internal/config"
)

var errConfigMenuBack = errors.New("back")

type configFieldSelection struct {
	back       bool
	fieldIndex int
	hasSelect  bool
}

// ShowFieldList はカテゴリ内のフィールドリストを表示する。
func (m *ConfigMenu) ShowFieldList(cat *config.ConfigCategory) (*config.ConfigField, error) {
	ctx := newConfigPromptContext(m.Runtime)
	promptIO := ctx.promptIO
	out := ctx.out

	for {
		m.renderConfigFieldList(out, cat)
		input := readConfigMenuInput(&promptIO)
		selection := resolveConfigFieldSelection(input, len(cat.Fields))
		if selection.back {
			return nil, errConfigMenuBack
		}
		if selection.hasSelect {
			return &cat.Fields[selection.fieldIndex], nil
		}
	}
}

func (m *ConfigMenu) renderConfigFieldList(out io.Writer, cat *config.ConfigCategory) {
	_, _ = fmt.Fprintf(out, "\n%s── %s %s ───────────────────────────────%s\n\n",
		colorCyan, cat.Icon, cat.DisplayName, colorReset)

	for i, field := range cat.Fields {
		num := i + 1
		if num == 10 {
			num = 0
		}

		currentVal := formatValue(field.Current)
		_, _ = fmt.Fprintf(out, "  [%d] %-20s = %s\n", num, field.DisplayName, truncateString(currentVal, 15))
	}

	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "  [b] Back")
	_, _ = fmt.Fprintf(out, "\n%sSelect field:%s ", colorCyan, colorReset)
}

func resolveConfigFieldSelection(input string, fieldCount int) configFieldSelection {
	switch input {
	case "b", "back":
		return configFieldSelection{back: true}
	default:
		idx, ok := parseMenuNumberWithZeroAsTen(input, fieldCount)
		if !ok {
			return configFieldSelection{}
		}
		return configFieldSelection{
			fieldIndex: idx,
			hasSelect:  true,
		}
	}
}
