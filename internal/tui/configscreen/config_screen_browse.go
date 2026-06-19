package configscreen

import tea "github.com/charmbracelet/bubbletea"

func (cs *Screen) handleBrowseKey(msg tea.KeyMsg, layout configLayout, providerConfigKey string) (configCommand, tea.Cmd) {
	s := msg.String()

	switch {
	case msg.Type == tea.KeyEsc:
		return cs.handleBrowseEscape(), nil
	case s == "q":
		return cs.tryClose(), nil
	case s == "s":
		return configCommandSave, nil
	case s == "/":
		cs.beginFilter()
		return configCommandNone, nil
	case s == "r":
		cs.resetSelectedFieldToDefault(providerConfigKey)
		return configCommandNone, nil
	case msg.Type == tea.KeyUp || s == "k":
		cs.navUp(layout)
		return configCommandNone, nil
	case msg.Type == tea.KeyDown || s == "j":
		cs.navDown(layout)
		return configCommandNone, nil
	case msg.Type == tea.KeyLeft || s == "h":
		cs.navLeft()
		return configCommandNone, nil
	case msg.Type == tea.KeyRight || s == "l":
		cs.navRight(layout)
		return configCommandNone, nil
	case s == " ":
		cs.spaceToggle(providerConfigKey)
		return configCommandNone, nil
	case isEnterKey(msg):
		return cs.handleBrowseEnter(layout, providerConfigKey)
	default:
		return configCommandNone, nil
	}
}

func (cs *Screen) beginFilter() {
	cs.filterMode = true
	cs.filterInput.SetValue("")
	cs.filterInput.Focus()
}
