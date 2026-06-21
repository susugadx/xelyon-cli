package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
)

type createVerifyOnlyRawOutputArtifactStore struct {
	inner *rawoutputs.Store
}

func (s createVerifyOnlyRawOutputArtifactStore) Create(ctx context.Context, req rawoutputs.CreateRequest) (rawoutputs.CreateResult, error) {
	return s.inner.Create(ctx, req)
}

func (s createVerifyOnlyRawOutputArtifactStore) MaterializeLegacy(ctx context.Context, req rawoutputs.LegacyMaterializeRequest) (rawoutputs.CreateResult, error) {
	return s.inner.MaterializeLegacy(ctx, req)
}

func (s createVerifyOnlyRawOutputArtifactStore) Verify(ctx context.Context, ref rawoutputs.RawOutputRef) (rawoutputs.VerifyResult, error) {
	return s.inner.Verify(ctx, ref)
}

type countingRawOutputArtifactStore struct {
	inner       *rawoutputs.Store
	createCalls int
	verifyCalls int
	scanCalls   int
	lookupCalls int
}

func (s *countingRawOutputArtifactStore) Create(ctx context.Context, req rawoutputs.CreateRequest) (rawoutputs.CreateResult, error) {
	s.createCalls++
	return s.inner.Create(ctx, req)
}

func (s *countingRawOutputArtifactStore) MaterializeLegacy(ctx context.Context, req rawoutputs.LegacyMaterializeRequest) (rawoutputs.CreateResult, error) {
	s.createCalls++
	return s.inner.MaterializeLegacy(ctx, req)
}

func (s *countingRawOutputArtifactStore) Verify(ctx context.Context, ref rawoutputs.RawOutputRef) (rawoutputs.VerifyResult, error) {
	s.verifyCalls++
	return s.inner.Verify(ctx, ref)
}

func (s *countingRawOutputArtifactStore) Scan(ctx context.Context, req rawoutputs.ScanRequest) (rawoutputs.ScanResult, error) {
	s.scanCalls++
	return s.inner.Scan(ctx, req)
}

func (s *countingRawOutputArtifactStore) LookupRef(ctx context.Context, sessionID, refID string) (rawoutputs.RawOutputRef, error) {
	s.lookupCalls++
	return s.inner.LookupRef(ctx, sessionID, refID)
}

func newProviderHistoryRawOutputRequestAgent(t *testing.T) (*Agent, *providerFacingHistoryMutationProbe, *rawoutputs.Store) {
	t.Helper()
	disableColors(t)
	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{}
	agent := newChatRequestTestAgent(t, provider, &out)
	applyActiveContextProviderFixture(agent, activeContextOpenAIResponses)
	store, err := rawoutputs.OpenStore(rawoutputs.Root(t.TempDir()), rawoutputs.StoreOptions{})
	if err != nil {
		t.Fatalf("rawoutputs.OpenStore() error = %v", err)
	}
	return agent, provider, store
}

func configureProviderHistoryRawOutputRequestApply(agent *Agent, budgetTokens, maxBudgetTokens int) {
	agent.Runtime.Options.ProviderHistoryReductionMode = ProviderHistoryReductionApply
	agent.Runtime.Options.ProviderHistoryReductionModeSet = true
	agent.Runtime.Options.EnableProviderHistoryRehydrateContext = true
	agent.Runtime.Options.ProviderHistoryRawOutputArtifacts = config.ProviderHistoryRawOutputArtifactsConfig{
		Mode:                         config.ProviderHistoryRawOutputArtifactsModeApply,
		ActiveContextBudgetTokens:    budgetTokens,
		ActiveContextBudgetMaxTokens: maxBudgetTokens,
	}
}

func syncProviderHistoryRawOutputRequestSession(agent *Agent) {
	for _, msg := range agent.History {
		agent.session.AddMessageFromAPI(msg, agent.CurrentModel)
	}
}

func providerHistoryLargeSafeWebSearchResult() string {
	return strings.Repeat(`1. OpenAI Responses API guide
URL: https://example.test/docs/responses?utm_campaign=private#private-fragment
safe web search snippet about response ids and follow-up calls.
2. OpenAI API reference
URL: https://platform.openai.com/docs/api-reference/responses
safe web search snippet about response fields.
`, 180)
}
