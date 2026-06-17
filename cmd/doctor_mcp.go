package cmd

import (
	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/clidoctor"
)

func newMCPDoctorCommand() *cobra.Command {
	state := newDoctorFlagState()
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Diagnose MCP configuration and tool discovery",
		Long: `Diagnose MCP configuration and tool discovery.

By default this command is local-only: it reads config.yaml and an existing
~/.xelyon/mcp.json without creating files or starting MCP server processes.
Use --connect to start configured MCP servers and run initialize/tools-list.
doctor mcp never calls MCP tools.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			silenceUsage, err := clidoctor.RunMCPDoctor(cmd.Context(), cmd.OutOrStdout(), clidoctor.MCPOptions{
				JSON:         state.json,
				Connect:      state.mcpConnect,
				Server:       state.mcpServer,
				IncludeTools: state.mcpTools,
			})
			return applyDoctorResult(cmd, silenceUsage, err)
		},
	}

	cmd.Flags().BoolVar(&state.mcpConnect, "connect", false, "Start MCP servers and run initialize/tools-list without calling tools")
	cmd.Flags().StringVar(&state.mcpServer, "server", "", "Limit MCP diagnostics to one configured server name")
	cmd.Flags().BoolVar(&state.mcpTools, "tools", false, "Print tool names and visibility when used with --connect")
	addDoctorJSONFlag(cmd, state, "mcp")

	return cmd
}
