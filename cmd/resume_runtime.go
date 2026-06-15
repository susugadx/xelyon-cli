package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
)

type resumeRuntimeTarget struct {
	sessionID string
	last      bool
}

type interactiveRuntimeSelection struct {
	cfg      *config.Config
	model    string
	provider api.Provider
}

func loadInteractiveConfigSelection(cmd *cobra.Command) *config.Config {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: Failed to load config: %v\n", err)
		cfg = config.DefaultConfig()
	}

	cfg.ApplyEnvironmentOverrides()

	var loopPtr, diffPtr *int
	if flag := cmd.Flags().Lookup("loop-threshold"); flag != nil && flag.Changed {
		loopPtr = &loopThreshold
	}
	if flag := cmd.Flags().Lookup("diff-lines"); flag != nil && flag.Changed {
		diffPtr = &diffLines
	}
	cfg.ApplyFlagOverrides(loopPtr, diffPtr)
	return cfg
}

func loadRuntimeSelectionForMode(cmd *cobra.Command, mode executionMode) (interactiveRuntimeSelection, error) {
	cfg := loadInteractiveConfigSelection(cmd)
	return selectRuntime(cmd, cfg, mode)
}

func loadInteractiveRuntimeSelection(cmd *cobra.Command) (interactiveRuntimeSelection, error) {
	return loadRuntimeSelectionForMode(cmd, executionModeInteractive)
}

func loadResumeRuntimeSelection(cmd *cobra.Command, target resumeRuntimeTarget) (interactiveRuntimeSelection, error) {
	cfg := loadInteractiveConfigSelection(cmd)

	session, err := loadResumeRuntimeSession(target)
	if err != nil {
		if target.last && errors.Is(err, history.ErrNoResumeSessions) {
			return selectInteractiveRuntime(cfg)
		}
		return interactiveRuntimeSelection{}, err
	}

	providerName := strings.TrimSpace(session.ProviderConfigKey)
	if providerName == "" {
		providerName = strings.TrimSpace(session.ProviderName)
	}
	if providerName == "" {
		return selectInteractiveRuntime(cfg)
	}

	model := strings.TrimSpace(session.Model)
	savedModelExplicit := model != ""
	if model == "" {
		model = cfg.GetSelectedModelForProvider(providerName)
	}

	provider, err := resolveInteractiveProvider(providerName)
	if err != nil {
		return interactiveRuntimeSelection{}, err
	}
	if !api.IsProviderSetupRequired(provider) {
		if err := validateSelectedProviderModelWithContext(cfg, provider, model, selectedProviderModelValidationContext{
			explicitModel:     savedModelExplicit,
			providerConfigKey: providerName,
		}); err != nil {
			return interactiveRuntimeSelection{}, err
		}
	}

	return interactiveRuntimeSelection{
		cfg:      cfg,
		model:    model,
		provider: provider,
	}, nil
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
	storage, err := history.NewStorage()
	if err != nil {
		return nil, err
	}

	sessionID := strings.TrimSpace(target.sessionID)
	if target.last {
		sessionID, err = storage.GetLastResumeSession(history.ResumeListOptions{})
		if err != nil {
			return nil, err
		}
	}
	if sessionID == "" {
		return nil, fmt.Errorf("resume session ID is required")
	}

	session, err := storage.Load(sessionID)
	if err != nil {
		return nil, fmt.Errorf("load resume session: %w", err)
	}
	return session, nil
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
