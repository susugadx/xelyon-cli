package cmd

import (
	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/clidoctor"
)

func newGeminiDoctorCommand() *cobra.Command {
	state := newDoctorFlagState()
	cmd := &cobra.Command{
		Use:   "gemini",
		Short: "Diagnose Gemini provider configuration",
		Long: `Diagnose Gemini provider configuration.

Checks GEMINI_API_KEY, GEMINI_API_URL, provider registration, model/catalog
model resolution, native Gemini request routes, model-aware function calling,
image input, thinking, context caching, web search, and token/cost metadata. Use
--smoke for minimal text/SSE connectivity, --tool-smoke for the function-calling
runtime path, --image-smoke to send one tiny image request, --web-search-smoke
to verify native Gemini web search, --capabilities or --require-capability to
verify resolved local capabilities without sending a live request, or
--print-request to print sanitized request JSON without sending it.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			silenceUsage, err := clidoctor.RunGeminiDoctor(cmd.Context(), cmd.OutOrStdout(), clidoctor.GeminiOptions{
				CommonOptions:  state.commonOptions(),
				Model:          state.model,
				ImageSmoke:     state.imageSmoke,
				WebSearchSmoke: state.webSearchSmoke,
			})
			return applyDoctorResult(cmd, silenceUsage, err)
		},
	}

	cmd.Flags().StringVar(&state.model, "model", "", "Gemini model or configured alias for 'doctor gemini'")
	addDoctorCatalogModelFlag(cmd, state, "Catalog model for 'doctor gemini' token/pricing/thinking/function-calling policy")
	addDoctorSmokeFlag(cmd, state, "Send a live minimal Gemini text/SSE smoke request")
	addDoctorToolSmokeFlag(cmd, state, "Send a live Gemini function-calling smoke request that forces a dummy tool call")
	cmd.Flags().BoolVar(&state.imageSmoke, "image-smoke", false, "Send one live Gemini image input smoke request")
	cmd.Flags().BoolVar(&state.webSearchSmoke, "web-search-smoke", false, "Send one live Gemini native web search smoke request")
	addDoctorCapabilitiesFlag(cmd, state, "Print resolved Gemini model capabilities without sending a live request")
	addDoctorRequiredCapabilityFlag(cmd, state)
	addDoctorTimeoutFlag(cmd, state, "gemini", "Per-request timeout for 'doctor gemini' live smoke requests")
	addDoctorJSONFlag(cmd, state, "gemini")
	addDoctorPrintRequestFlag(cmd, state, "gemini")

	return cmd
}
