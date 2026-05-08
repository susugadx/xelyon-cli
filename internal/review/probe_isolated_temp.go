package review

const (
	reviewProbeHostReadOnlyTempPrefix  = "xelyon-review-host-readonly-"
	reviewProbeHostReadOnlyTempPattern = reviewProbeHostReadOnlyTempPrefix + "*"

	reviewProbeScratchTempPrefix  = "xelyon-review-scratch-"
	reviewProbeScratchTempPattern = reviewProbeScratchTempPrefix + "*"

	reviewProbeSandboxTempPrefix  = "xelyon-review-sandbox-"
	reviewProbeSandboxTempPattern = reviewProbeSandboxTempPrefix + "*"
)

var reviewProbeIsolatedTempRootPrefixes = [...]string{
	reviewProbeHostReadOnlyTempPrefix,
	reviewProbeScratchTempPrefix,
	reviewProbeSandboxTempPrefix,
}
