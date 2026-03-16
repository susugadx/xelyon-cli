package agent

// normalizeToolLoopLimit converts negative values to the unlimited mode.
// Validation should report negative values, but runtime stays defensive.
func normalizeToolLoopLimit(limit int) int {
	if limit < 0 {
		return 0
	}
	return limit
}

// emitLoopWarning shows a soft warning for long-running unlimited tool loops.
// The warning is informational only and never changes control flow.
func emitLoopWarning(a *Agent, iteration int) {
	switch {
	case iteration == 100:
		dim.Fprintf(a.output(), "\n💡 100 iterations reached. Still working...\n")
	case iteration == 200:
		yellow.Fprintf(a.output(), "\n⚠️  200 iterations reached. Use Ctrl+C or /stop to abort if stuck.\n")
	case iteration > 200 && iteration%100 == 0:
		yellow.Fprintf(a.output(), "\n⚠️  %d iterations reached. Use Ctrl+C or /stop to abort if stuck.\n", iteration)
	}
}
