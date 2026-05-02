package commandcatalog

import (
	"fmt"
	"strings"
)

// RenderCommandsText は全 surface 向けの help 表示用コマンド一覧を返す。
func RenderCommandsText() string {
	return renderCommandsText(Commands)
}

// RenderCommandsTextForSurface は指定 surface の help 表示用コマンド一覧を返す。
func RenderCommandsTextForSurface(surface CommandSurface) string {
	return renderCommandsText(CommandsForSurface(surface))
}

func renderCommandsText(commands []CommandInfo) string {
	var b strings.Builder
	b.WriteString("Commands:\n")
	for _, cmd := range commands {
		name := cmd.Name
		if len(cmd.Aliases) > 0 {
			name += ", " + strings.Join(cmd.Aliases, ", ")
		}
		if cmd.Args != "" {
			name += " " + cmd.Args
		}
		fmt.Fprintf(&b, "  %-25s - %s\n", name, cmd.Description)
		for _, sub := range cmd.SubCommands {
			fmt.Fprintf(&b, "                            %s - %s\n", sub.Name, sub.Description)
		}
	}
	return b.String()
}

// RenderTipsText は help 表示用の Tips 一覧を返す。
func RenderTipsText() string {
	var b strings.Builder
	b.WriteString("Tips:\n")
	for _, tip := range Tips {
		fmt.Fprintf(&b, "  - %s\n", tip)
	}
	return b.String()
}
