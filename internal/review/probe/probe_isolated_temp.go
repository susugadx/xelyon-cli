package probe

const (
	reviewProbeHostReadOnlyTempPrefix  = "xelyon-review-host-readonly-"
	reviewProbeHostReadOnlyTempPattern = reviewProbeHostReadOnlyTempPrefix + "*"

	reviewProbeScratchTempPrefix  = "xelyon-review-scratch-"
	reviewProbeScratchTempPattern = reviewProbeScratchTempPrefix + "*"

	reviewProbeSandboxTempPrefix  = "xelyon-review-sandbox-"
	reviewProbeSandboxTempPattern = reviewProbeSandboxTempPrefix + "*"
)

const (
	// ReviewProbeHostReadOnlyTempPrefix は host_readonly 一時 root の prefix。
	ReviewProbeHostReadOnlyTempPrefix = reviewProbeHostReadOnlyTempPrefix
	// ReviewProbeScratchTempPrefix は scratch_only 一時 root の prefix。
	ReviewProbeScratchTempPrefix = reviewProbeScratchTempPrefix
	// ReviewProbeSandboxTempPrefix は repo_sandbox 一時 root の prefix。
	ReviewProbeSandboxTempPrefix = reviewProbeSandboxTempPrefix
)

var reviewProbeIsolatedTempRootPrefixes = [...]string{
	reviewProbeHostReadOnlyTempPrefix,
	reviewProbeScratchTempPrefix,
	reviewProbeSandboxTempPrefix,
}

// ReviewProbeIsolatedTempRootPrefixes は probe が使う一時 root prefix 一覧を返す。
func ReviewProbeIsolatedTempRootPrefixes() []string {
	return append([]string(nil), reviewProbeIsolatedTempRootPrefixes[:]...)
}
