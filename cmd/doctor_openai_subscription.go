package cmd

import (
	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/clidoctor"
)

func newOpenAISubscriptionDoctorCommand() *cobra.Command {
	state := newDoctorFlagState()
	cmd := &cobra.Command{
		Use:     "openai-subscription",
		Aliases: []string{"openai_subscription", "chatgpt", "codex-subscription"},
		Short:   "Diagnose OpenAI Subscription provider configuration",
		Long: `Diagnose OpenAI Subscription provider configuration.

Checks local ChatGPT/Codex OAuth auth status, provider registration, model
allowlist, endpoint/originator, billing/cost display, and v2 Responses runtime
policy. The default command is local-only: it does not refresh tokens and does
not send network requests. Use --smoke for a live text request, --tool-smoke
for function_call/function_call_output continuation, --cache-smoke for
prompt_cache_key request-shape verification, --retention-smoke to confirm
full-payload fallback policy, --thinking-smoke for the selected thinking level,
--web-search-smoke for provider-native web_search, and --compact-smoke to
classify optional Compact API support. Use
--print-request to print a sanitized structural request preview without
sending it.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			silenceUsage, err := clidoctor.RunOpenAISubscriptionDoctor(cmd.Context(), out, clidoctor.OpenAISubscriptionOptions{
				CommonOptions:  state.commonOptions(),
				Model:          state.model,
				RetentionSmoke: state.retentionSmoke,
				CacheSmoke:     state.cacheSmoke,
				CompactSmoke:   state.compactSmoke,
				ThinkingSmoke:  state.thinkingSmoke,
				WebSearchSmoke: state.webSearchSmoke,
				SmokeOutput:    out,
			})
			return applyDoctorResult(cmd, silenceUsage, err)
		},
	}

	cmd.Flags().StringVar(&state.model, "model", "", "OpenAI Subscription model for 'doctor openai-subscription'")
	addDoctorCatalogModelFlag(cmd, state, "Catalog model for 'doctor openai-subscription' capability/token policy")
	addDoctorSmokeFlag(cmd, state, "Send a live minimal OpenAI Subscription text smoke request")
	addDoctorToolSmokeFlag(cmd, state, "Send a live OpenAI Subscription tool-loop smoke request")
	addDoctorCapabilitiesFlag(cmd, state, "Print resolved OpenAI Subscription capabilities without sending a live request")
	addDoctorRequiredCapabilityFlag(cmd, state)
	cmd.Flags().BoolVar(&state.retentionSmoke, "retention-smoke", false, "Verify OpenAI Subscription full-payload retention policy")
	cmd.Flags().BoolVar(&state.cacheSmoke, "cache-smoke", false, "Verify OpenAI Subscription prompt_cache_key request shape")
	cmd.Flags().BoolVar(&state.compactSmoke, "compact-smoke", false, "Classify optional OpenAI Subscription Compact API support")
	cmd.Flags().BoolVar(&state.thinkingSmoke, "thinking-smoke", false, "Verify OpenAI Subscription reasoning.effort request shape for the configured thinking level")
	cmd.Flags().BoolVar(&state.webSearchSmoke, "web-search-smoke", false, "Send one live OpenAI Subscription native web_search smoke request")
	addDoctorTimeoutFlag(cmd, state, "openai-subscription", "")
	addDoctorJSONFlag(cmd, state, "openai-subscription")
	addDoctorPrintRequestFlag(cmd, state, "openai-subscription")

	return cmd
}
