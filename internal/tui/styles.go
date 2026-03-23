package tui

import "github.com/charmbracelet/lipgloss"

var (
	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	inputPrefixStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("46")) // 緑（今のプロンプト色と合わせる）

	newOutputBadge = lipgloss.NewStyle().
			Background(lipgloss.Color("63")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1)
)
