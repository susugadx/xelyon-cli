package cmd

import (
	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/clidoctor"
)

func newClaudeDoctorCommand() *cobra.Command {
	state := newDoctorFlagState()
	cmd := &cobra.Command{
		Use:   "claude",
		Short: "Diagnose Claude provider configuration",
		Long: `Diagnose Claude provider configuration.

Checks ANTHROPIC_API_KEY, ANTHROPIC_API_URL, provider registration, model/catalog
model resolution, Anthropic Messages route, function calling, image input,
thinking request config, context management, Claude compaction, native web
search, and token/cost metadata. ANTHROPIC_API_URL is an exact Messages
endpoint override; the official path ends with /v1/messages, and other paths
are treated as intentional proxy endpoints. Use --smoke to send a live text
request, --tool-smoke to force a dummy tool call, --image-smoke to send one
tiny image request, --thinking-smoke to send one thinking request,
--web-search-smoke to verify native Claude web search, --capabilities or
--require-capability to verify resolved local capabilities without sending a
live request, or --print-request to print sanitized request JSON without sending
it.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			silenceUsage, err := clidoctor.RunClaudeDoctor(cmd.Context(), cmd.OutOrStdout(), clidoctor.ClaudeOptions{
				CommonOptions:  state.commonOptions(),
				Model:          state.model,
				ImageSmoke:     state.imageSmoke,
				ThinkingSmoke:  state.thinkingSmoke,
				WebSearchSmoke: state.webSearchSmoke,
			})
			return applyDoctorResult(cmd, silenceUsage, err)
		},
	}

	cmd.Flags().StringVar(&state.model, "model", "", "Claude model or configured alias for 'doctor claude'")
	addDoctorCatalogModelFlag(cmd, state, "Catalog model for 'doctor claude' token/pricing policy")
	addDoctorSmokeFlag(cmd, state, "Send a live minimal Claude text smoke request")
	addDoctorToolSmokeFlag(cmd, state, "Send a live Claude smoke request that forces a dummy tool call")
	cmd.Flags().BoolVar(&state.imageSmoke, "image-smoke", false, "Send one live Claude image input smoke request")
	cmd.Flags().BoolVar(&state.thinkingSmoke, "thinking-smoke", false, "Send one live Claude thinking request smoke")
	cmd.Flags().BoolVar(&state.webSearchSmoke, "web-search-smoke", false, "Send one live Claude native web search smoke request")
	addDoctorCapabilitiesFlag(cmd, state, "Print resolved Claude model capabilities without sending a live request")
	addDoctorRequiredCapabilityFlag(cmd, state)
	addDoctorTimeoutFlag(cmd, state, "claude", "")
	addDoctorJSONFlag(cmd, state, "claude")
	addDoctorPrintRequestFlag(cmd, state, "claude")

	return cmd
}
