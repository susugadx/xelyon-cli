package cmd

import (
	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/clidoctor"
)

func newBedrockDoctorCommand() *cobra.Command {
	state := newDoctorFlagState()
	cmd := &cobra.Command{
		Use:   "bedrock",
		Short: "Diagnose AWS Bedrock configuration",
		Long: `Diagnose AWS Bedrock configuration.

Checks AWS region and credentials, provider registration, model/catalog model
resolution, Bedrock route selection, function calling settings, token/cost
metadata, and optional live smoke requests. Use --smoke for a text request,
--tool-smoke for a dummy tool call, --image-smoke for a tiny image request, and
--thinking-smoke for an extended-thinking request. ConverseStream image and
thinking smoke requests are reported as skipped because that route does not
support those request shapes yet. Use --capabilities or --require-capability to
verify resolved local capabilities without sending a live request. Use
--print-request to print sanitized request JSON without sending it.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			silenceUsage, err := clidoctor.RunBedrockDoctor(cmd.Context(), cmd.OutOrStdout(), clidoctor.BedrockOptions{
				CommonOptions: state.commonOptions(),
				Model:         state.model,
				ImageSmoke:    state.imageSmoke,
				ThinkingSmoke: state.thinkingSmoke,
			})
			return applyDoctorResult(cmd, silenceUsage, err)
		},
	}

	cmd.Flags().StringVar(&state.model, "model", "", "Bedrock model ID or configured alias for 'doctor bedrock'")
	addDoctorCatalogModelFlag(cmd, state, "Catalog model for 'doctor bedrock' capability/token/pricing policy")
	addDoctorSmokeFlag(cmd, state, "Send a live minimal Bedrock text smoke request")
	addDoctorToolSmokeFlag(cmd, state, "Send a live Bedrock smoke request that forces a dummy tool call")
	cmd.Flags().BoolVar(&state.imageSmoke, "image-smoke", false, "Send a live Bedrock image input smoke request")
	cmd.Flags().BoolVar(&state.thinkingSmoke, "thinking-smoke", false, "Send a live Bedrock extended-thinking smoke request")
	addDoctorCapabilitiesFlag(cmd, state, "Print resolved Bedrock model capabilities without sending a live request")
	addDoctorRequiredCapabilityFlag(cmd, state)
	addDoctorTimeoutFlag(cmd, state, "bedrock", "")
	addDoctorJSONFlag(cmd, state, "bedrock")
	addDoctorPrintRequestFlag(cmd, state, "bedrock")

	return cmd
}
