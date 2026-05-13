package review

import (
	"fmt"
	"strings"
)

// ValidateReviewProbePlan は LLM probe plan schema v2 の構造契約を検証する。
func ValidateReviewProbePlan(plan ReviewProbePlan) error {
	if plan.SchemaVersion != ReviewProbePlanSchemaVersionV2 {
		return fmt.Errorf("schema_version must be %q: got %q", ReviewProbePlanSchemaVersionV2, plan.SchemaVersion)
	}
	if plan.TargetKind != TargetCurrentChanges {
		return fmt.Errorf("target_kind must be %q: got %q", TargetCurrentChanges, plan.TargetKind)
	}
	surfaceIDs, err := validateReviewProbeImpactSurfaces(plan.ImpactSurfaces)
	if err != nil {
		return err
	}
	riskIDs, err := validateReviewProbeCandidateRisks(plan.CandidateRisks, surfaceIDs)
	if err != nil {
		return err
	}
	if len(plan.Probes) > MaxReviewProbePlanProbes {
		return fmt.Errorf("probes must contain at most %d entries: got %d", MaxReviewProbePlanProbes, len(plan.Probes))
	}
	if len(plan.Probes) == 0 {
		return validateReviewProbePlanNoProbeCompletion(plan, surfaceIDs, riskIDs)
	}
	if plan.NoProbeReason != "" {
		return fmt.Errorf("no_probe_reason must be empty when probes is non-empty")
	}

	seenIDs := make(map[string]struct{}, len(plan.Probes))
	linkage := newReviewProbePlanProbeLinkageValidator(surfaceIDs, riskIDs, len(plan.Probes))
	for i, probe := range plan.Probes {
		if err := validateReviewPlannedProbe(i, probe, seenIDs, linkage); err != nil {
			return err
		}
	}
	return linkage.validateCoverage(plan.ImpactSurfaces, plan.CandidateRisks)
}

func validateReviewProbeImpactSurfaces(surfaces []ReviewProbeImpactSurface) (map[string]struct{}, error) {
	if len(surfaces) == 0 {
		return nil, fmt.Errorf("impact_surfaces must contain at least one entry")
	}

	seenIDs := make(map[string]struct{}, len(surfaces))
	for i, surface := range surfaces {
		field := fmt.Sprintf("impact_surfaces[%d]", i)
		id, err := validateReviewProbePlanID(field+".id", surface.ID)
		if err != nil {
			return nil, err
		}
		if _, exists := seenIDs[id]; exists {
			return nil, fmt.Errorf("%s.id duplicates %q", field, id)
		}
		seenIDs[id] = struct{}{}

		if err := validateReviewProbePlanRequiredText(field+".summary", surface.Summary); err != nil {
			return nil, err
		}
		if err := validateReviewProbePlanRequiredText(field+".reason", surface.Reason); err != nil {
			return nil, err
		}
		if !isKnownReviewProbeImpactSurfaceCategory(surface.Category) {
			return nil, fmt.Errorf("%s.category must be known enum value: got %q", field, surface.Category)
		}
		if !isKnownReviewProbeImpactSurfaceStatus(surface.Status) {
			return nil, fmt.Errorf("%s.status must be known enum value: got %q", field, surface.Status)
		}
		if err := validateReviewProbePlanPreProbeEvidence(field, surface.EvidenceSummary, surface.EvidenceRefs); err != nil {
			return nil, err
		}
	}
	return seenIDs, nil
}

