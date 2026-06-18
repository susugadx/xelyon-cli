package replaceengine

const envBatchExactLineStats = "XELYON_STR_REPLACE_BATCH_EXACT_LINE_STATS"

type batchDiffLineStatsPolicy struct {
	resolveExact bool
	tuning       myersDiffTuning
}

func resolveBatchDiffLineStatsPolicy(stdoutSuppressed bool) batchDiffLineStatsPolicy {
	forceExact := resolveEnvBoolOrDefault(envBatchExactLineStats, false)
	return batchDiffLineStatsPolicy{
		resolveExact: !stdoutSuppressed || forceExact,
		tuning:       resolveMyersDiffTuning(),
	}
}
