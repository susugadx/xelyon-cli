package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/cliruntime"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
)

var errResumeRuntimeStorageUnavailable = cliruntime.ErrResumeRuntimeStorageUnavailable

type resumeRuntimeTarget struct {
	sessionID string
	last      bool
}

type interactiveRuntimeSelection struct {
	cfg      *config.Config
	model    string
	provider api.Provider
}

type resumeRuntimeSelection struct {
	interactiveRuntimeSelection
	preloadLastSession bool
}

func loadInteractiveConfigSelection(cmd *cobra.Command) *config.Config {
	return loadConfigSelection(cmd, false)
}

func loadConfigSelection(cmd *cobra.Command, readOnly bool) *config.Config {
	overrides := cliruntime.ConfigOverrides{}
	if flag := cmd.Flags().Lookup("loop-threshold"); flag != nil && flag.Changed {
		overrides.LoopThresholdChanged = true
		overrides.LoopThreshold = loopThreshold
	}
	if flag := cmd.Flags().Lookup("diff-lines"); flag != nil && flag.Changed {
		overrides.DiffLinesChanged = true
		overrides.DiffLines = diffLines
	}
	if readOnly {
		return cliruntime.LoadConfigSelectionReadOnly(cmd.ErrOrStderr(), overrides)
	}
	return cliruntime.LoadConfigSelection(cmd.ErrOrStderr(), overrides)
}

func loadRuntimeSelectionForMode(cmd *cobra.Command, mode executionMode) (interactiveRuntimeSelection, error) {
	cfg := loadConfigSelection(cmd, mode == executionModeHeadless && (readOnly || dryRun))
	return selectRuntime(cmd, cfg, mode)
}

func loadInteractiveRuntimeSelection(cmd *cobra.Command) (interactiveRuntimeSelection, error) {
	return loadRuntimeSelectionForMode(cmd, executionModeInteractive)
}

func loadResumeRuntimeSelection(cmd *cobra.Command, target resumeRuntimeTarget) (resumeRuntimeSelection, error) {
	cfg := loadInteractiveConfigSelection(cmd)

	session, err := loadResumeRuntimeSession(target)
	if err != nil {
		if target.last && errors.Is(err, history.ErrNoResumeSessions) {
			runtime, err := selectInteractiveRuntime(cfg)
			return resumeRuntimeSelection{
				interactiveRuntimeSelection: runtime,
				preloadLastSession:          true,
			}, err
		}
		if target.last && errors.Is(err, errResumeRuntimeStorageUnavailable) {
			runtime, err := selectInteractiveRuntime(cfg)
			return resumeRuntimeSelection{
				interactiveRuntimeSelection: runtime,
				preloadLastSession:          false,
			}, err
		}
		return resumeRuntimeSelection{}, err
	}

	providerName := strings.TrimSpace(session.ProviderConfigKey)
	if providerName == "" {
		providerName = strings.TrimSpace(session.ProviderName)
	}
	if providerName == "" {
		runtime, err := selectInteractiveRuntime(cfg)
		return resumeRuntimeSelection{
			interactiveRuntimeSelection: runtime,
			preloadLastSession:          target.last,
		}, err
	}

	model := strings.TrimSpace(session.Model)
	savedModelExplicit := model != ""
	if model == "" {
		model = cfg.GetSelectedModelForProvider(providerName)
	}

	provider, err := resolveInteractiveProvider(providerName)
	if err != nil {
		return resumeRuntimeSelection{}, err
	}
	if !api.IsProviderSetupRequired(provider) {
		if err := validateSelectedProviderModelWithContext(cfg, provider, model, selectedProviderModelValidationContext{
			explicitModel:     savedModelExplicit,
			providerConfigKey: providerName,
		}); err != nil {
			return resumeRuntimeSelection{}, err
		}
	}

	return resumeRuntimeSelection{
		interactiveRuntimeSelection: interactiveRuntimeSelection{
			cfg:      cfg,
			model:    model,
			provider: provider,
		},
		preloadLastSession: target.last,
	}, nil
}

func runTUIForResumeRuntime(selection resumeRuntimeSelection, autoApprove bool) error {
	if selection.preloadLastSession {
		return runTUIWithResume(selection.model, selection.provider, selection.cfg, autoApprove)
	}
	runTUI(selection.model, selection.provider, selection.cfg, autoApprove)
	return nil
}

func selectInteractiveRuntime(cfg *config.Config) (interactiveRuntimeSelection, error) {
	return selectRuntime(nil, cfg, executionModeInteractive)
}

func selectRuntime(cmd *cobra.Command, cfg *config.Config, mode executionMode) (interactiveRuntimeSelection, error) {
	model := getModel(cfg)
	providerName := resolveProviderName(providerFlag, cfg.DefaultProvider)
	provider, err := resolveProviderForExecutionMode(cmd, providerName, mode, model)
	if err != nil {
		return interactiveRuntimeSelection{}, err
	}
	if !api.IsProviderSetupRequired(provider) {
		if err := validateSelectedProviderModel(cfg, provider, model); err != nil {
			if mode == executionModeHeadless {
				return interactiveRuntimeSelection{}, newHeadlessRuntimeSelectionConfigError(provider.Name(), model, err)
			}
			return interactiveRuntimeSelection{}, err
		}
	}
	return interactiveRuntimeSelection{
		cfg:      cfg,
		model:    model,
		provider: provider,
	}, nil
}

func loadResumePickerRuntimeSelection(cmd *cobra.Command) interactiveRuntimeSelection {
	runtime, err := loadInteractiveRuntimeSelection(cmd)
	if err == nil {
		return runtime
	}

	cfg := loadInteractiveConfigSelection(cmd)
	return interactiveRuntimeSelection{
		cfg:      cfg,
		model:    "resume-bootstrap",
		provider: resumeBootstrapProvider{reason: err},
	}
}

func loadResumeRuntimeSession(target resumeRuntimeTarget) (*history.Session, error) {
	return cliruntime.LoadResumeSession(cliruntime.ResumeTarget{
		SessionID: target.sessionID,
		Last:      target.last,
	})
}

type resumeBootstrapProvider struct {
	reason error
}

func (p resumeBootstrapProvider) Name() string {
	return "resume_bootstrap"
}

func (p resumeBootstrapProvider) SupportsImages() bool {
	return false
}

func (p resumeBootstrapProvider) IsFunctionCallingEnabled() bool {
	return false
}

func (p resumeBootstrapProvider) ChatWithTools(context.Context, string, []api.Message, string) (string, error) {
	return "", p.chatError()
}

func (p resumeBootstrapProvider) ChatWithImage(context.Context, string, []api.Message, string, *api.ImageData, string) (string, error) {
	return "", p.chatError()
}

func (p resumeBootstrapProvider) chatError() error {
	if p.reason == nil {
		return fmt.Errorf("resume a session or switch provider before chatting")
	}
	return fmt.Errorf("resume a session or switch provider before chatting: %w", p.reason)
}
