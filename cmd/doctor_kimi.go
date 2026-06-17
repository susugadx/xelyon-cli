package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/clidoctor"
)

const defaultKimiDoctorModel = "kimi-k2.6"

func newKimiDoctorCommand() *cobra.Command {
	state := newDoctorFlagState()
	cmd := &cobra.Command{
		Use:   "kimi",
		Short: "Diagnose Kimi native provider configuration",
		Long: `Diagnose Kimi native provider configuration.

Checks MOONSHOT_API_KEY, KIMI_API_URL, provider registration, model/catalog
config, Chat Completions route, image capability, unsupported native features,
and prompt_cache_key request shape. KIMI_API_URL is an exact Chat Completions
endpoint override; the official path ends with /v1/chat/completions, and other
paths are treated as intentional proxy endpoints. Use --smoke to send live Kimi
Chat Completions requests, --image-smoke to send one tiny image request,
--tool-smoke to include a dummy tool call, --web-search-smoke to verify the
built-in $web_search route, --capabilities or --require-capability to verify
resolved local capabilities without sending a live request, or --print-request
to preview sanitized request JSON without sending it.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			silenceUsage, err := clidoctor.RunKimiDoctor(cmd.Context(), cmd.OutOrStdout(), clidoctor.KimiOptions{
				CommonOptions:  state.commonOptions(),
				Model:          state.model,
				ModelChanged:   cmd.Flags().Changed("model"),
				ImageSmoke:     state.imageSmoke,
				WebSearchSmoke: state.webSearchSmoke,
			})
			return applyDoctorResult(cmd, silenceUsage, err)
		},
	}

	cmd.Flags().StringVar(&state.model, "model", "", fmt.Sprintf("Kimi model for 'doctor kimi' (default: config/XELYON_MODEL, fallback %s)", defaultKimiDoctorModel))
	addDoctorCatalogModelFlag(cmd, state, "Catalog model for 'doctor kimi' token/pricing policy")
	addDoctorSmokeFlag(cmd, state, "Send live minimal Kimi Chat Completions smoke requests")
	cmd.Flags().BoolVar(&state.imageSmoke, "image-smoke", false, "Send one live Kimi image input smoke request")
	addDoctorToolSmokeFlag(cmd, state, "Send a live Kimi smoke request that forces a dummy tool call")
	cmd.Flags().BoolVar(&state.webSearchSmoke, "web-search-smoke", false, "Send a live Kimi built-in $web_search smoke request")
	addDoctorCapabilitiesFlag(cmd, state, "Print resolved Kimi model capabilities without sending a live request")
	addDoctorRequiredCapabilityFlag(cmd, state)
	addDoctorTimeoutFlag(cmd, state, "kimi", "")
	addDoctorJSONFlag(cmd, state, "kimi")
	addDoctorPrintRequestFlag(cmd, state, "kimi")

	return cmd
}
