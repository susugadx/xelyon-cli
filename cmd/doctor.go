package cmd

import (
	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/clidoctor"
)

func newDoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run provider configuration diagnostics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newAzureDoctorCommand())
	cmd.AddCommand(newBedrockDoctorCommand())
	cmd.AddCommand(newClaudeDoctorCommand())
	cmd.AddCommand(newDeepSeekDoctorCommand())
	cmd.AddCommand(newGeminiDoctorCommand())
	cmd.AddCommand(newGroqDoctorCommand())
	cmd.AddCommand(newKimiDoctorCommand())
	cmd.AddCommand(newMCPDoctorCommand())
	cmd.AddCommand(newOllamaDoctorCommand())
	cmd.AddCommand(newOpenAIDoctorCommand())
	cmd.AddCommand(newOpenAISubscriptionDoctorCommand())
	cmd.AddCommand(newOpenRouterDoctorCommand())
	return cmd
}

func newAzureDoctorCommand() *cobra.Command {
	state := newDoctorFlagState()
	cmd := &cobra.Command{
		Use:   "azure",
		Short: "Diagnose Azure OpenAI configuration",
		Long: `Diagnose Azure OpenAI configuration.

Checks base URL, authentication, deployment resolution, catalog model,
function calling settings, and Responses API retention settings. Use --smoke
to send a minimal live Responses API request. Use --tool-smoke to force a
dummy tool call and verify function calling support for the deployment. Use
--retention-smoke to verify a previous_response_id chain. Use --capabilities
to print resolved model/deployment capabilities without sending a live request.
AZURE_OPENAI_BASE_URL is a resource v1 base URL; resource root and /openai
normalize to /openai/v1, and request preview / live smoke use
<normalized_base_url>/responses. Non-standard paths are treated as intentional
proxy base URLs and reported as warnings.
Use --print-request to print the sanitized smoke request JSON without sending
it. Use --require-capability to fail when a resolved local capability is
missing. Use --print-config with --deployment and --catalog-model to print a
config YAML snippet without running diagnostics.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			silenceUsage, err := clidoctor.RunAzureDoctor(cmd.Context(), cmd.OutOrStdout(), clidoctor.AzureOptions{
				CommonOptions:  state.commonOptions(),
				Deployment:     state.deployment,
				RetentionSmoke: state.retentionSmoke,
				PrintConfig:    state.printConfig,
			})
			return applyDoctorResult(cmd, silenceUsage, err)
		},
	}

	cmd.Flags().StringVar(&state.deployment, "deployment", "", "Azure OpenAI deployment name for 'doctor azure'")
	addDoctorCatalogModelFlag(cmd, state, "Catalog model for 'doctor azure' capability/token policy")
	addDoctorSmokeFlag(cmd, state, "Send a live minimal Responses API request for 'doctor azure'")
	addDoctorToolSmokeFlag(cmd, state, "Send a live Azure OpenAI smoke request that forces a dummy tool call")
	addDoctorCapabilitiesFlag(cmd, state, "Print resolved Azure OpenAI deployment capabilities without sending a live request")
	addDoctorRequiredCapabilityFlag(cmd, state)
	cmd.Flags().BoolVar(&state.retentionSmoke, "retention-smoke", false, "Send live Azure OpenAI Responses API requests that verify previous_response_id retention")
	addDoctorTimeoutFlag(cmd, state, "azure", "Timeout for 'doctor azure --smoke'")
	addDoctorJSONFlag(cmd, state, "azure")
	addDoctorPrintRequestFlag(cmd, state, "azure")
	cmd.Flags().BoolVar(&state.printConfig, "print-config", false, "Print Azure OpenAI config YAML for the given deployment/catalog model")

	return cmd
}
