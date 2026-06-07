package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/crypto"
	"github.com/susugadx/xelyon-cli/internal/providerhistory"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	"github.com/susugadx/xelyon-cli/internal/taskstate"
)

type providerHistoryProjectionResult struct {
	History []api.Message
	Report  ProviderHistoryProjectionReport
}

func (a *Agent) providerFacingHistory() []api.Message {
	if a == nil {
		return nil
	}
	return a.providerFacingHistoryFromRaw(a.cloneRawHistoryForProviderProjection())
}

func (a *Agent) providerFacingHistoryExcludingLatestMessage() []api.Message {
	if a == nil {
		return nil
	}
	raw := a.cloneRawHistoryForProviderProjection()
	if len(raw) > 0 {
		raw = raw[:len(raw)-1]
	}
	return a.providerFacingHistoryFromRaw(raw)
}

func (a *Agent) providerFacingHistoryFromRaw(raw []api.Message) []api.Message {
	result := a.providerFacingHistoryProjectionFromRaw(raw)
	a.recordLastProviderHistoryProjectionReport(result.Report)
	return result.History
}

func (a *Agent) providerFacingHistoryProjectionFromRaw(raw []api.Message) providerHistoryProjectionResult {
	return a.buildProviderHistoryProjectionFromRaw(providerHistoryReductionPolicyForRuntime(a.Runtime), raw)
}

func (a *Agent) tokenBudgetHistory() []api.Message {
	if a == nil {
		return nil
	}
	result, _ := a.providerHistoryProjectionForTokenBudget(context.Background(), a.cloneRawHistoryForProviderProjection())
	return result.History
}

func (a *Agent) buildProviderHistoryProjection(policy ProviderHistoryReductionPolicy) providerHistoryProjectionResult {
	if a == nil {
		result := providerhistory.Project(providerhistory.ProjectionInput{Policy: normalizeProviderHistoryReductionPolicy(policy)})
		return providerHistoryProjectionResult{History: result.History, Report: result.Report}
	}
	return a.buildProviderHistoryProjectionFromRaw(policy, a.cloneRawHistoryForProviderProjection())
}

func (a *Agent) cloneRawHistoryForProviderProjection() []api.Message {
	if a == nil {
		return nil
	}
	a.historyMu.Lock()
	raw := api.CloneMessages(a.History)
	a.historyMu.Unlock()
	return raw
}

func (a *Agent) buildProviderHistoryProjectionFromRaw(policy ProviderHistoryReductionPolicy, raw []api.Message) providerHistoryProjectionResult {
	return a.buildProviderHistoryProjectionFromRawResolvedPolicy(a.providerHistoryProjectionPolicy(policy), raw)
}

func (a *Agent) buildProviderHistoryProjectionFromRawResolvedPolicy(policy ProviderHistoryReductionPolicy, raw []api.Message) providerHistoryProjectionResult {
	result := providerhistory.Project(providerhistory.ProjectionInput{
		Messages: raw,
		Policy:   policy,
	})
	return providerHistoryProjectionResult{
		History: result.History,
		Report:  result.Report,
	}
}

func (a *Agent) buildProviderHistoryProjectionFromRawWithRawOutputApplyDisabledReason(raw []api.Message, reason string) providerHistoryProjectionResult {
	policy := a.providerHistoryProjectionPolicy(providerHistoryReductionPolicyForRuntime(a.Runtime))
	policy.RawOutputArtifactsMode = providerhistory.RawOutputArtifactsDryRun
	policy.RawOutputApplyDisabledReason = reason
	return a.buildProviderHistoryProjectionFromRawResolvedPolicy(policy, raw)
}

func (a *Agent) buildProviderHistoryProjectionFromRawForTokenBudget(raw []api.Message) providerHistoryProjectionResult {
	policy := providerHistoryReductionPolicyForRuntime(a.Runtime)
	policy.SideEffects = providerhistory.ProjectionSideEffectsReadOnly
	policy = a.providerHistoryProjectionPolicy(policy)
	policy.RawOutputApplyDisabledReason = "provider_history_projection_read_only"
	return a.buildProviderHistoryProjectionFromRawResolvedPolicy(policy, raw)
}

func (a *Agent) providerHistoryProjectionPolicy(policy ProviderHistoryReductionPolicy) ProviderHistoryReductionPolicy {
	policy = normalizeProviderHistoryReductionPolicy(policy)
	if a == nil || a.Runtime == nil {
		return policy
	}

	policy.RawOutputArtifactsMode = providerHistoryRawOutputArtifactsModeForRuntime(a.Runtime)
	policy.RawOutputRehydrateContextEnabled = a.Runtime.Options.EnableProviderHistoryRehydrateContext
	if policy.RawOutputArtifactsMode != providerhistory.RawOutputArtifactsDisabled &&
		policy.SideEffects != providerhistory.ProjectionSideEffectsReadOnly {
		policy.SessionID = a.providerHistoryRawOutputArtifactSessionID()
		policy.RawOutputArtifactStore = a.providerHistoryRawOutputArtifactStore()
	}
	if a.Runtime.Options.EnableProviderHistoryRehydrateContext {
		policy.ActiveContextTransportAvailable = a.providerActiveContextTransport() != api.ActiveContextTransportNone
	}
	if policy.Mode == ProviderHistoryReductionApply {
		policy.EvidencePointers = a.providerHistoryReductionEvidencePointers()
		if a.Runtime.Options.EnableProviderHistoryRehydrateContext {
			policy.EvidenceReductionRequiresActiveContext = true
		}
	}
	return policy
}

