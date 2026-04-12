package agent

import (
	"fmt"
	"io"
)

func printHelpToWriter(out io.Writer, agent *Agent) {
	_, _ = fmt.Fprint(out, GeneratedHelpCommandsText)
	printCurrentSurfaceToolsToWriter(out, agent)
	_, _ = fmt.Fprint(out, "\n")
	_, _ = fmt.Fprint(out, GeneratedHelpTipsText)
}

func printCurrentSurfaceToolsToWriter(out io.Writer, agent *Agent) {
	policy := helpToolVisibilityPolicy(agent)
	sections := helpToolSectionsForCurrentRuntime(agent, policy)
	if len(sections.builtIn) == 0 && len(sections.mcp) == 0 {
		return
	}

	if len(sections.builtIn) > 0 {
		_, _ = fmt.Fprintf(
			out,
			"\nBuilt-in tools available in current surface (phase: %s, %s):\n",
			helpToolSurfacePhase(agent),
			helpSurfaceSummary(policy.investigationSurface),
		)
		_, _ = fmt.Fprintln(out, "  Hidden or provider-specific tools are omitted until this surface exposes them.")
		for _, summary := range sections.builtIn {
			_, _ = fmt.Fprintf(out, "  %-17s - %s\n", summary.Name, summary.Description)
		}
	}

	if len(sections.mcp) > 0 {
		_, _ = fmt.Fprintln(out, "\nConnected MCP tools available in current runtime:")
		_, _ = fmt.Fprintln(out, "  These depend on the current MCP connections and registry state.")
		for _, summary := range sections.mcp {
			_, _ = fmt.Fprintf(out, "  %-17s - %s\n", summary.Name, summary.Description)
		}
	}
}
