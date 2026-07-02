package cmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func resetRootFlagsForTest() {
	resume = false
	once = false
	interactive = false
	quiet = false
	providerFlag = ""
	modelFlag = ""
	autoApprove = false
	loopThreshold = 0
	diffLines = -1
	outputFormat = "text"
	headless = false
	failOnToolError = false
	readOnly = false
	dryRun = false
	exitCodePolicy = string(agent.HeadlessExitPolicyLegacy)
	noUpdateCheck = false
	imageFlag = ""
	promptFile = ""
	legacyNoTUI = false
	openAISubscriptionAuthDeviceFlag = false
	rootCmd = newRootCommand()
}

func newDoctorSubcommandTest(t *testing.T, newCommand func() *cobra.Command) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	resetRootFlagsForTest()
	t.Cleanup(resetRootFlagsForTest)

	var out bytes.Buffer
	cmd := newCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	return cmd, &out
}

func newRootCommandExecutionTest(t *testing.T) *bytes.Buffer {
	t.Helper()
	resetRootFlagsForTest()
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
		resetRootFlagsForTest()
	})

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	return &out
}

type rootCommandRunners struct {
	runLegacyInteractive           func(string, api.Provider, *config.Config, bool)
	runLegacyInteractiveWithResume func(string, api.Provider, *config.Config, bool)
	runLegacyInteractiveWithImage  func(string, string, api.Provider, string, *config.Config, bool) error
	runTUI                         func(string, api.Provider, *config.Config, bool)
	runTUIWithResume               func(string, api.Provider, *config.Config, bool) error
	runTUIWithResumeDirect         func(string, api.Provider, *config.Config, bool, string) error
	runTUIWithResumePicker         func(string, api.Provider, *config.Config, bool, bool)
	runTUIWithImage                func(string, string, api.Provider, string, *config.Config, bool) error
	runHeadless                    func(context.Context, string, string, api.Provider, *config.Config, agent.HeadlessRunOptions) *agent.HeadlessResult
	runOnce                        func(string, string, api.Provider, *config.Config, bool, bool) error
	runOnceWithImage               func(string, string, api.Provider, string, *config.Config, bool, bool) error
}

func snapshotRootCommandRunners() rootCommandRunners {
	return rootCommandRunners{
		runLegacyInteractive:           runLegacyInteractive,
		runLegacyInteractiveWithResume: runLegacyInteractiveWithResume,
		runLegacyInteractiveWithImage:  runLegacyInteractiveWithImage,
		runTUI:                         runTUI,
		runTUIWithResume:               runTUIWithResume,
		runTUIWithResumeDirect:         runTUIWithResumeDirect,
		runTUIWithResumePicker:         runTUIWithResumePicker,
		runTUIWithImage:                runTUIWithImage,
		runHeadless:                    runHeadless,
		runOnce:                        runOnce,
		runOnceWithImage:               runOnceWithImage,
	}
}

func restoreRootCommandRunners(r rootCommandRunners) {
	runLegacyInteractive = r.runLegacyInteractive
	runLegacyInteractiveWithResume = r.runLegacyInteractiveWithResume
	runLegacyInteractiveWithImage = r.runLegacyInteractiveWithImage
	runTUI = r.runTUI
	runTUIWithResume = r.runTUIWithResume
	runTUIWithResumeDirect = r.runTUIWithResumeDirect
	runTUIWithResumePicker = r.runTUIWithResumePicker
	runTUIWithImage = r.runTUIWithImage
	runHeadless = r.runHeadless
	runOnce = r.runOnce
	runOnceWithImage = r.runOnceWithImage
}

func withRootCommandTest(t *testing.T) {
	t.Helper()

	originalRunners := snapshotRootCommandRunners()
	resetRootFlagsForTest()
	rootCmd.SetArgs(nil)

	t.Cleanup(func() {
		restoreRootCommandRunners(originalRunners)
		resetRootFlagsForTest()
		rootCmd.SetArgs(nil)
	})
}
