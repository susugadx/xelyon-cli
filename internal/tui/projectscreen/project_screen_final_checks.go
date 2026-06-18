package projectscreen

import "github.com/susugadx/xelyon-cli/internal/config"

func (ps *Screen) ensureFinalChecks() {
	if ps.pc != nil && ps.pc.FinalChecks == nil {
		ps.pc.FinalChecks = &config.FinalChecksConfig{}
		ps.tuiCreatedFinalChecks = true
	}
}

func (ps *Screen) clearTUIOnlyFinalChecksWithoutCommands() {
	if ps.pc == nil || ps.pc.FinalChecks == nil {
		return
	}
	if ps.savedHasFinalChecks || !ps.tuiCreatedFinalChecks {
		return
	}
	if len(ps.pc.FinalChecks.Commands) == 0 {
		ps.pc.FinalChecks = nil
		ps.tuiCreatedFinalChecks = false
	}
}

func (ps *Screen) hasProjectFinalCheckCommands() bool {
	return ps.pc != nil && ps.pc.FinalChecks != nil && len(ps.pc.FinalChecks.Commands) > 0
}

func (ps *Screen) finalChecksTimeoutForDisplay() int {
	if ps.pc == nil || ps.pc.FinalChecks == nil || ps.pc.FinalChecks.Timeout <= 0 {
		return 600
	}
	return ps.pc.FinalChecks.Timeout
}