func providerHistoryReductionPolicyForRuntime(runtime *AgentRuntime) ProviderHistoryReductionPolicy {
	return ProviderHistoryReductionPolicy{
		Mode:                             providerHistoryReductionModeResolutionForRuntime(runtime).effective,
		RawOutputArtifactsMode:           providerHistoryRawOutputArtifactsModeForRuntime(runtime),
		RawOutputRehydrateContextEnabled: providerHistoryRawOutputRehydrateContextEnabledForRuntime(runtime),
	}
}

func providerHistoryRawOutputArtifactsModeForRuntime(runtime *AgentRuntime) providerhistory.RawOutputArtifactsMode {
	if runtime == nil {
		return providerhistory.RawOutputArtifactsDisabled
	}
	switch runtime.Options.ProviderHistoryRawOutputArtifacts.Mode {
	case config.ProviderHistoryRawOutputArtifactsModeApply:
		return providerhistory.RawOutputArtifactsApply
	case config.ProviderHistoryRawOutputArtifactsModeDryRun:
		return providerhistory.RawOutputArtifactsDryRun
	default:
		return providerhistory.RawOutputArtifactsDisabled
	}
}

func providerHistoryRawOutputRehydrateContextEnabledForRuntime(runtime *AgentRuntime) bool {
	return runtime != nil && runtime.Options.EnableProviderHistoryRehydrateContext
}

func (a *Agent) providerHistoryRawOutputArtifactSessionID() string {
	if a == nil || a.session == nil {
		return ""
	}
	return a.session.ID
}

func (a *Agent) providerHistoryRawOutputArtifactStore() providerhistory.RawOutputArtifactStore {
	if a == nil || a.Runtime == nil {
		return nil
	}
	if a.Runtime.RawOutputArtifactStore != nil {
		return a.Runtime.RawOutputArtifactStore
	}
	store, err := openProviderHistoryRawOutputArtifactStore(a.Runtime)
	if err != nil {
		return nil
	}
	a.Runtime.RawOutputArtifactStore = store
	return store
}

func openProviderHistoryRawOutputArtifactStore(runtime *AgentRuntime) (providerhistory.RawOutputArtifactStore, error) {
	if runtime == nil {
		return nil, nil
	}
	root, err := resolveProviderHistoryRawOutputArtifactRoot(runtime)
	if err != nil {
		return nil, err
	}
	opts := providerHistoryRawOutputStoreOptions(runtime.Options.ProviderHistoryRawOutputArtifacts)
	if os.Getenv("XELYON_ENCRYPT_HISTORY") == "1" {
		passphrase, err := crypto.GetOrCreatePassphrase()
		if err != nil {
			return nil, err
		}
		opts.EncryptionEnabled = true
		opts.Passphrase = passphrase
	}
	return rawoutputs.OpenStore(rawoutputs.Root(root.Root), opts)
}

type providerHistoryRawOutputArtifactRootResolution struct {
	Root   string
	Source string
}

func resolveProviderHistoryRawOutputArtifactRoot(runtime *AgentRuntime) (providerHistoryRawOutputArtifactRootResolution, error) {
	if rawRoot, ok := os.LookupEnv(config.ProviderHistoryRawOutputArtifactRootEnvVar); ok && strings.TrimSpace(rawRoot) != "" {
		return providerHistoryRawOutputArtifactRootResolution{
			Root:   strings.TrimSpace(rawRoot),
			Source: "env:" + config.ProviderHistoryRawOutputArtifactRootEnvVar,
		}, nil
	}
	if runtime != nil {
		if root := strings.TrimSpace(runtime.Options.ProviderHistoryRawOutputArtifacts.Root); root != "" {
			return providerHistoryRawOutputArtifactRootResolution{Root: root, Source: "config"}, nil
		}
		if root := strings.TrimSpace(runtime.RawOutputArtifactRoot); root != "" {
			return providerHistoryRawOutputArtifactRootResolution{Root: root, Source: "runtime"}, nil
		}
	}
	root, err := defaultProviderHistoryRawOutputArtifactRoot()
	if err != nil {
		return providerHistoryRawOutputArtifactRootResolution{}, err
	}
	return providerHistoryRawOutputArtifactRootResolution{Root: root, Source: "default"}, nil
}

func defaultProviderHistoryRawOutputArtifactRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".xelyon", "history", "rawoutputs"), nil
}

func providerHistoryRawOutputStoreOptions(cfg config.ProviderHistoryRawOutputArtifactsConfig) rawoutputs.StoreOptions {
	return rawoutputs.StoreOptions{
		MaxArtifactBytes:  int64(cfg.MaxArtifactBytes),
		SessionQuotaBytes: int64(cfg.SessionQuotaBytes),
		ChunkBytes:        cfg.ChunkBytes,
	}
}

func (a *Agent) recordLastProviderHistoryProjectionReport(report ProviderHistoryProjectionReport) {
	if a == nil || a.Runtime == nil {
		return
	}
	a.Runtime.LastProviderHistoryProjectionReport = cloneProviderHistoryProjectionReport(report)
}

func cloneProviderHistoryProjectionReport(report ProviderHistoryProjectionReport) ProviderHistoryProjectionReport {
	return providerhistory.CloneProjectionReport(report)
}

func (a *Agent) providerHistoryReductionEvidencePointers() []taskstate.EvidencePointer {
	if a == nil || a.Runtime == nil || a.Runtime.TaskLedger == nil {
		return nil
	}
	return taskstate.EvidencePointersFromState(a.Runtime.TaskLedger.Snapshot())
}
