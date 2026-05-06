package review

const (
	reviewProbeScratchTempPrefix  = "xelyon-review-scratch-"
	reviewProbeScratchTempPattern = reviewProbeScratchTempPrefix + "*"

	reviewProbeSandboxTempPrefix  = "xelyon-review-sandbox-"
	reviewProbeSandboxTempPattern = reviewProbeSandboxTempPrefix + "*"
)

var reviewProbeIsolatedTempRootPrefixes = [...]string{
	reviewProbeScratchTempPrefix,
	reviewProbeSandboxTempPrefix,
}
