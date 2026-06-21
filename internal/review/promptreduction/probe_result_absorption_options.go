package promptreduction

import reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"

// CandidatePromptContextOptions は未適用の absorption 候補を prompt context options へ反映する。
func (p ProbeResultAbsorptionPlan) CandidatePromptContextOptions(opts reviewmodelinput.ProbeResultPromptContextOptions) reviewmodelinput.ProbeResultPromptContextOptions {
	if len(p.probes) > 0 {
		opts.AbsorptionCandidateProbeIDs = make(map[string]struct{}, len(p.probes))
		for _, probeID := range p.ProbeIDs() {
			opts.AbsorptionCandidateProbeIDs[probeID] = struct{}{}
		}
	}
	if len(p.commands) > 0 {
		opts.AbsorptionCandidateCommands = make(map[reviewmodelinput.ProbeCommandResultKey]struct{}, len(p.commands))
		for _, key := range p.CommandKeys() {
			opts.AbsorptionCandidateCommands[key] = struct{}{}
		}
	}
	return opts
}

// AbsorbedPromptContextOptions は適用済み absorption summary を prompt context options へ反映する。
func (p ProbeResultAbsorptionPlan) AbsorbedPromptContextOptions(opts reviewmodelinput.ProbeResultPromptContextOptions, refs ProbeResultAbsorptionArtifactRefs) reviewmodelinput.ProbeResultPromptContextOptions {
	if len(p.probes) > 0 {
		opts.AbsorbedProbeResults = make(map[string]reviewmodelinput.ProbeResultAbsorptionSummary, len(p.probes))
		for _, probeID := range p.ProbeIDs() {
			refID := refs.probeResultRef(probeID)
			candidate := p.probes[probeID].withRawArtifactRef(refID)
			if refID != "" {
				candidate.summary.Summary = probeResultAbsorptionAppliedSummary(probeID, false, 0)
			}
			opts.AbsorbedProbeResults[probeID] = candidate.summaryCopy()
		}
	}
	if len(p.commands) > 0 {
		opts.AbsorbedProbeCommands = make(map[reviewmodelinput.ProbeCommandResultKey]reviewmodelinput.ProbeResultAbsorptionSummary, len(p.commands))
		for _, key := range p.CommandKeys() {
			refID := refs.probeCommandRef(key)
			candidate := p.commands[key].withRawArtifactRef(refID)
			if refID != "" {
				candidate.summary.Summary = probeResultAbsorptionAppliedSummary(key.ProbeID, true, key.CommandIndex)
			}
			opts.AbsorbedProbeCommands[key] = candidate.summaryCopy()
		}
	}
	return opts
}
