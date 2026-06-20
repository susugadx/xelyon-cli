package agent

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/crypto"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
)

const rawOutputsCommandRefLimit = 20

type rawOutputsCommandAction string

const (
	rawOutputsCommandSummary  rawOutputsCommandAction = "summary"
	rawOutputsCommandVerify   rawOutputsCommandAction = "verify"
	rawOutputsCommandRefs     rawOutputsCommandAction = "refs"
	rawOutputsCommandGCDryRun rawOutputsCommandAction = "gc_dry_run"
)

// handleRawOutputsCommand は raw output artifact store の read-only diagnostics を表示する。
func handleRawOutputsCommand(agent *Agent, args []string) bool {
	out := agent.output()
	action, err := parseRawOutputsCommandAction(args)
	if err != nil {
		yellow.Fprintf(out, "%v\n", err)
		printRawOutputsUsage(out)
		return true
	}
	sessionID := rawOutputsCommandSessionID(agent)
	if sessionID == "" {
		yellow.Fprintln(out, "No active session; /rawoutputs diagnostics require a session.")
		return true
	}
	store, root, err := openProviderHistoryRawOutputArtifactStoreReadOnly(agentRuntimeForRawOutputsCommand(agent))
	if err != nil {
		red.Fprintf(out, "Failed to open raw output artifact store: %v\n", err)
		return true
	}
	liveRefs := providerHistoryRawOutputLiveRefsForAgent(agent, sessionID)
	req := rawoutputs.DiagnosticsRequest{
		SessionID:     sessionID,
		LiveRefs:      liveRefs,
		IncludeVerify: action == rawOutputsCommandVerify,
		IncludeRefs:   action == rawOutputsCommandRefs,
		RefLimit:      rawOutputsCommandRefLimit,
	}
	if action == rawOutputsCommandGCDryRun {
		req.IncludeGCDryRun = true
	}
	diagnostics, err := store.Diagnostics(context.Background(), req)
	if err != nil {
		red.Fprintf(out, "Failed to inspect raw output artifact store: %v\n", err)
		return true
	}
	renderRawOutputsDiagnostics(out, rawOutputsCommandRuntimeConfig(agentRuntimeForRawOutputsCommand(agent)), root, action, diagnostics)
	return true
}

func parseRawOutputsCommandAction(args []string) (rawOutputsCommandAction, error) {
	if len(args) == 0 {
		return rawOutputsCommandSummary, nil
	}
	switch args[0] {
	case "summary":
		if len(args) == 1 {
			return rawOutputsCommandSummary, nil
		}
	case "verify":
		if len(args) == 1 {
			return rawOutputsCommandVerify, nil
		}
	case "refs":
		if len(args) == 1 {
			return rawOutputsCommandRefs, nil
		}
	case "gc":
		if len(args) == 2 && args[1] == "--dry-run" {
			return rawOutputsCommandGCDryRun, nil
		}
		if len(args) == 2 && args[1] == "--apply" {
			return "", fmt.Errorf("/rawoutputs gc --apply is not supported; use /rawoutputs gc --dry-run")
		}
	case "delete", "repair":
		return "", fmt.Errorf("/rawoutputs %s is not supported; this command is read-only", args[0])
	}
	return "", fmt.Errorf("unknown /rawoutputs arguments: %s", strings.Join(args, " "))
}

func printRawOutputsUsage(out io.Writer) {
	_, _ = fmt.Fprintln(out, "Usage: /rawoutputs [summary|verify|refs|gc --dry-run]")
}

func agentRuntimeForRawOutputsCommand(agent *Agent) *AgentRuntime {
	if agent == nil {
		return nil
	}
	return agent.Runtime
}

func rawOutputsCommandSessionID(agent *Agent) string {
	if agent == nil || agent.session == nil {
		return ""
	}
	return strings.TrimSpace(agent.session.ID)
}

func rawOutputsCommandRuntimeConfig(runtime *AgentRuntime) config.ProviderHistoryRawOutputArtifactsConfig {
	if runtime == nil {
		return config.DefaultProviderHistoryRawOutputArtifactsConfig()
	}
	cfg := config.NormalizeProviderHistoryRawOutputArtifactsConfig(runtime.Options.ProviderHistoryRawOutputArtifacts)
	mode, _, err := config.NormalizeProviderHistoryRawOutputArtifactsMode(cfg.Mode)
	if err == nil {
		cfg.Mode = mode
	}
	return cfg
}

