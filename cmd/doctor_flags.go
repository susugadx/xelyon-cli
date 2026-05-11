package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

const defaultDoctorTimeout = 120 * time.Second

func addDoctorCatalogModelFlag(cmd *cobra.Command, usage string) {
	cmd.Flags().StringVar(&doctorCatalogModelFlag, "catalog-model", "", usage)
}

func addDoctorSmokeFlag(cmd *cobra.Command, usage string) {
	cmd.Flags().BoolVar(&doctorSmokeFlag, "smoke", false, usage)
}

func addDoctorToolSmokeFlag(cmd *cobra.Command, usage string) {
	cmd.Flags().BoolVar(&doctorToolSmokeFlag, "tool-smoke", false, usage)
}

func addDoctorCapabilitiesFlag(cmd *cobra.Command, usage string) {
	cmd.Flags().BoolVar(&doctorCapabilitiesFlag, "capabilities", false, usage)
}

func addDoctorRequiredCapabilityFlag(cmd *cobra.Command) {
	cmd.Flags().StringSliceVar(
		&doctorRequiredCapabilityFlags,
		"require-capability",
		nil,
		"Fail if a resolved local capability is missing; supported: "+providerdiag.SupportedRequiredCapabilitiesText(),
	)
}

func addDoctorTimeoutFlag(cmd *cobra.Command, provider, usage string) {
	if usage == "" {
		usage = "Timeout for 'doctor " + provider + "' live smoke requests"
	}
	cmd.Flags().DurationVar(&doctorTimeoutFlag, "timeout", defaultDoctorTimeout, usage)
}

func addDoctorJSONFlag(cmd *cobra.Command, provider string) {
	cmd.Flags().BoolVar(&doctorJSONFlag, "json", false, "Print 'doctor "+provider+"' diagnostics as JSON")
}

func addDoctorPrintRequestFlag(cmd *cobra.Command, provider string) {
	cmd.Flags().BoolVar(&doctorPrintRequestFlag, "print-request", false, "Print sanitized 'doctor "+provider+"' smoke request JSON without sending it")
}
