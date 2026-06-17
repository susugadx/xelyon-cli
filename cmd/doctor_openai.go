package cmd

import (
	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/clidoctor"
)

func newOpenAIDoctorCommand() *cobra.Command {
	state := newDoctorFlagState()
	cmd := &cobra.Command{
		Use:   "openai",
		Short: "Diagnose OpenAI provider configuration",
		Long: `Diagnose OpenAI provider configuration.

Checks OPENAI_API_KEY, OpenAI Chat Completions and Responses endpoints,
provider registration, model/catalog model resolution, route selection,
function calling settings, token/cost metadata, and Responses API retention
settings. Use --smoke to send a minimal live request. Use --tool-smoke to
force a dummy tool call when function calling is enabled. Use
--retention-smoke to verify a Responses API previous_response_id chain. Use
--capabilities to print resolved model capabilities without sending a live
request. Use --require-capability to fail when a resolved local capability is
missing. Use --print-request to print the sanitized smoke request JSON without
sending it.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			silenceUsage, err := clidoctor.RunOpenAIDoctor(cmd.Context(), cmd.OutOrStdout(), clidoctor.OpenAIOptions{
				CommonOptions:  state.commonOptions(),
				Model:          state.model,
				RetentionSmoke: state.retentionSmoke,
			})
			return applyDoctorResult(cmd, silenceUsage, err)
		},
	}

	cmd.Flags().StringVar(&state.model, "model", "", "OpenAI model or configured alias for 'doctor openai'")
	addDoctorCatalogModelFlag(cmd, state, "Catalog model for 'doctor openai' capability/token/pricing policy")
	addDoctorSmokeFlag(cmd, state, "Send a live minimal OpenAI text smoke request")
	addDoctorToolSmokeFlag(cmd, state, "Send a live OpenAI smoke request that forces a dummy tool call")
	addDoctorCapabilitiesFlag(cmd, state, "Print resolved OpenAI model capabilities without sending a live request")
	addDoctorRequiredCapabilityFlag(cmd, state)
	cmd.Flags().BoolVar(&state.retentionSmoke, "retention-smoke", false, "Send live OpenAI Responses API requests that verify previous_response_id retention")
	addDoctorTimeoutFlag(cmd, state, "openai", "")
	addDoctorJSONFlag(cmd, state, "openai")
	addDoctorPrintRequestFlag(cmd, state, "openai")

	return cmd
}
