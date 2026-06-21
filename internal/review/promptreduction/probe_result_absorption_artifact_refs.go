package promptreduction

import (
	"strings"

	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
)

// ProbeResultAbsorptionArtifactRefs は absorption を適用する raw output artifact ref を表す。
type ProbeResultAbsorptionArtifactRefs struct {
	ProbeResults  map[string]string
	ProbeCommands map[reviewmodelinput.ProbeCommandResultKey]string
}

func (r ProbeResultAbsorptionArtifactRefs) probeResultRef(probeID string) string {
	if len(r.ProbeResults) == 0 {
		return ""
	}
	return strings.TrimSpace(r.ProbeResults[probeID])
}

func (r ProbeResultAbsorptionArtifactRefs) probeCommandRef(key reviewmodelinput.ProbeCommandResultKey) string {
	if len(r.ProbeCommands) == 0 {
		return ""
	}
	return strings.TrimSpace(r.ProbeCommands[key])
}
