package cmd

import (
	"github.com/susugadx/xelyon-cli/internal/app"

	// プロバイダーの init() を実行するための副作用インポート
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/all"
)

var (
	resume          bool
	once            bool
	interactive     bool
	quiet           bool
	providerFlag    string
	modelFlag       string
	autoApprove     bool
	loopThreshold   int
	diffLines       int
	outputFormat    string
	headless        bool
	failOnToolError bool
	readOnly        bool
	dryRun          bool
	exitCodePolicy  string
	noUpdateCheck   bool
	imageFlag       string
	promptFile      string
	legacyNoTUI     bool

	runLegacyInteractive           = app.RunLegacyInteractiveWithConfig
	runLegacyInteractiveWithResume = app.RunLegacyInteractiveWithResumeWithConfig
	runLegacyInteractiveWithImage  = app.RunLegacyInteractiveWithImageWithConfig
	runTUI                         = app.RunTUIWithConfig
	runTUIWithResume               = app.RunTUIWithResumeWithConfig
	runTUIWithResumeDirect         = app.RunTUIWithResumeSessionWithConfig
	runTUIWithResumePicker         = app.RunTUIWithResumePickerWithConfig
	runTUIWithImage                = app.RunTUIWithImageWithConfig
	runHeadless                    = app.RunHeadlessWithConfigOptions
	runOnce                        = app.RunOnceWithConfig
	runOnceWithImage               = app.RunOnceWithImageWithConfig
)

var rootCmd = newRootCommand()