func validateReviewProbeCandidateRisks(risks []ReviewProbeCandidateRisk, surfaceIDs map[string]struct{}) (map[string]struct{}, error) {
	seenIDs := make(map[string]struct{}, len(risks))
	for i, risk := range risks {
		field := fmt.Sprintf("candidate_risks[%d]", i)
		id, err := validateReviewProbePlanID(field+".id", risk.ID)
		if err != nil {
			return nil, err
		}
		if _, exists := seenIDs[id]; exists {
			return nil, fmt.Errorf("%s.id duplicates %q", field, id)
		}
		seenIDs[id] = struct{}{}

		if err := validateReviewProbePlanRequiredText(field+".summary", risk.Summary); err != nil {
			return nil, err
		}
		if err := validateReviewProbePlanRequiredText(field+".verification_strategy", risk.VerificationStrategy); err != nil {
			return nil, err
		}
		if !isKnownReviewGroupSeverity(risk.Severity) {
			return nil, fmt.Errorf("%s.severity must be known enum value: got %q", field, risk.Severity)
		}
		if !isKnownReviewProbeCandidateRiskStatus(risk.Status) {
			return nil, fmt.Errorf("%s.status must be known enum value: got %q", field, risk.Status)
		}
		if len(risk.SurfaceIDs) == 0 {
			return nil, fmt.Errorf("%s.surface_ids must contain at least one impact surface ID", field)
		}
		for j, surfaceID := range risk.SurfaceIDs {
			refField := fmt.Sprintf("%s.surface_ids[%d]", field, j)
			canonicalSurfaceID, err := validateReviewProbePlanID(refField, surfaceID)
			if err != nil {
				return nil, err
			}
			if _, exists := surfaceIDs[canonicalSurfaceID]; !exists {
				return nil, fmt.Errorf("%s references unknown impact surface ID %q", refField, canonicalSurfaceID)
			}
		}
		if err := validateReviewProbePlanPreProbeEvidence(field, risk.EvidenceSummary, risk.EvidenceRefs); err != nil {
			return nil, err
		}
	}
	return seenIDs, nil
}

func validateReviewProbePlanPreProbeEvidence(field, evidenceSummary string, refs []ReviewEvidenceRef) error {
	if strings.TrimSpace(evidenceSummary) == "" && len(refs) == 0 {
		return fmt.Errorf("%s requires evidence_summary or evidence_refs", field)
	}
	return validateReviewProbePlanPreProbeEvidenceRefs(field+".evidence_refs", refs)
}

func validateReviewProbePlanPreProbeEvidenceRefs(field string, refs []ReviewEvidenceRef) error {
	for i, ref := range refs {
		refField := fmt.Sprintf("%s[%d]", field, i)
		if !isReviewProbePlanPreProbeEvidenceKind(ref.Kind) {
			if !isKnownReviewEvidenceKind(ref.Kind) {
				return validateEvidenceRef(refField, ref, nil)
			}
			return fmt.Errorf("%s.kind must reference pre-probe evidence, got %q", refField, ref.Kind)
		}
		if ref.ProbeID != "" {
			return fmt.Errorf("%s.probe_id is not allowed in probe plan pre-probe evidence", refField)
		}
		if ref.CommandIndex != nil {
			return fmt.Errorf("%s.command_index is not allowed in probe plan pre-probe evidence", refField)
		}
		if err := validateEvidenceRef(refField, ref, nil); err != nil {
			return err
		}
	}
	return nil
}

func validateReviewProbePlanNoProbeCompletion(plan ReviewProbePlan, surfaceIDs, riskIDs map[string]struct{}) error {
	if strings.TrimSpace(plan.NoProbeReason) == "" {
		return fmt.Errorf("no_probe_reason must be non-empty when probes is empty")
	}
	for i, surface := range plan.ImpactSurfaces {
		field := fmt.Sprintf("impact_surfaces[%d]", i)
		if surface.Status != ReviewProbeImpactSurfaceChecked {
			return fmt.Errorf("%s.status must be %q when probes is empty: got %q", field, ReviewProbeImpactSurfaceChecked, surface.Status)
		}
		if _, exists := surfaceIDs[surface.ID]; exists && !strings.Contains(plan.NoProbeReason, surface.ID) {
			return fmt.Errorf("no_probe_reason must mention checked impact surface ID %q when probes is empty", surface.ID)
		}
	}
	for i, risk := range plan.CandidateRisks {
		field := fmt.Sprintf("candidate_risks[%d]", i)
		if risk.Status != ReviewProbeCandidateRiskCheckedByEvidence {
			return fmt.Errorf("%s.status must be %q when probes is empty: got %q", field, ReviewProbeCandidateRiskCheckedByEvidence, risk.Status)
		}
		if _, exists := riskIDs[risk.ID]; exists && !strings.Contains(plan.NoProbeReason, risk.ID) {
			return fmt.Errorf("no_probe_reason must mention checked candidate risk ID %q when probes is empty", risk.ID)
		}
	}
	return nil
}

