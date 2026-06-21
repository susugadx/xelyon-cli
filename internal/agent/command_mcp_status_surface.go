package agent

import (
	"fmt"
	"github.com/susugadx/xelyon-cli/internal/mcpsurface"
	"io"
	"sort"
)

func printMCPStatusToolSurfaceAnalysis(out io.Writer, analysis mcpsurface.Report) {
	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "📊 MCP tool surface")
	if analysis.TotalTools == 0 {
		dim.Fprintln(out, "  No MCP tool surface is visible in this session")
		return
	}
	_, _ = fmt.Fprintf(
		out,
		"  Summary: %d visible / %d registered / %d total, %d omitted, %d estimated tokens (all analyzed), %s schema\n",
		analysis.VisibleTools,
		analysis.RegisteredTools,
		analysis.TotalTools,
		analysis.OmittedTools,
		analysis.EstimatedTokens,
		mcpsurface.FormatBytes(analysis.SchemaBytes),
	)
	printMCPStatusTopServers(out, analysis)
	_, _ = fmt.Fprintf(out, "  Top omitted reasons: %s\n", mcpsurface.FormatReasonCounts(analysis.OmittedReasons, mcpStatusSurfaceLimit))
	printMCPStatusToolMetrics(out, "Largest schema tools", analysis.LargestSchemaTools, true)
	printMCPStatusToolMetrics(out, "Highest estimated token tools", analysis.HighestEstimatedTokenTools, false)
	printMCPStatusRecommendations(out, analysis)
}

func printMCPStatusTopServers(out io.Writer, analysis mcpsurface.Report) {
	servers := append([]mcpsurface.ServerSummary(nil), analysis.Servers...)
	sort.SliceStable(servers, func(i, j int) bool {
		if servers[i].EstimatedTokens != servers[j].EstimatedTokens {
			return servers[i].EstimatedTokens > servers[j].EstimatedTokens
		}
		if servers[i].SchemaBytes != servers[j].SchemaBytes {
			return servers[i].SchemaBytes > servers[j].SchemaBytes
		}
		if servers[i].RegisteredTools != servers[j].RegisteredTools {
			return servers[i].RegisteredTools > servers[j].RegisteredTools
		}
		return servers[i].ServerName < servers[j].ServerName
	})
	if len(servers) > mcpStatusSurfaceLimit {
		servers = servers[:mcpStatusSurfaceLimit]
	}
	_, _ = fmt.Fprintln(out, "  Top heavy servers:")
	for _, server := range servers {
		_, _ = fmt.Fprintf(
			out,
			"    - %s: %d tokens, %s schema, %d visible / %d omitted\n",
			server.ServerName,
			server.EstimatedTokens,
			mcpsurface.FormatBytes(server.SchemaBytes),
			server.VisibleTools,
			server.OmittedTools,
		)
	}
}

func printMCPStatusToolMetrics(out io.Writer, label string, metrics []mcpsurface.ToolMetric, schema bool) {
	_, _ = fmt.Fprintf(out, "  %s:\n", label)
	if len(metrics) == 0 {
		dim.Fprintln(out, "    none")
		return
	}
	for _, metric := range metrics {
		name := mcpsurface.MetricName(metric)
		if schema {
			_, _ = fmt.Fprintf(out, "    - %s: %s schema\n", name, mcpsurface.FormatBytes(metric.SchemaBytes))
			continue
		}
		_, _ = fmt.Fprintf(out, "    - %s: %d tokens\n", name, metric.EstimatedTokens)
	}
}

func printMCPStatusRecommendations(out io.Writer, analysis mcpsurface.Report) {
	_, _ = fmt.Fprintln(out, "  Recommendations:")
	if len(analysis.Recommendations) == 0 {
		dim.Fprintln(out, "    none")
		return
	}
	_, _ = fmt.Fprintln(out, "    1. Narrow ~/.xelyon/mcp.json mcpServers.<server>.tools.include/exclude:")
	for _, recommendation := range analysis.Recommendations {
		_, _ = fmt.Fprintf(out, "      - %s: %s\n", recommendation.ServerName, recommendation.Reason)
		_, _ = fmt.Fprintf(out, "        mcpServers fragment: %s\n", mcpsurface.IncludeSnippet(recommendation))
	}
	budget := mcpsurface.DefaultBudget()
	if analysis.EffectiveBudget != nil {
		budget = *analysis.EffectiveBudget
	}
	_, _ = fmt.Fprintln(out, "    2. If the server is intentionally large, raise ~/.xelyon/config.yaml mcp.surface_budget:")
	_, _ = fmt.Fprintf(out, "       %s\n", mcpsurface.SurfaceBudgetSnippet(budget))
}
