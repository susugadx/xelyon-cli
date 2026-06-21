package promptreduction

import (
	"strings"

	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

// ProbeResultAbsorptionPlan は report evidence に吸収済みの probe result 削減候補を表す。
type ProbeResultAbsorptionPlan struct {
	probes   map[string]ProbeResultAbsorptionCandidate
	commands map[reviewmodelinput.ProbeCommandResultKey]ProbeResultAbsorptionCandidate
}

// ProbeResultAbsorptionCandidate は probe result prompt context の削減候補を表す。
type ProbeResultAbsorptionCandidate struct {
	summary          reviewmodelinput.ProbeResultAbsorptionSummary
	originalBytes    int
	replacementBytes int
	savedBytes       int
	savedTokens      int
}

// BuildProbeResultAbsorptionPlan は report evidence から probe result absorption 候補を構築する。
func BuildProbeResultAbsorptionPlan(report reviewreport.ReviewReport, probeResults []reviewprobe.ReviewProbeResult) ProbeResultAbsorptionPlan {
	plan := ProbeResultAbsorptionPlan{}
	if strings.TrimSpace(report.SchemaVersion) == "" || report.ScopeCoverage == nil || len(probeResults) == 0 {
		return plan
	}
	refs := probeResultAbsorptionRefs(report)
	for _, result := range probeResults {
		probeID := strings.TrimSpace(result.ID)
		if probeID == "" {
			continue
		}
		if !probeResultSafeForAbsorbedPrompt(result) {
			continue
		}

		probeAbsorbed := false
		absorbedBy := refs.safeProbes[probeID]
		if len(absorbedBy) > 0 && !refs.probeUnsafeForFullAbsorption(probeID) {
			if candidate, ok := buildProbeResultAbsorptionCandidate(probeID, absorbedBy, probeResultPromptOriginalBytes(result), false, 0); ok {
				if plan.probes == nil {
					plan.probes = make(map[string]ProbeResultAbsorptionCandidate)
				}
				plan.probes[probeID] = candidate
				probeAbsorbed = true
			}
		}

		if probeAbsorbed {
			continue
		}

		for commandIndex, command := range result.CommandResults {
			key := reviewmodelinput.ProbeCommandResultKey{ProbeID: probeID, CommandIndex: commandIndex}
			absorbedBy := refs.safeCommands[key]
			if len(absorbedBy) == 0 || refs.commandUnsafeForAbsorption(key) {
				continue
			}
			if !probeCommandResultSafeForAbsorbedPrompt(command) {
				continue
			}
			originalBytes := probeCommandResultPromptOriginalBytes(command)
			if candidate, ok := buildProbeResultAbsorptionCandidate(probeID, absorbedBy, originalBytes, true, commandIndex); ok {
				if plan.commands == nil {
					plan.commands = make(map[reviewmodelinput.ProbeCommandResultKey]ProbeResultAbsorptionCandidate)
				}
				plan.commands[key] = candidate
			}
		}
	}
	return plan
}

// Empty は absorption 候補がないかを返す。
func (p ProbeResultAbsorptionPlan) Empty() bool {
	return len(p.probes) == 0 && len(p.commands) == 0
}

// ProbeCount は full probe result absorption 候補数を返す。
func (p ProbeResultAbsorptionPlan) ProbeCount() int {
	return len(p.probes)
}

// CommandCount は command-level absorption 候補数を返す。
func (p ProbeResultAbsorptionPlan) CommandCount() int {
	return len(p.commands)
}

// ProbeIDs は full probe result absorption 候補 ID を安定順で返す。
func (p ProbeResultAbsorptionPlan) ProbeIDs() []string {
	return sortedProbeAbsorptionProbeIDs(p.probes)
}

// CommandKeys は command-level absorption 候補 key を安定順で返す。
func (p ProbeResultAbsorptionPlan) CommandKeys() []reviewmodelinput.ProbeCommandResultKey {
	return sortedProbeCommandAbsorptionKeys(p.commands)
}

// ProbeCandidate は probe ID に対応する absorption 候補を返す。
func (p ProbeResultAbsorptionPlan) ProbeCandidate(probeID string) (ProbeResultAbsorptionCandidate, bool) {
	candidate, ok := p.probes[probeID]
	return candidate, ok
}

// CommandCandidate は command key に対応する absorption 候補を返す。
func (p ProbeResultAbsorptionPlan) CommandCandidate(key reviewmodelinput.ProbeCommandResultKey) (ProbeResultAbsorptionCandidate, bool) {
	candidate, ok := p.commands[key]
	return candidate, ok
}
