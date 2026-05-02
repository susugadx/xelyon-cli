package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
)

func TestGeneratedHelpCommandsText_SyncedWithCommandCatalog(t *testing.T) {
	wantClassic := commandcatalog.RenderCommandsTextForSurface(commandcatalog.CommandSurfaceClassic)
	if GeneratedHelpCommandsText != wantClassic {
		t.Fatalf("GeneratedHelpCommandsText is stale; run `go generate ./internal/agent`")
	}

	wantTUI := commandcatalog.RenderCommandsTextForSurface(commandcatalog.CommandSurfaceTUI)
	if GeneratedTUIHelpCommandsText != wantTUI {
		t.Fatalf("GeneratedTUIHelpCommandsText is stale; run `go generate ./internal/agent`")
	}

	wantTips := commandcatalog.RenderTipsText()
	if GeneratedHelpTipsText != wantTips {
		t.Fatalf("GeneratedHelpTipsText is stale; run `go generate ./internal/agent`")
	}
}