type reviewProbePlanProbeLinkageValidator struct {
	surfaceIDs       map[string]struct{}
	riskIDs          map[string]struct{}
	linkedSurfaceIDs map[string]struct{}
	linkedRiskIDs    map[string]struct{}
}

func newReviewProbePlanProbeLinkageValidator(surfaceIDs, riskIDs map[string]struct{}, probeCount int) reviewProbePlanProbeLinkageValidator {
	return reviewProbePlanProbeLinkageValidator{
		surfaceIDs:       surfaceIDs,
		riskIDs:          riskIDs,
		linkedSurfaceIDs: make(map[string]struct{}, probeCount),
		linkedRiskIDs:    make(map[string]struct{}, probeCount),
	}
}

func (v reviewProbePlanProbeLinkageValidator) validateProbe(field string, probe ReviewPlannedProbe) error {
	if len(probe.SurfaceIDs) == 0 && len(probe.RiskIDs) == 0 {
		return fmt.Errorf("%s.surface_ids or %s.risk_ids must contain at least one referenced surface or risk ID", field, field)
	}
	for i, surfaceID := range probe.SurfaceIDs {
		refField := fmt.Sprintf("%s.surface_ids[%d]", field, i)
		canonicalSurfaceID, err := validateReviewProbePlanID(refField, surfaceID)
		if err != nil {
			return err
		}
		if _, exists := v.surfaceIDs[canonicalSurfaceID]; !exists {
			return fmt.Errorf("%s references unknown impact surface ID %q", refField, canonicalSurfaceID)
		}
		v.linkedSurfaceIDs[canonicalSurfaceID] = struct{}{}
	}
	for i, riskID := range probe.RiskIDs {
		refField := fmt.Sprintf("%s.risk_ids[%d]", field, i)
		canonicalRiskID, err := validateReviewProbePlanID(refField, riskID)
		if err != nil {
			return err
		}
		if _, exists := v.riskIDs[canonicalRiskID]; !exists {
			return fmt.Errorf("%s references unknown candidate risk ID %q", refField, canonicalRiskID)
		}
		v.linkedRiskIDs[canonicalRiskID] = struct{}{}
	}
	return nil
}

func (v reviewProbePlanProbeLinkageValidator) validateCoverage(surfaces []ReviewProbeImpactSurface, risks []ReviewProbeCandidateRisk) error {
	for i, surface := range surfaces {
		if surface.Status != ReviewProbeImpactSurfaceNeedsProbe && surface.Status != ReviewProbeImpactSurfaceUnverified {
			continue
		}
		if _, exists := v.linkedSurfaceIDs[surface.ID]; !exists {
			return fmt.Errorf("impact_surfaces[%d].id %q with status %q must be referenced by at least one probe surface_ids entry", i, surface.ID, surface.Status)
		}
	}
	for i, risk := range risks {
		if risk.Status != ReviewProbeCandidateRiskNeedsProbe && risk.Status != ReviewProbeCandidateRiskUnverified {
			continue
		}
		if _, exists := v.linkedRiskIDs[risk.ID]; !exists {
			return fmt.Errorf("candidate_risks[%d].id %q with status %q must be referenced by at least one probe risk_ids entry", i, risk.ID, risk.Status)
		}
	}
	return nil
}

