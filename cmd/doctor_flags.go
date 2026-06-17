package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/clidoctor"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

type doctorFlagState struct {
	deployment           string
	catalogModel         string
	model                string
	smoke                bool
	toolSmoke            bool
	capabilities         bool
	requiredCapabilities []string
	retentionSmoke       bool
	cacheSmoke           bool
	compactSmoke         bool
	thinkingSmoke        bool
	imageSmoke           bool
	webSearchSmoke       bool
	mcpConnect           bool
	mcpServer            string
	mcpTools             bool
	timeout              time.Duration
	json                 bool
	printConfig          bool
	printRequest         bool
}

func newDoctorFlagState() *doctorFlagState {
	return &doctorFlagState{timeout: clidoctor.DefaultTimeout}
}

func (state *doctorFlagState) commonOptions() clidoctor.CommonOptions {
	return clidoctor.CommonOptions{
		CatalogModel:         state.catalogModel,
		Smoke:                state.smoke,
		ToolSmoke:            state.toolSmoke,
		Capabilities:         state.capabilities,
		RequiredCapabilities: state.requiredCapabilities,
		Timeout:              state.timeout,
		JSON:                 state.json,
		PrintRequest:         state.printRequest,
	}
}

func applyDoctorResult(cmd *cobra.Command, silenceUsage bool, err error) error {
	if silenceUsage {
		cmd.SilenceUsage = true
	}
	return err
}

func addDoctorCatalogModelFlag(cmd *cobra.Command, state *doctorFlagState, usage string) {
	cmd.Flags().StringVar(&state.catalogModel, "catalog-model", "", usage)
}

func addDoctorSmokeFlag(cmd *cobra.Command, state *doctorFlagState, usage string) {
	cmd.Flags().BoolVar(&state.smoke, "smoke", false, usage)
}

func addDoctorToolSmokeFlag(cmd *cobra.Command, state *doctorFlagState, usage string) {
	cmd.Flags().BoolVar(&state.toolSmoke, "tool-smoke", false, usage)
}

func addDoctorCapabilitiesFlag(cmd *cobra.Command, state *doctorFlagState, usage string) {
	cmd.Flags().BoolVar(&state.capabilities, "capabilities", false, usage)
}

func addDoctorRequiredCapabilityFlag(cmd *cobra.Command, state *doctorFlagState) {
	cmd.Flags().StringSliceVar(
		&state.requiredCapabilities,
		"require-capability",
		nil,
		"Fail if a resolved local capability is missing; supported: "+providerdiag.SupportedRequiredCapabilitiesText(),
	)
}

func addDoctorTimeoutFlag(cmd *cobra.Command, state *doctorFlagState, provider, usage string) {
	if usage == "" {
		usage = "Timeout for 'doctor " + provider + "' live smoke requests"
	}
	cmd.Flags().DurationVar(&state.timeout, "timeout", clidoctor.DefaultTimeout, usage)
}

func addDoctorJSONFlag(cmd *cobra.Command, state *doctorFlagState, provider string) {
	cmd.Flags().BoolVar(&state.json, "json", false, "Print 'doctor "+provider+"' diagnostics as JSON")
}

func addDoctorPrintRequestFlag(cmd *cobra.Command, state *doctorFlagState, provider string) {
	cmd.Flags().BoolVar(&state.printRequest, "print-request", false, "Print sanitized 'doctor "+provider+"' smoke request JSON without sending it")
}
