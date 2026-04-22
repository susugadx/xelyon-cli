package ui

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestNewConfigMenu_UsesDefaultRuntime(t *testing.T) {
	menu := NewConfigMenu(config.DefaultConfig(), nil)
	if menu.Runtime == nil {
		t.Fatal("Runtime should not be nil")
	}
	if menu.Config == nil {
		t.Fatal("Config should not be nil")
	}
}