func validateReviewPlannedProbe(index int, probe ReviewPlannedProbe, seenIDs map[string]struct{}, linkage reviewProbePlanProbeLinkageValidator) error {
	field := fmt.Sprintf("probes[%d]", index)

	id, err := validateReviewProbePlanID(field+".id", probe.ID)
	if err != nil {
		return err
	}
	if _, exists := seenIDs[id]; exists {
		return fmt.Errorf("%s.id duplicates %q", field, id)
	}
	seenIDs[id] = struct{}{}

	if err := validateReviewProbePlanPurpose(field+".purpose", probe.Purpose); err != nil {
		return err
	}
	if err := linkage.validateProbe(field, probe); err != nil {
		return err
	}
	if !isKnownReviewProbeMode(probe.Mode) {
		return fmt.Errorf("%s.mode must be known enum value: got %q", field, probe.Mode)
	}
	if err := validateReviewPlannedProbeFiles(field+".files", probe.Mode, probe.Files); err != nil {
		return err
	}
	if len(probe.Commands) == 0 {
		return fmt.Errorf("%s.commands must contain at least one command", field)
	}
	if len(probe.Commands) > MaxReviewProbePlanCommands {
		return fmt.Errorf("%s.commands must contain at most %d entries: got %d", field, MaxReviewProbePlanCommands, len(probe.Commands))
	}
	if probe.TimeoutSeconds < 0 {
		return fmt.Errorf("%s.timeout_seconds must be non-negative: got %d", field, probe.TimeoutSeconds)
	}
	if probe.TimeoutSeconds > MaxReviewProbePlanTimeoutSeconds {
		return fmt.Errorf("%s.timeout_seconds must be at most %d: got %d", field, MaxReviewProbePlanTimeoutSeconds, probe.TimeoutSeconds)
	}
	if probe.MaxOutputBytes < 0 {
		return fmt.Errorf("%s.max_output_bytes must be non-negative: got %d", field, probe.MaxOutputBytes)
	}
	if probe.MaxOutputBytes > MaxReviewProbePlanMaxOutputBytes {
		return fmt.Errorf("%s.max_output_bytes must be at most %d: got %d", field, MaxReviewProbePlanMaxOutputBytes, probe.MaxOutputBytes)
	}

	for i, command := range probe.Commands {
		if err := validateReviewPlannedProbeCommand(fmt.Sprintf("%s.commands[%d]", field, i), command); err != nil {
			return err
		}
	}
	return nil
}

func validateReviewProbePlanID(field, candidate string) (string, error) {
	if strings.TrimSpace(candidate) == "" {
		return "", fmt.Errorf("%s must be non-empty", field)
	}
	if strings.TrimSpace(candidate) != candidate {
		return "", fmt.Errorf("%s must be canonical ID without leading/trailing whitespace: got %q", field, candidate)
	}
	if containsAnyWhitespace(candidate) {
		return "", fmt.Errorf("%s must not include whitespace: got %q", field, candidate)
	}
	return candidate, nil
}

func validateReviewProbePlanRequiredText(field, candidate string) error {
	if strings.TrimSpace(candidate) == "" {
		return fmt.Errorf("%s must be non-empty", field)
	}
	return nil
}

func validateReviewProbePlanPurpose(field, candidate string) error {
	if strings.TrimSpace(candidate) == "" {
		return fmt.Errorf("%s must be non-empty", field)
	}
	if strings.TrimSpace(candidate) != candidate {
		return fmt.Errorf("%s must be canonical purpose without leading/trailing whitespace: got %q", field, candidate)
	}
	if len([]byte(candidate)) > MaxReviewProbePlanPurposeBytes {
		return fmt.Errorf("%s must be at most %d bytes", field, MaxReviewProbePlanPurposeBytes)
	}
	return nil
}