func openProviderHistoryRawOutputArtifactStoreReadOnly(runtime *AgentRuntime) (*rawoutputs.Store, providerHistoryRawOutputArtifactRootResolution, error) {
	root, err := resolveProviderHistoryRawOutputArtifactRoot(runtime)
	if err != nil {
		return nil, providerHistoryRawOutputArtifactRootResolution{}, err
	}
	cfg := rawOutputsCommandRuntimeConfig(runtime)
	opts := providerHistoryRawOutputStoreOptions(cfg)
	if os.Getenv("XELYON_ENCRYPT_HISTORY") == "1" {
		if passphrase, err := crypto.GetExistingPassphrase(); err == nil {
			opts.EncryptionEnabled = true
			opts.Passphrase = passphrase
		}
	}
	store, err := rawoutputs.OpenStoreReadOnly(rawoutputs.Root(root.Root), opts)
	return store, root, err
}

func providerHistoryRawOutputLiveRefsForAgent(agent *Agent, sessionID string) []rawoutputs.RawOutputRef {
	if agent == nil || agent.Runtime == nil {
		return nil
	}
	refs := append([]rawoutputs.RawOutputRef(nil), agent.Runtime.LastProviderHistoryProjectionReport.RawOutputRefs...)
	for _, ledger := range agent.Runtime.LastReviewPromptReductionReport.RawOutputLedgers {
		refs = appendReviewRawOutputLedgerRefs(refs, sessionID, ledger.RequiredRefs)
		refs = appendReviewRawOutputLedgerRefs(refs, sessionID, ledger.OptionalRefs)
		refs = appendReviewRawOutputLedgerRefs(refs, sessionID, ledger.RehydratedRefs)
		refs = appendReviewRawOutputLedgerRefs(refs, sessionID, ledger.MetadataOnlyRefs)
	}
	return dedupeRawOutputLiveRefs(refs, sessionID)
}

func appendReviewRawOutputLedgerRefs(out []rawoutputs.RawOutputRef, sessionID string, refs []reviewpromptreduction.ReviewProbeRawOutputLedgerRef) []rawoutputs.RawOutputRef {
	for _, ref := range refs {
		refID := strings.TrimSpace(ref.RefID)
		if refID == "" {
			continue
		}
		out = append(out, rawoutputs.RawOutputRef{
			RefID:       refID,
			SessionID:   sessionID,
			ContentHash: ref.ContentHash,
			ByteSize:    ref.ByteSize,
		})
	}
	return out
}

func dedupeRawOutputLiveRefs(refs []rawoutputs.RawOutputRef, sessionID string) []rawoutputs.RawOutputRef {
	byKey := map[string]rawoutputs.RawOutputRef{}
	for _, ref := range refs {
		ref.SessionID = strings.TrimSpace(ref.SessionID)
		if ref.SessionID == "" {
			ref.SessionID = sessionID
		}
		ref.RefID = strings.TrimSpace(ref.RefID)
		if ref.SessionID != sessionID || ref.RefID == "" {
			continue
		}
		key := ref.SessionID + "\x00" + ref.RefID
		if _, ok := byKey[key]; !ok {
			byKey[key] = ref
		}
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]rawoutputs.RawOutputRef, 0, len(keys))
	for _, key := range keys {
		out = append(out, byKey[key])
	}
	return out
}

