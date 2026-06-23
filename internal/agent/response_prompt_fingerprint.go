package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

const responsePromptFingerprintPrefix = "xelyon.response_prompt.v1"

func responsePromptFingerprintFor(systemPrompt string) string {
	systemPrompt = canonicalResponsePromptForFingerprint(systemPrompt)
	if systemPrompt == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(responsePromptFingerprintPrefix + "\n" + systemPrompt))
	return hex.EncodeToString(sum[:])
}

func canonicalResponsePromptForFingerprint(systemPrompt string) string {
	systemPrompt = strings.ReplaceAll(systemPrompt, "\r\n", "\n")
	systemPrompt = strings.ReplaceAll(systemPrompt, "\r", "\n")
	lines := strings.Split(systemPrompt, "\n")
	out := make([]string, 0, len(lines))
	prevEmpty := true
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		empty := strings.TrimSpace(line) == ""
		if empty {
			if prevEmpty {
				continue
			}
			out = append(out, "")
			prevEmpty = true
			continue
		}
		out = append(out, line)
		prevEmpty = false
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

func (a *Agent) prepareResponseContextForPrompt(ctx context.Context, systemPrompt string) context.Context {
	if a == nil {
		return ctx
	}
	ridProvider, ok := a.CurrentProvider.(ResponseIDCapable)
	if !ok || !ridProvider.HasCachedResponseID() {
		return ctx
	}

	fingerprint := responsePromptFingerprintFor(systemPrompt)
	snapshot := a.responseContextSnapshotForCurrentProvider(ridProvider)
	if snapshot.shouldReuseForRuntimePrompt(a.CurrentModel, a.ProviderName, a.currentProviderConfigKey(), fingerprint) {
		return ctx
	}

	ridProvider.SetResponseID("")
	a.responseContext = responseContextSnapshot{}
	clearSavedResponseContext(a.session)
	return api.WithResponseIDChainDisabled(ctx)
}

func (a *Agent) recordResponseContextForPrompt(systemPrompt string) {
	if a == nil {
		return
	}
	ridProvider, ok := a.CurrentProvider.(ResponseIDCapable)
	if !ok || !ridProvider.HasCachedResponseID() {
		return
	}
	fingerprint := responsePromptFingerprintFor(systemPrompt)
	if fingerprint == "" {
		return
	}
	snapshot := responseContextSnapshotFromRuntime(
		a.CurrentModel,
		a.ProviderName,
		a.currentProviderConfigKey(),
		ridProvider.GetResponseID(),
	)
	snapshot.promptFingerprint = fingerprint
	a.responseContext = snapshot
}

func (a *Agent) responseContextSnapshotForCurrentProvider(ridProvider ResponseIDCapable) responseContextSnapshot {
	if a == nil || ridProvider == nil {
		return responseContextSnapshot{}
	}
	responseID := strings.TrimSpace(ridProvider.GetResponseID())
	if responseID == "" {
		return responseContextSnapshot{}
	}
	if a.responseContext.responseID == responseID {
		return a.responseContext
	}
	if a.session != nil && strings.TrimSpace(a.session.ResponseID) == responseID {
		return responseContextSnapshotFromSession(a.session)
	}
	return responseContextSnapshotFromRuntime(
		a.CurrentModel,
		a.ProviderName,
		a.currentProviderConfigKey(),
		responseID,
	)
}