func validateReviewPlannedProbeCommand(field string, command ReviewPlannedProbeCommand) error {
	if strings.TrimSpace(command.Command) == "" {
		return fmt.Errorf("%s.command must be non-empty", field)
	}
	if strings.TrimSpace(command.Command) != command.Command {
		return fmt.Errorf("%s.command must be canonical command without leading/trailing whitespace: got %q", field, command.Command)
	}
	if containsAnyWhitespace(command.Command) {
		return fmt.Errorf("%s.command must not include whitespace: got %q", field, command.Command)
	}
	if strings.ContainsRune(command.Command, '\x00') {
		return fmt.Errorf("%s.command must not contain null byte", field)
	}
	if strings.ContainsAny(command.Command, `/\`) {
		return fmt.Errorf("%s.command must be a command name without path separators: got %q", field, command.Command)
	}
	for i, arg := range command.Args {
		if strings.ContainsRune(arg, '\x00') {
			return fmt.Errorf("%s.args[%d] must not contain null byte", field, i)
		}
	}
	if command.WorkDir == "" || command.WorkDir == "." {
		return nil
	}
	return validateReviewProbePlanRelativePath(field+".work_dir", command.WorkDir)
}

func validateReviewPlannedProbeFiles(field string, mode ReviewProbeMode, files []ReviewPlannedProbeFile) error {
	if mode == ReviewProbeHostReadOnly && len(files) > 0 {
		return fmt.Errorf("%s must be empty when mode is %q", field, ReviewProbeHostReadOnly)
	}
	if len(files) > MaxReviewProbePlanFiles {
		return fmt.Errorf("%s must contain at most %d entries: got %d", field, MaxReviewProbePlanFiles, len(files))
	}

	seenPaths := make(map[string]struct{}, len(files))
	totalContentBytes := 0
	for i, file := range files {
		fileField := fmt.Sprintf("%s[%d]", field, i)
		if err := validateReviewPlannedProbeFile(fileField, file); err != nil {
			return err
		}
		if _, exists := seenPaths[file.Path]; exists {
			return fmt.Errorf("%s.path duplicates %q", fileField, file.Path)
		}
		seenPaths[file.Path] = struct{}{}

		contentBytes := len([]byte(file.Content))
		if contentBytes > MaxReviewProbePlanFileContentBytes {
			return fmt.Errorf("%s.content must be at most %d bytes", fileField, MaxReviewProbePlanFileContentBytes)
		}
		totalContentBytes += contentBytes
		if totalContentBytes > MaxReviewProbePlanTotalFileContentBytes {
			return fmt.Errorf("%s content must total at most %d bytes", field, MaxReviewProbePlanTotalFileContentBytes)
		}
	}
	return nil
}

func validateReviewPlannedProbeFile(field string, file ReviewPlannedProbeFile) error {
	return validateReviewProbePlanRelativePath(field+".path", file.Path)
}

func validateReviewProbePlanRelativePath(field, candidate string) error {
	return validateReviewCanonicalRelativePath(field, candidate, reviewRelativePathValidationPolicy{
		pathKind:       "relative path",
		rejectNullByte: true,
	})
}

func isKnownReviewProbeImpactSurfaceCategory(category ReviewProbeImpactSurfaceCategory) bool {
	for _, known := range reviewProbeImpactSurfaceCategories {
		if category == known {
			return true
		}
	}
	return false
}

func isReviewProbePlanPreProbeEvidenceKind(kind string) bool {
	if !isKnownReviewEvidenceKind(kind) {
		return false
	}
	switch kind {
	case ReviewEvidenceKindProbe, ReviewEvidenceKindProbeCommand:
		return false
	default:
		return true
	}
}

func isKnownReviewProbeImpactSurfaceStatus(status ReviewProbeImpactSurfaceStatus) bool {
	for _, known := range reviewProbeImpactSurfaceStatuses {
		if status == known {
			return true
		}
	}
	return false
}

func isKnownReviewProbeCandidateRiskStatus(status ReviewProbeCandidateRiskStatus) bool {
	for _, known := range reviewProbeCandidateRiskStatuses {
		if status == known {
			return true
		}
	}
	return false
}
