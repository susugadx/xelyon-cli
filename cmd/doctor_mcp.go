package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/config"
	mcpdiag "github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/mcpsurface"
)

func newMCPDoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Diagnose MCP configuration and tool discovery",
		Long: `Diagnose MCP configuration and tool discovery.

By default this command is local-only: it reads config.yaml and an existing
~/.xelyon/mcp.json without creating files or starting MCP server processes.
Use --connect to start configured MCP servers and run initialize/tools-list.
doctor mcp never calls MCP tools.`,
		Args: cobra.NoArgs,
		RunE: runMCPDoctorInvocation,
	}

	cmd.Flags().BoolVar(&doctorMCPConnectFlag, "connect", false, "Start MCP servers and run initialize/tools-list without calling tools")
	cmd.Flags().StringVar(&doctorMCPServerFlag, "server", "", "Limit MCP diagnostics to one configured server name")
	cmd.Flags().BoolVar(&doctorMCPToolsFlag, "tools", false, "Print tool names and visibility when used with --connect")
	addDoctorJSONFlag(cmd, "mcp")

	return cmd
}

func runMCPDoctorInvocation(cmd *cobra.Command, args []string) error {
	cfg, loadErr := config.LoadConfigReadOnly()
	if loadErr != nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()

	report := mcpdiag.Diagnose(cmd.Context(), mcpdiag.DiagnosticOptions{
		MCPEnabled:   cfg.MCP.Enabled,
		MCPHeadless:  cfg.MCP.Headless,
		Connect:      doctorMCPConnectFlag,
		Server:       doctorMCPServerFlag,
		IncludeTools: doctorMCPToolsFlag,
	})
	if loadErr != nil {
		report.Checks = append([]mcpdiag.DiagnosticCheck{{
			Name:       "config_yaml",
			Status:     mcpdiag.DiagnosticStatusWarn,
			Message:    "failed to load config.yaml; using defaults for diagnostics",
			Detail:     loadErr.Error(),
			Suggestion: "Fix ~/.xelyon/config.yaml, then rerun this command",
		}}, report.Checks...)
	}

	if doctorJSONFlag {
		if err := renderMCPDoctorJSON(cmd.OutOrStdout(), report); err != nil {
			return err
		}
	} else {
		renderMCPDoctorText(cmd.OutOrStdout(), report)
	}

	if report.HasFailures() {
		cmd.SilenceUsage = true
		return fmt.Errorf("MCP diagnostics failed")
	}
	return nil
}

func renderMCPDoctorJSON(w io.Writer, report mcpdiag.DiagnosticReport) error {
	return renderDoctorJSON(w, report)
}

