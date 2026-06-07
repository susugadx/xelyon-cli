package providerhistory

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
)

type providerHistoryDataBearingToolArtifactCandidateSpec struct {
	Surface                    rawoutputs.Surface
	Source                     rawoutputs.SourceMetadata
	Classification             rawoutputs.ClassificationMetadata
	Reason                     string
	ReplacementKind            string
	ArtifactsDisabledReason    string
	MissingArtifactReason      string
	RehydrateUnavailableReason string
	BuildPlaceholder           func(rawoutputs.RawOutputRef) string
}

func recordProviderHistoryDataBearingToolArtifactCandidate(report *ProjectionReport, policy Policy, entry ReductionCandidate, content string, spec providerHistoryDataBearingToolArtifactCandidateSpec) {
	entry.CandidateOnly = false
	entry.FutureApplyCandidate = true
	entry.Reason = spec.Reason
	entry.SuggestedReplacementKind = spec.ReplacementKind
	entry.ArtifactBackedCandidate = true
	entry.SafetyStatus = "data_bearing"
	entry.OriginalByteSize = len(content)
	entry.OriginalRuneSize = utf8.RuneCountInString(content)

	if policy.RawOutputArtifactsMode == RawOutputArtifactsDisabled {
		entry.KeepReason = spec.ArtifactsDisabledReason
		entry.FailClosedReason = entry.KeepReason
		report.Candidates = append(report.Candidates, entry)
		report.Kept = append(report.Kept, entry)
		return
	}
	if !rawOutputArtifactMaterializationAllowed(policy) {
		entry.KeepReason = providerHistoryRawOutputMaterializationReadOnlyReason
		entry.FailClosedReason = entry.KeepReason
		entry.ArtifactGateStatus = "read_only"
		report.Candidates = append(report.Candidates, entry)
		report.Kept = append(report.Kept, entry)
		return
	}
	if strings.TrimSpace(policy.SessionID) == "" {
		entry.KeepReason = "raw_output_ref_missing"
		entry.FailClosedReason = entry.KeepReason
		entry.ArtifactGateStatus = "missing_session_id"
		report.Candidates = append(report.Candidates, entry)
		report.Kept = append(report.Kept, entry)
		return
	}
	if policy.RawOutputArtifactStore == nil {
		entry.KeepReason = spec.MissingArtifactReason
		entry.FailClosedReason = "raw_output_artifact_missing"
		entry.ArtifactGateStatus = "missing_store"
		report.Candidates = append(report.Candidates, entry)
		report.Kept = append(report.Kept, entry)
		return
	}

	createReq := rawoutputs.CreateRequest{
		Surface:        spec.Surface,
		SessionID:      policy.SessionID,
		Source:         spec.Source,
		Classification: spec.Classification,
		Body:           strings.NewReader(content),
		SizeHintBytes:  int64(len(content)),
	}
	exactSourceID, ambiguous := providerHistoryLegacyRawOutputExactSourceID(policy.SessionID, spec.Surface, spec.Source, content)
	createResult, err := policy.RawOutputArtifactStore.MaterializeLegacy(context.Background(), rawoutputs.LegacyMaterializeRequest{
		CreateRequest: createReq,
		ExactSourceID: exactSourceID,
		Ambiguous:     ambiguous,
	})
	if err != nil {
		reason := providerHistoryRawOutputCreateFailureReason(err)
		entry.KeepReason = reason
		entry.FailClosedReason = reason
		entry.ArtifactGateStatus = "create_failed"
		report.Candidates = append(report.Candidates, entry)
		report.Kept = append(report.Kept, entry)
		return
	}
	verifyResult, err := policy.RawOutputArtifactStore.Verify(context.Background(), createResult.Ref)
	if err != nil || !verifyResult.OK {
		reason := providerHistoryRawOutputVerifyFailureReason(verifyResult, err)
		entry.KeepReason = reason
		entry.FailClosedReason = reason
		entry.ArtifactGateStatus = "verify_failed"
		report.Candidates = append(report.Candidates, entry)
		report.Kept = append(report.Kept, entry)
		return
	}

	entry.RawOutputRefID = createResult.Ref.RefID
	entry.ArtifactGateStatus = "verified"
	report.RawOutputRefs = append(report.RawOutputRefs, createResult.Ref)
	replacementText := ""
	if spec.BuildPlaceholder != nil {
		replacementText = spec.BuildPlaceholder(createResult.Ref)
	}
	entry.SuggestedReplacementText = replacementText
	savedBytes, savedTokens, thresholdStatus, thresholdOK := providerHistoryArtifactBackedReplacementEligibility(content, replacementText)
	entry.ThresholdStatus = thresholdStatus
	if savedBytes > 0 && savedTokens > 0 {
		entry.EstimatedSavedBytes = savedBytes
		entry.ApproxEstimatedSavedTokens = savedTokens
	}
	if !thresholdOK {
		entry.KeepReason = thresholdStatus
		entry.FailClosedReason = thresholdStatus
		report.Candidates = append(report.Candidates, entry)
		report.Kept = append(report.Kept, entry)
		return
	}
	if !policy.RawOutputRehydrateContextEnabled || !policy.ActiveContextTransportAvailable {
		entry.RehydrateGateStatus = "unsupported"
		entry.KeepReason = spec.RehydrateUnavailableReason
		entry.FailClosedReason = "raw_output_rehydrate_unsupported"
		report.Candidates = append(report.Candidates, entry)
		report.Kept = append(report.Kept, entry)
		return
	}
	entry.RehydrateGateStatus = "available"
	entry.ArtifactBackedApplyEligible = policy.RawOutputArtifactsMode == RawOutputArtifactsApply && policy.Mode == Apply
	if !entry.ArtifactBackedApplyEligible {
		if policy.RawOutputApplyDisabledReason != "" {
			entry.KeepReason = policy.RawOutputApplyDisabledReason
		} else if policy.Mode != Apply {
			entry.KeepReason = "raw_output_parent_apply_mode_disabled"
		} else {
			entry.KeepReason = "raw_output_artifacts_apply_mode_disabled"
		}
		entry.FailClosedReason = entry.KeepReason
		report.Candidates = append(report.Candidates, entry)
		report.Kept = append(report.Kept, entry)
		return
	}
	report.Candidates = append(report.Candidates, entry)
}
