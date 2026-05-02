package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tui/slash"
)

func (m Model) handleAttachmentCommandSubmission(command slash.Command) (tea.Model, tea.Cmd) {
	m.recordHandledCommand(command.Input)

	switch command.ResolvedName {
	case "/attach":
		m.handleAttachCommand(command)
	case "/detach":
		m.handleDetachCommand(command)
	case "/detach-all":
		m.handleDetachAllCommand(command)
	default:
		m.setTransientStatus("Unsupported attachment command: " + command.Input)
	}

	return m, nil
}

func (m *Model) handleAttachCommand(command slash.Command) {
	if len(command.Args) != 1 {
		m.setTransientStatus("Usage: /attach <path>")
		return
	}

	normalized := normalizePastedPathToken(command.Args[0])
	if !normalized.isOK() {
		m.setTransientStatus("Invalid path: " + command.Args[0])
		return
	}

	added := m.addAttachmentFromPath(normalized.path, composerAttachmentSourceCommand)
	m.presentAttachmentAddResult(added, attachmentAddDisplayCommand, "")
}

func (m *Model) handleDetachCommand(command slash.Command) {
	if len(command.Args) != 1 {
		m.setTransientStatus("Usage: /detach <index>")
		return
	}
	if len(m.attachments) == 0 {
		m.setTransientStatus("No attachments to detach")
		return
	}

	index, ok := parseAttachmentIndex(command.Args[0], len(m.attachments))
	if !ok {
		m.setTransientStatus("Invalid index: " + command.Args[0])
		return
	}

	removed := m.attachments[index-1]
	if !m.removeAttachmentAt(index - 1) {
		m.setTransientStatus("Failed to detach attachment #" + strconv.Itoa(index))
		return
	}

	m.setTransientStatus(fmt.Sprintf("Detached %s %s (#%d)", removed.kindLabel(), removed.basename(), index))
}

func (m *Model) handleDetachAllCommand(command slash.Command) {
	if len(command.Args) != 0 {
		m.setTransientStatus("Usage: /detach-all")
		return
	}

	count := len(m.attachments)
	if !m.clearAllAttachments() {
		m.setTransientStatus("No attachments to detach")
		return
	}

	m.setTransientStatus(fmt.Sprintf("Detached %d attachment(s)", count))
}

func parseAttachmentIndex(raw string, max int) (int, bool) {
	value := strings.TrimSpace(strings.TrimPrefix(raw, "#"))
	if value == "" {
		return 0, false
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 || n > max {
		return 0, false
	}
	return n, true
}
