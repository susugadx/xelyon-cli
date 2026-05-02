package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/commandruntime"
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
	args, ok := parseCommandArgs(command)
	if !ok || len(args) != 1 {
		m.setTransientStatus("Usage: /attach <path>")
		return
	}

	path, ok := normalizePastedPathToken(args[0])
	if !ok {
		m.setTransientStatus("Invalid path: " + args[0])
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		m.setTransientStatus("Attach failed: " + err.Error())
		return
	}
	if info.IsDir() {
		m.setTransientStatus("Attach failed: directories are not supported")
		return
	}

	kind := composerAttachmentFile
	if isImageAttachmentPath(path) {
		kind = composerAttachmentImage
	}
	att := composerAttachment{
		Kind:   kind,
		Source: composerAttachmentSourceCommand,
		Path:   path,
		Size:   info.Size(),
	}
	if !m.appendAttachment(att) {
		m.setTransientStatus("Already attached: " + att.basename())
		return
	}

	m.syncComposerLayout()
	m.refreshSlashSuggestions()
	m.chromeDirty = true
	m.setTransientStatus(fmt.Sprintf("Attached %s %s (#%d)", att.kindLabel(), att.basename(), len(m.attachments)))
}

func (m *Model) handleDetachCommand(command slash.Command) {
	args, ok := parseCommandArgs(command)
	if !ok || len(args) != 1 {
		m.setTransientStatus("Usage: /detach <index>")
		return
	}
	if len(m.attachments) == 0 {
		m.setTransientStatus("No attachments to detach")
		return
	}

	index, ok := parseAttachmentIndex(args[0], len(m.attachments))
	if !ok {
		m.setTransientStatus("Invalid index: " + args[0])
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
	args, ok := parseCommandArgs(command)
	if !ok || len(args) != 0 {
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

func parseCommandArgs(command slash.Command) ([]string, bool) {
	parts := commandruntime.Split(command.Input)
	if len(parts) == 0 {
		return nil, false
	}
	return parts[1:], true
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
