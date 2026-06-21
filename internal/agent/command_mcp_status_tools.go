package agent

import (
	"fmt"
	"github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/mcpnames"
	"io"
	"sort"
	"strings"
)

func printMCPStatusToolSamples(out io.Writer, surface mcpToolSurfaceSelection) {
	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "🧰 MCP tools")
	printMCPStatusVisibleSamples(out, surface)
	printMCPStatusOmittedSamples(out, surface)
}

func printMCPStatusVisibleSamples(out io.Writer, surface mcpToolSurfaceSelection) {
	samples := mcpStatusVisibleSamples(surface.selected, mcpStatusSampleLimit)
	if len(samples) == 0 {
		dim.Fprintln(out, "  Visible: none")
		return
	}
	_, _ = fmt.Fprintf(out, "  Visible: %d\n", len(surface.selected))
	for _, sample := range samples {
		_, _ = fmt.Fprintf(out, "    - %s (%s)\n", sample.exportedName, sample.approval)
	}
	if remaining := len(surface.selected) - len(samples); remaining > 0 {
		_, _ = fmt.Fprintf(out, "    ... %d more visible MCP tools\n", remaining)
	}
}

func printMCPStatusOmittedSamples(out io.Writer, surface mcpToolSurfaceSelection) {
	samples := mcpStatusOmittedSamples(surface.omitted, mcpStatusSampleLimit)
	if len(samples) == 0 {
		dim.Fprintln(out, "  Omitted: none")
		return
	}
	_, _ = fmt.Fprintf(out, "  Omitted: %d\n", len(surface.omitted))
	for _, sample := range samples {
		_, _ = fmt.Fprintf(out, "    - %s (%s)\n", sample.exportedName, sample.reason)
	}
	if remaining := len(surface.omitted) - len(samples); remaining > 0 {
		_, _ = fmt.Fprintf(out, "    ... %d more omitted MCP tools\n", remaining)
	}
}

func mcpStatusVisibleSamples(tools []mcp.MCPTool, limit int) []mcpStatusToolSample {
	if limit <= 0 || len(tools) == 0 {
		return nil
	}
	samples := make([]mcpStatusToolSample, 0, len(tools))
	for _, tool := range tools {
		samples = append(samples, mcpStatusToolSample{
			exportedName: mcpnames.ExportedToolName(tool.ServerName, tool.Name),
			approval:     tool.ApprovalMode().String(),
		})
	}
	sort.SliceStable(samples, func(i, j int) bool {
		return samples[i].exportedName < samples[j].exportedName
	})
	if len(samples) > limit {
		return samples[:limit]
	}
	return samples
}

func mcpStatusOmittedSamples(omissions []mcpToolSurfaceOmission, limit int) []mcpStatusToolSample {
	if limit <= 0 || len(omissions) == 0 {
		return nil
	}
	samples := make([]mcpStatusToolSample, 0, len(omissions))
	for _, omission := range omissions {
		if strings.TrimSpace(omission.exportedName) == "" {
			continue
		}
		samples = append(samples, mcpStatusToolSample{
			exportedName: omission.exportedName,
			reason:       omission.reason,
		})
	}
	sort.SliceStable(samples, func(i, j int) bool {
		return samples[i].exportedName < samples[j].exportedName
	})
	if len(samples) > limit {
		return samples[:limit]
	}
	return samples
}

func mcpStatusSurfaceCounts(surface mcpToolSurfaceSelection) (map[string]int, map[string]int) {
	visible := make(map[string]int)
	for _, tool := range surface.selected {
		visible[tool.ServerName]++
	}
	omitted := make(map[string]int)
	for _, omission := range surface.omitted {
		if strings.TrimSpace(omission.serverName) == "" {
			continue
		}
		omitted[omission.serverName]++
	}
	return visible, omitted
}
