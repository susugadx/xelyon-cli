package review

type probeCommandResolutionContext = commandResolutionContext

func resolveProbeCommandPath(command string, ctx probeCommandResolutionContext) (string, error) {
	return resolveCommandPath(command, ctx)
}