func renderMCPDoctorText(w io.Writer, report mcpdiag.DiagnosticReport) {
	fmt.Fprintln(w, "MCP doctor")
	fmt.Fprintf(w, "Status: %s\n", strings.ToUpper(string(report.SummaryStatus())))
	fmt.Fprintf(w, "Config: %s (%s)\n", mcpConfigPathText(report.ConfigPath), mcpConfigPresenceText(report.ConfigExists))
	fmt.Fprintf(w, "Runtime: enabled=%t headless=%t\n", report.RuntimeEnabled, report.RuntimeHeadless)
	fmt.Fprintf(w, "Connect: %t\n", report.Connect)
	if strings.TrimSpace(report.ServerFilter) != "" {
		fmt.Fprintf(w, "Server filter: %s\n", report.ServerFilter)
	}
	fmt.Fprintln(w)

	renderDoctorChecks(w, mcpDoctorCheckLines(report.Checks))
	if len(report.Servers) == 0 {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Servers:")
	for _, server := range report.Servers {
		renderMCPDoctorServer(w, server)
	}
}

func renderMCPDoctorServer(w io.Writer, server mcpdiag.DiagnosticServerReport) {
	fmt.Fprintf(
		w,
		"- %s: disabled=%t command=%s args=%d env_keys=%d approval=%s startup=%ds tool=%ds\n",
		server.Name,
		server.Disabled,
		doctorEmptyText(server.Command),
		server.ArgCount,
		len(server.EnvKeys),
		server.Approval,
		server.StartupTimeoutSeconds,
		server.ToolTimeoutSeconds,
	)
	if len(server.Include) > 0 {
		fmt.Fprintf(w, "  include: %s\n", strings.Join(server.Include, ", "))
	}
	if len(server.Exclude) > 0 {
		fmt.Fprintf(w, "  exclude: %s\n", strings.Join(server.Exclude, ", "))
	}
	if len(server.EnvKeys) > 0 {
		fmt.Fprintf(w, "  env keys: %s\n", strings.Join(server.EnvKeys, ", "))
	}
	if server.ToolApprovalCount > 0 {
		fmt.Fprintf(w, "  tool approvals: %d\n", server.ToolApprovalCount)
	}
	if server.Connection != nil {
		renderMCPDoctorConnection(w, *server.Connection)
	}
	if len(server.Checks) > 0 {
		renderDoctorChecks(w, indentDoctorCheckLines(mcpDoctorCheckLines(server.Checks), "  "))
	}
	if len(server.Tools) > 0 {
		fmt.Fprintln(w, "  tools:")
		for _, tool := range server.Tools {
			renderMCPDoctorTool(w, tool)
		}
	}
}

func renderMCPDoctorConnection(w io.Writer, connection mcpdiag.DiagnosticConnectionReport) {
	fmt.Fprintf(
		w,
		"  connection: attempted=%t status=%s raw=%d registered=%d skipped=%d filtered=%d denied=%d collisions=%d\n",
		connection.Attempted,
		connection.Status,
		connection.RawToolCount,
		connection.RegisteredToolCount,
		connection.SkippedToolCount,
		connection.FilteredToolCount,
		connection.DeniedToolCount,
		connection.CollisionToolCount,
	)
	if len(connection.UnknownToolApprovals) > 0 {
		fmt.Fprintf(w, "  unknown toolApprovals: %s\n", strings.Join(connection.UnknownToolApprovals, ", "))
	}
	if len(connection.UnknownIncludes) > 0 {
		fmt.Fprintf(w, "  unknown include: %s\n", strings.Join(connection.UnknownIncludes, ", "))
	}
	if len(connection.UnknownExcludes) > 0 {
		fmt.Fprintf(w, "  unknown exclude: %s\n", strings.Join(connection.UnknownExcludes, ", "))
	}
	if strings.TrimSpace(connection.Error) != "" {
		fmt.Fprintf(w, "  connection error: %s\n", connection.Error)
	}
	if connection.ToolSurface != nil {
		renderMCPDoctorToolSurface(w, *connection.ToolSurface)
	}
}

func renderMCPDoctorTool(w io.Writer, tool mcpdiag.DiagnosticToolReport) {
	if tool.Visible {
		fmt.Fprintf(w, "    OK   %s -> %s approval=%s\n", tool.Name, tool.ExportedName, tool.Approval)
		return
	}
	fmt.Fprintf(w, "    SKIP %s reason=%s\n", tool.Name, tool.HiddenReason)
}

func renderMCPDoctorToolSurface(w io.Writer, report mcpsurface.Report) {
	fmt.Fprintf(
		w,
		"  tool surface: visible=%d registered=%d total=%d omitted=%d estimated_tokens=%d schema=%s\n",
		report.VisibleTools,
		report.RegisteredTools,
		report.TotalTools,
		report.OmittedTools,
		report.EstimatedTokens,
		mcpsurface.FormatBytes(report.SchemaBytes),
	)
	fmt.Fprintf(w, "  top omitted reasons: %s\n", mcpsurface.FormatReasonCounts(report.OmittedReasons, 0))
	if len(report.LargestSchemaTools) > 0 {
		fmt.Fprintln(w, "  largest schema tools:")
		for _, metric := range report.LargestSchemaTools {
			fmt.Fprintf(w, "    - %s: %s schema\n", mcpsurface.MetricName(metric), mcpsurface.FormatBytes(metric.SchemaBytes))
		}
	}
	if len(report.HighestEstimatedTokenTools) > 0 {
		fmt.Fprintln(w, "  highest estimated token tools:")
		for _, metric := range report.HighestEstimatedTokenTools {
			fmt.Fprintf(w, "    - %s: %d tokens\n", mcpsurface.MetricName(metric), metric.EstimatedTokens)
		}
	}
	if len(report.Recommendations) == 0 {
		return
	}
	fmt.Fprintln(w, "  recommendations:")
	for _, recommendation := range report.Recommendations {
		fmt.Fprintf(w, "    - %s: %s\n", recommendation.ServerName, recommendation.Reason)
		fmt.Fprintf(w, "      ~/.xelyon/mcp.json mcpServers fragment: %s\n", mcpsurface.IncludeSnippet(recommendation))
	}
}

func mcpDoctorCheckLines(checks []mcpdiag.DiagnosticCheck) []doctorCheckLine {
	lines := make([]doctorCheckLine, 0, len(checks))
	for _, check := range checks {
		lines = append(lines, doctorCheckLine{
			Status:     string(check.Status),
			Name:       check.Name,
			Message:    check.Message,
			Detail:     check.Detail,
			Suggestion: check.Suggestion,
		})
	}
	return lines
}

func indentDoctorCheckLines(checks []doctorCheckLine, prefix string) []doctorCheckLine {
	indented := make([]doctorCheckLine, 0, len(checks))
	for _, check := range checks {
		check.Name = prefix + check.Name
		indented = append(indented, check)
	}
	return indented
}

func mcpConfigPresenceText(exists bool) string {
	if exists {
		return "found"
	}
	return "missing"
}

func mcpConfigPathText(path string) string {
	if strings.TrimSpace(path) == "" {
		return "(unresolved)"
	}
	return path
}

func doctorEmptyText(value string) string {
	if strings.TrimSpace(value) == "" {
		return "(empty)"
	}
	return value
}
