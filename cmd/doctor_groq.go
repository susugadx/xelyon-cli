package cmd

import (
	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/clidoctor"
)

func newGroqDoctorCommand() *cobra.Command {
	state := newDoctorFlagState()
	cmd := &cobra.Command{
		Use:   "groq",
		Short: "Diagnose Groq provider configuration",
		Long: `Diagnose Groq provider configuration.

Checks GROQ_API_KEY, GROQ_API_URL, provider registration, model/catalog model
resolution, Chat Completions route selection, function calling settings, and
token/cost metadata. GROQ_API_URL is an exact Chat Completions endpoint
override; the official Groq path ends with /openai/v1/chat/completions.
OpenAI-compatible /v1/chat/completions proxy paths are allowed but reported as
endpoint warnings. Use --smoke to send a minimal live request. Use --tool-smoke
to force a dummy tool call when function calling is enabled. Use --capabilities
or --require-capability to verify resolved local capabilities without sending a
live request. Use --print-request to print the sanitized smoke request JSON
without sending it.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			silenceUsage, err := clidoctor.RunGroqDoctor(cmd.Context(), cmd.OutOrStdout(), clidoctor.GroqOptions{
				CommonOptions: state.commonOptions(),
				Model:         state.model,
			})
			return applyDoctorResult(cmd, silenceUsage, err)
		},
	}

	cmd.Flags().StringVar(&state.model, "model", "", "Groq model or configured alias for 'doctor groq'")
	addDoctorCatalogModelFlag(cmd, state, "Catalog model for 'doctor groq' token/pricing policy")
	addDoctorSmokeFlag(cmd, state, "Send a live minimal Groq text smoke request")
	addDoctorToolSmokeFlag(cmd, state, "Send a live Groq smoke request that forces a dummy tool call")
	addDoctorCapabilitiesFlag(cmd, state, "Print resolved Groq model capabilities without sending a live request")
	addDoctorRequiredCapabilityFlag(cmd, state)
	addDoctorTimeoutFlag(cmd, state, "groq", "")
	addDoctorJSONFlag(cmd, state, "groq")
	addDoctorPrintRequestFlag(cmd, state, "groq")

	return cmd
}
