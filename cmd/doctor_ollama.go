package cmd

import (
	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/clidoctor"
)

func newOllamaDoctorCommand() *cobra.Command {
	state := newDoctorFlagState()
	cmd := &cobra.Command{
		Use:   "ollama",
		Short: "Diagnose Ollama provider configuration",
		Long: `Diagnose Ollama provider configuration.

Checks OLLAMA_BASE_URL, provider registration, model/catalog model resolution,
installed local model availability, Ollama /api/chat route selection, function
calling settings, and token/cost metadata. Use --smoke to send a minimal live
local request. Use --tool-smoke to force a dummy tool call when function
calling is enabled. Use --capabilities or --require-capability to verify
resolved local capabilities without sending a generation request. Requiring
local_model_available performs /api/tags discovery. Use --print-request to
print the sanitized smoke request JSON without sending it.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			silenceUsage, err := clidoctor.RunOllamaDoctor(cmd.Context(), cmd.OutOrStdout(), clidoctor.OllamaOptions{
				CommonOptions: state.commonOptions(),
				Model:         state.model,
			})
			return applyDoctorResult(cmd, silenceUsage, err)
		},
	}

	cmd.Flags().StringVar(&state.model, "model", "", "Ollama model or configured alias for 'doctor ollama'")
	addDoctorCatalogModelFlag(cmd, state, "Catalog model for 'doctor ollama' token/pricing policy")
	addDoctorSmokeFlag(cmd, state, "Send a live minimal Ollama text smoke request")
	addDoctorToolSmokeFlag(cmd, state, "Send a live Ollama smoke request that forces a dummy tool call")
	addDoctorCapabilitiesFlag(cmd, state, "Print resolved Ollama model capabilities without sending a generation request")
	addDoctorRequiredCapabilityFlag(cmd, state)
	addDoctorTimeoutFlag(cmd, state, "ollama", "")
	addDoctorJSONFlag(cmd, state, "ollama")
	addDoctorPrintRequestFlag(cmd, state, "ollama")

	return cmd
}