func renderRawOutputsDiagnostics(out io.Writer, cfg config.ProviderHistoryRawOutputArtifactsConfig, root providerHistoryRawOutputArtifactRootResolution, action rawOutputsCommandAction, diagnostics rawoutputs.DiagnosticsResult) {
	_, _ = fmt.Fprintln(out, "Raw output artifacts")
	_, _ = fmt.Fprintf(out, "  mode: %s\n", cfg.Mode)
	_, _ = fmt.Fprintf(out, "  root: %s\n", root.Root)
	_, _ = fmt.Fprintf(out, "  root_source: %s\n", root.Source)
	_, _ = fmt.Fprintf(out, "  symlink_policy: reject_existing_parent_symlinks\n")
	_, _ = fmt.Fprintf(out, "  session: %s\n", diagnostics.SessionID)
	_, _ = fmt.Fprintf(out, "  store_exists: %t\n", diagnostics.StoreExists)
	_, _ = fmt.Fprintf(out, "  refs: %d\n", diagnostics.RefCount)
	_, _ = fmt.Fprintf(out, "  artifacts: %d\n", diagnostics.ArtifactCount)
	_, _ = fmt.Fprintf(out, "  bytes: %d\n", diagnostics.ByteSize)
	_, _ = fmt.Fprintf(out, "  live_ref_sources: %d\n", diagnostics.LiveRefSourceCount)
	_, _ = fmt.Fprintf(out, "  retention: %s\n", cfg.Retention)
	if action == rawOutputsCommandVerify {
		renderRawOutputsVerifyDiagnostics(out, diagnostics)
	}
	if action == rawOutputsCommandRefs {
		renderRawOutputsRefDiagnostics(out, diagnostics.Refs)
	}
	if action == rawOutputsCommandGCDryRun {
		renderRawOutputsGCDryRun(out, diagnostics)
	}
}

func renderRawOutputsVerifyDiagnostics(out io.Writer, diagnostics rawoutputs.DiagnosticsResult) {
	_, _ = fmt.Fprintln(out, "  verify:")
	_, _ = fmt.Fprintf(out, "    missing_objects: %d\n", diagnostics.MissingObjects)
	_, _ = fmt.Fprintf(out, "    hash_mismatches: %d\n", diagnostics.HashMismatches)
	_, _ = fmt.Fprintf(out, "    decrypt_failures: %d\n", diagnostics.DecryptFailures)
	_, _ = fmt.Fprintf(out, "    path_failures: %d\n", diagnostics.PathFailures)
	_, _ = fmt.Fprintf(out, "    quarantined_refs: %d\n", diagnostics.QuarantinedRefs)
	_, _ = fmt.Fprintf(out, "    tombstoned_refs: %d\n", diagnostics.TombstonedRefs)
	_, _ = fmt.Fprintf(out, "    collected_refs: %d\n", diagnostics.CollectedRefs)
}

func renderRawOutputsRefDiagnostics(out io.Writer, refs []rawoutputs.RefDiagnostic) {
	_, _ = fmt.Fprintln(out, "  ref_details:")
	if len(refs) == 0 {
		_, _ = fmt.Fprintln(out, "    none")
		return
	}
	for _, ref := range refs {
		_, _ = fmt.Fprintf(
			out,
			"    - ref_id: %s | lifecycle: %s | live: %s | surface: %s | bytes: %d | sha256: %s\n",
			ledgerDisplayValue(ref.Ref.RefID, "missing"),
			ledgerDisplayValue(ref.Lifecycle, "unknown"),
			ledgerDisplayValue(ref.LiveStatus, "unknown"),
			ledgerDisplayValue(ref.Ref.Surface, "unknown"),
			ref.Ref.ByteSize,
			ledgerRawOutputHashPrefix(ref.Ref.ContentHash),
		)
	}
}

func renderRawOutputsGCDryRun(out io.Writer, diagnostics rawoutputs.DiagnosticsResult) {
	_, _ = fmt.Fprintln(out, "  gc_dry_run:")
	if !diagnostics.GCDryRunAvailable {
		_, _ = fmt.Fprintf(out, "    unavailable: %s\n", ledgerDisplayValue(diagnostics.GCDryRunUnavailableReason, "unknown"))
		return
	}
	_, _ = fmt.Fprintf(out, "    tombstone_refs: %d\n", len(diagnostics.GCDryRun.TombstonedRefIDs))
	_, _ = fmt.Fprintf(out, "    collect_artifacts: %d\n", len(diagnostics.GCDryRun.CollectedArtifactIDs))
	_, _ = fmt.Fprintf(out, "    kept_artifacts: %d\n", len(diagnostics.GCDryRun.KeptArtifactIDs))
}
