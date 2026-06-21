package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	"github.com/susugadx/xelyon-cli/internal/token"
)

func TestNormalModeRequestApplyCompactsDataBearingCommandAndInjectsRawOutputContext(t *testing.T) {
	agent, provider, store := newProviderHistoryRawOutputRequestAgent(t)
	commandOutput := providerHistoryNumberedLines("api-result", 6000)
	command := "curl 'https://api.example.test/items?foo=bar#frag'"
	agent.Runtime.RawOutputArtifactStore = store
	configureProviderHistoryRawOutputRequestApply(agent, 4096, 8192)
	agent.History = []api.Message{
		{Role: "user", Content: "inspect api history"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_curl", "bash", map[string]string{"command": command})),
		providerHistoryToolResult("call_curl", "bash", commandOutput),
		{Role: "assistant", Content: "api data reviewed"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "ready"},
	}
	syncProviderHistoryRawOutputRequestSession(agent)
	beforeHistory := api.CloneMessages(agent.History)
	beforeSession := append(agent.session.Messages[:0:0], agent.session.Messages...)

	if err := agent.chatInternal("show api-result-3000", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	projected := provider.capturedHistory[2].Content
	if projected == commandOutput ||
		!strings.Contains(projected, "[compacted old data-bearing command output;") ||
		!strings.Contains(projected, "raw_output_ref=") {
		t.Fatalf("provider command output = %q, want artifact-backed placeholder", projected)
	}
	if !provider.capturedResponseIDChainDisabled {
		t.Fatal("provider request context did not disable response ID chain for artifact-backed command replacement")
	}
	if len(provider.capturedActiveContextBlocks) != 1 {
		t.Fatalf("active context blocks = %#v, want one raw output block", provider.capturedActiveContextBlocks)
	}
	block := provider.capturedActiveContextBlocks[0]
	if block.Name != providerHistoryRawOutputActiveContextName {
		t.Fatalf("active context block name = %q, want %q", block.Name, providerHistoryRawOutputActiveContextName)
	}
	for _, want := range []string{
		providerHistoryRawOutputContextHeader,
		"ref: rawout_",
		"surface: command_output",
		"tool_name: bash",
		"?redacted",
		"#redacted",
		"family: network",
		"classifier: network_response",
		"matched raw output excerpt",
		"api-result-3000",
	} {
		if !strings.Contains(block.Content, want) {
			t.Fatalf("raw output active context missing %q:\n%s", want, block.Content)
		}
	}
	for _, reject := range []string{"foo=bar", "#frag", "api-result-0001", "api-result-6000"} {
		if strings.Contains(block.Content, reject) {
			t.Fatalf("raw output active context leaked %q:\n%s", reject, block.Content)
		}
	}
	for i, want := range beforeHistory {
		if !reflect.DeepEqual(agent.History[i], want) {
			t.Fatalf("Agent.History[%d] changed after raw output artifact request:\n got %#v\nwant %#v", i, agent.History[i], want)
		}
	}
	for i, want := range beforeSession {
		if !reflect.DeepEqual(agent.session.Messages[i], want) {
			t.Fatalf("session.Messages[%d] changed after raw output artifact request:\n got %#v\nwant %#v", i, agent.session.Messages[i], want)
		}
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.CommandEditDryRun.ArtifactBackedCommandReplacedCount != 1 ||
		report.RawOutputRefCount != 1 ||
		!report.ResponsesChainDisabled {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want artifact-backed command replacement report", report)
	}
}

func TestTokenBudgetHistoryDoesNotMaterializeRawOutputArtifacts(t *testing.T) {
	agent, provider, store := newProviderHistoryRawOutputRequestAgent(t)
	countingStore := &countingRawOutputArtifactStore{inner: store}
	commandOutput := providerHistoryNumberedLines("api-result", 6000)
	agent.Runtime.RawOutputArtifactStore = countingStore
	configureProviderHistoryRawOutputRequestApply(agent, 4096, 8192)
	agent.History = []api.Message{
		{Role: "user", Content: "inspect api history"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_curl", "bash", map[string]string{"command": "curl https://api.example.test/items"})),
		providerHistoryToolResult("call_curl", "bash", commandOutput),
		{Role: "assistant", Content: "api data reviewed"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "ready"},
	}
	syncProviderHistoryRawOutputRequestSession(agent)

	projectedForBudget := agent.tokenBudgetHistory()
	if projectedForBudget[2].Content != commandOutput {
		t.Fatalf("token budget history command output = %q, want raw output without artifact materialization", projectedForBudget[2].Content)
	}
	if blocks := agent.providerFacingActiveContextBlocksForTokenBudget(context.Background()); len(blocks) != 0 {
		t.Fatalf("token budget active context blocks = %#v, want none without artifact materialization", blocks)
	}
	if countingStore.createCalls != 0 || countingStore.verifyCalls != 0 {
		t.Fatalf("token budget artifact calls = create:%d verify:%d, want no side effects", countingStore.createCalls, countingStore.verifyCalls)
	}

	if err := agent.chatInternal("show api-result-3000", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}
	if countingStore.createCalls == 0 || countingStore.verifyCalls == 0 {
		t.Fatalf("request artifact calls = create:%d verify:%d, want materialization during provider request", countingStore.createCalls, countingStore.verifyCalls)
	}
	if countingStore.scanCalls == 0 {
		t.Fatalf("request raw output scan calls = %d, want streaming active-context scan", countingStore.scanCalls)
	}
	if projected := provider.capturedHistory[2].Content; projected == commandOutput || !strings.Contains(projected, "raw_output_ref=") {
		t.Fatalf("provider command output = %q, want artifact-backed placeholder during request", projected)
	}
}

func TestNormalModeRequestApplyKeepsDataBearingCommandRawWhenRawOutputContextCoverageInsufficient(t *testing.T) {
	agent, provider, store := newProviderHistoryRawOutputRequestAgent(t)
	commandOutput := providerHistoryNumberedLines("api-result", 6000)
	agent.Runtime.RawOutputArtifactStore = store
	configureProviderHistoryRawOutputRequestApply(agent, 4096, 8192)
	agent.History = []api.Message{
		{Role: "user", Content: "inspect api history"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_curl", "bash", map[string]string{"command": "curl https://api.example.test/items"})),
		providerHistoryToolResult("call_curl", "bash", commandOutput),
		{Role: "assistant", Content: "api data reviewed"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "ready"},
	}
	syncProviderHistoryRawOutputRequestSession(agent)

	if err := agent.chatInternal("next request", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	if got := provider.capturedHistory[2].Content; got != commandOutput {
		t.Fatalf("provider command output = %q, want raw output when active context coverage is insufficient", got)
	}
	if len(provider.capturedActiveContextBlocks) != 0 {
		t.Fatalf("active context blocks = %#v, want none after coverage fail-closed fallback", provider.capturedActiveContextBlocks)
	}
	if provider.capturedResponseIDChainDisabled {
		t.Fatal("response ID chain disabled despite no provider-facing replacement")
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.CommandEditDryRun.ArtifactBackedCommandReplacedCount != 0 ||
		report.CommandEditDryRun.ArtifactBackedCommandApplyEligible != 0 ||
		report.CommandEditDryRun.ArtifactBackedCommandCandidates != 1 ||
		report.ResponsesChainDisabled {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want raw-output dry-run candidate without apply", report)
	}
	if got := report.CommandEditDryRun.ArtifactBackedKeptReasonCounts[providerHistoryRawOutputActiveContextCoverageInsufficientReason]; got != 1 {
		t.Fatalf("ArtifactBackedKeptReasonCounts = %#v, want active context coverage insufficient", report.CommandEditDryRun.ArtifactBackedKeptReasonCounts)
	}
}

func TestTokenBudgetHistoryDoesNotOpenRawOutputArtifactStore(t *testing.T) {
	agent, _, _ := newProviderHistoryRawOutputRequestAgent(t)
	commandOutput := providerHistoryNumberedLines("api-result", 6000)
	root := filepath.Join(t.TempDir(), "rawoutputs")
	agent.Runtime.RawOutputArtifactStore = nil
	agent.Runtime.RawOutputArtifactRoot = root
	configureProviderHistoryRawOutputRequestApply(agent, 4096, 8192)
	agent.History = []api.Message{
		{Role: "user", Content: "inspect api history"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_curl", "bash", map[string]string{"command": "curl https://api.example.test/items"})),
		providerHistoryToolResult("call_curl", "bash", commandOutput),
		{Role: "assistant", Content: "api data reviewed"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "ready"},
	}

	projectedForBudget := agent.tokenBudgetHistory()
	if projectedForBudget[2].Content != commandOutput {
		t.Fatalf("token budget history command output = %q, want raw output without opening artifact store", projectedForBudget[2].Content)
	}
	if agent.Runtime.RawOutputArtifactStore != nil {
		t.Fatalf("runtime RawOutputArtifactStore = %#v, want nil after token budget estimate", agent.Runtime.RawOutputArtifactStore)
	}
	if _, err := os.Stat(root); err == nil {
		t.Fatalf("raw output artifact root %s exists after token budget estimate, want no store I/O", root)
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat(raw output artifact root) error = %v", err)
	}
}

func TestNormalModeRequestApplyKeepsDataBearingCommandRawWhenRawOutputContextCannotResolve(t *testing.T) {
	agent, provider, store := newProviderHistoryRawOutputRequestAgent(t)
	commandOutput := providerHistoryNumberedLines("api-result", 6000)
	agent.Runtime.RawOutputArtifactStore = createVerifyOnlyRawOutputArtifactStore{inner: store}
	configureProviderHistoryRawOutputRequestApply(agent, 4096, 8192)
	agent.History = []api.Message{
		{Role: "user", Content: "inspect api history"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_curl", "bash", map[string]string{"command": "curl https://api.example.test/items"})),
		providerHistoryToolResult("call_curl", "bash", commandOutput),
		{Role: "assistant", Content: "api data reviewed"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "ready"},
	}
	syncProviderHistoryRawOutputRequestSession(agent)

	if projectedForBudget := agent.tokenBudgetHistory(); projectedForBudget[2].Content != commandOutput {
		t.Fatalf("token budget history command output = %q, want raw output when raw context cannot resolve required ref", projectedForBudget[2].Content)
	}
	if blocks := agent.providerFacingActiveContextBlocksForTokenBudget(context.Background()); len(blocks) != 0 {
		t.Fatalf("token budget active context blocks = %#v, want none after raw-output budget fail-closed fallback", blocks)
	}

	if err := agent.chatInternal("show api-result-3000", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	if got := provider.capturedHistory[2].Content; got != commandOutput {
		t.Fatalf("provider command output = %q, want raw output when raw context cannot resolve", got)
	}
	if len(provider.capturedActiveContextBlocks) != 0 {
		t.Fatalf("active context blocks = %#v, want none after raw-output fail-closed fallback", provider.capturedActiveContextBlocks)
	}
	if provider.capturedResponseIDChainDisabled {
		t.Fatal("response ID chain disabled despite no provider-facing replacement")
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.CommandEditDryRun.ArtifactBackedCommandReplacedCount != 0 ||
		report.CommandEditDryRun.ArtifactBackedCommandApplyEligible != 0 ||
		report.CommandEditDryRun.ArtifactBackedCommandCandidates != 1 ||
		report.ResponsesChainDisabled {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want raw-output dry-run candidate without apply", report)
	}
	if got := report.CommandEditDryRun.ArtifactBackedKeptReasonCounts["raw_output_active_context_required_refs_missing"]; got != 1 {
		t.Fatalf("ArtifactBackedKeptReasonCounts = %#v, want active context required refs missing", report.CommandEditDryRun.ArtifactBackedKeptReasonCounts)
	}
}

func TestNormalModeRequestApplyKeepsDataBearingCommandRawWhenRawOutputContextBudgetCannotFit(t *testing.T) {
	agent, provider, store := newProviderHistoryRawOutputRequestAgent(t)
	commandOutput := providerHistoryNumberedLines("api-result", 6000)
	agent.Runtime.RawOutputArtifactStore = store
	configureProviderHistoryRawOutputRequestApply(agent, 1, 1)
	agent.History = []api.Message{
		{Role: "user", Content: "inspect api history"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_curl", "bash", map[string]string{"command": "curl https://api.example.test/items"})),
		providerHistoryToolResult("call_curl", "bash", commandOutput),
		{Role: "assistant", Content: "api data reviewed"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "ready"},
	}
	syncProviderHistoryRawOutputRequestSession(agent)

	if projectedForBudget := agent.tokenBudgetHistory(); projectedForBudget[2].Content != commandOutput {
		t.Fatalf("token budget history command output = %q, want raw output when active context budget cannot fit required ref", projectedForBudget[2].Content)
	}
	if blocks := agent.providerFacingActiveContextBlocksForTokenBudget(context.Background()); len(blocks) != 0 {
		t.Fatalf("token budget active context blocks = %#v, want none after raw-output budget fail-closed fallback", blocks)
	}

	if err := agent.chatInternal("show api-result-3000", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	if got := provider.capturedHistory[2].Content; got != commandOutput {
		t.Fatalf("provider command output = %q, want raw output when raw context budget cannot fit required ref", got)
	}
	if len(provider.capturedActiveContextBlocks) != 0 {
		t.Fatalf("active context blocks = %#v, want none after raw-output budget fail-closed fallback", provider.capturedActiveContextBlocks)
	}
	if provider.capturedResponseIDChainDisabled {
		t.Fatal("response ID chain disabled despite no provider-facing replacement")
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.CommandEditDryRun.ArtifactBackedCommandReplacedCount != 0 ||
		report.CommandEditDryRun.ArtifactBackedCommandApplyEligible != 0 ||
		report.CommandEditDryRun.ArtifactBackedCommandCandidates != 1 ||
		report.ResponsesChainDisabled {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want raw-output dry-run candidate without apply", report)
	}
	if got := report.CommandEditDryRun.ArtifactBackedKeptReasonCounts["raw_output_active_context_required_refs_missing"]; got != 1 {
		t.Fatalf("ArtifactBackedKeptReasonCounts = %#v, want active context required refs missing", report.CommandEditDryRun.ArtifactBackedKeptReasonCounts)
	}
}

func TestBuildProviderHistoryRawOutputActiveContextShrinksPreRenderedExcerptAfterMetadataBudget(t *testing.T) {
	agent, _, store := newProviderHistoryRawOutputRequestAgent(t)
	term := "target-4242"
	commandOutput, matchLine, matchIndex, totalLines := providerHistoryRawOutputTightBudgetFixture(term)
	created, err := store.Create(context.Background(), rawoutputs.CreateRequest{
		Surface:   rawoutputs.SurfaceCommandOutput,
		SessionID: "session-tight-active-context",
		Source: rawoutputs.SourceMetadata{
			CommandHash:    "sha256:test-tight-budget",
			CommandPreview: "curl https://api.example.test/items",
			ToolName:       "bash",
			ToolCallID:     "call_tight_budget",
			EventID:        "tool_call:call_tight_budget",
		},
		Classification: rawoutputs.ClassificationMetadata{
			SemanticRole: "data_bearing",
			Family:       "network",
			Classifier:   "network_response",
		},
		Body:          strings.NewReader(commandOutput),
		SizeHintBytes: int64(len(commandOutput)),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	agent.Runtime.RawOutputArtifactStore = store
	raw := []api.Message{{Role: "user", Content: "show " + term}}
	hints := providerHistoryRawOutputRehydrateHintsFromRaw(raw)
	scannerBudget := providerHistoryRawOutputTightScannerBudget(t, commandOutput, created.Ref, hints, matchLine, matchIndex, totalLines)
	totalBudget := token.EstimateTokenCount(providerHistoryRawOutputContextHeader) + scannerBudget
	configureProviderHistoryRawOutputRequestApply(agent, totalBudget, totalBudget)

	build := agent.buildProviderHistoryRawOutputActiveContext(context.Background(), ProviderHistoryProjectionReport{
		RawOutputContextRefs: []rawoutputs.RawOutputRef{created.Ref},
	}, raw)

	if build.missingRequiredRefs() {
		t.Fatalf("active context build = %#v, want shrinkable matched excerpt injected", build)
	}
	if len(build.Blocks) != 1 {
		t.Fatalf("active context blocks = %#v, want one block", build.Blocks)
	}
	content := build.Blocks[0].Content
	if !strings.Contains(content, term) || !strings.Contains(content, "matched raw output excerpt") {
		t.Fatalf("active context block missing shrunk matched excerpt:\n%s", content)
	}
	if strings.Contains(content, "context line 0001") || strings.Contains(content, "context line 0899") {
		t.Fatalf("active context block retained far context instead of shrinking:\n%s", content)
	}
}

func TestNormalModeRequestApplyCompactsRunSkillScriptResultAndInjectsRawOutputContext(t *testing.T) {
	agent, provider, store := newProviderHistoryRawOutputRequestAgent(t)
	output := providerHistoryNumberedLines("skill-script-output", 6000)
	agent.Runtime.RawOutputArtifactStore = store
	configureProviderHistoryRawOutputRequestApply(agent, 4096, 8192)
	agent.History = []api.Message{
		{Role: "user", Content: "inspect skill script history"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_script", "run_skill_script", map[string]string{
			"skill":     "coverage-audit",
			"script":    "scripts/report.sh",
			"args_json": `["--format","json","--path","internal/providerhistory"]`,
		})),
		providerHistoryToolResult("call_script", "run_skill_script", output),
		{Role: "assistant", Content: "skill script data reviewed"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "ready"},
	}
	syncProviderHistoryRawOutputRequestSession(agent)
	beforeHistory := api.CloneMessages(agent.History)
	beforeSession := append(agent.session.Messages[:0:0], agent.session.Messages...)

	if err := agent.chatInternal("show skill-script-output-3000", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	projected := provider.capturedHistory[2].Content
	if projected == output ||
		!strings.Contains(projected, "[compacted old run_skill_script result;") ||
		!strings.Contains(projected, "raw_output_ref=") {
		t.Fatalf("provider run_skill_script output = %q, want artifact-backed placeholder", projected)
	}
	for _, reject := range []string{"--format", "internal/providerhistory", "args_json"} {
		if strings.Contains(projected, reject) {
			t.Fatalf("provider run_skill_script placeholder leaked argument detail %q:\n%s", reject, projected)
		}
	}
	if !provider.capturedResponseIDChainDisabled {
		t.Fatal("provider request context did not disable response ID chain for run_skill_script artifact replacement")
	}
	if len(provider.capturedActiveContextBlocks) != 1 {
		t.Fatalf("active context blocks = %#v, want one raw output block", provider.capturedActiveContextBlocks)
	}
	block := provider.capturedActiveContextBlocks[0]
	for _, want := range []string{
		"surface: command_output",
		"tool_name: run_skill_script",
		"command_preview: run_skill_script skill=coverage-audit script=scripts/report.sh",
		"family: run_skill_script",
		"classifier: skill_script_output",
		"matched raw output excerpt",
		"skill-script-output-3000",
	} {
		if !strings.Contains(block.Content, want) {
			t.Fatalf("raw output active context missing %q:\n%s", want, block.Content)
		}
	}
	for _, reject := range []string{"--format", "internal/providerhistory", "args_json"} {
		if strings.Contains(block.Content, reject) {
			t.Fatalf("raw output active context leaked argument detail %q:\n%s", reject, block.Content)
		}
	}
	for i, want := range beforeHistory {
		if !reflect.DeepEqual(agent.History[i], want) {
			t.Fatalf("Agent.History[%d] changed after run_skill_script raw output request:\n got %#v\nwant %#v", i, agent.History[i], want)
		}
	}
	for i, want := range beforeSession {
		if !reflect.DeepEqual(agent.session.Messages[i], want) {
			t.Fatalf("session.Messages[%d] changed after run_skill_script raw output request:\n got %#v\nwant %#v", i, agent.session.Messages[i], want)
		}
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.ReplacedCount != 1 ||
		report.RawOutputRefCount != 1 ||
		report.DataBearingCandidateCount != 1 ||
		!report.ResponsesChainDisabled {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want applied run_skill_script artifact report", report)
	}
}

func TestNormalModeRequestApplyCompactsWebSearchResultAndInjectsRedactedRawOutputContext(t *testing.T) {
	agent, provider, store := newProviderHistoryRawOutputRequestAgent(t)
	webOutput := providerHistoryLargeSafeWebSearchResult()
	query := "OpenAI Responses API previous_response_id documentation"
	agent.Runtime.RawOutputArtifactStore = store
	configureProviderHistoryRawOutputRequestApply(agent, 4096, 8192)
	agent.History = []api.Message{
		{Role: "user", Content: "inspect web search history"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_web_old", "web_search", map[string]string{"query": query})),
		providerHistoryToolResult("call_web_old", "web_search", webOutput),
		{Role: "assistant", Content: "web data reviewed"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_web_dup", "web_search", map[string]string{"query": query})),
		providerHistoryToolResult("call_web_dup", "web_search", webOutput),
		{Role: "assistant", Content: "duplicate raw web result remains"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "ready"},
	}
	syncProviderHistoryRawOutputRequestSession(agent)

	if err := agent.chatInternal("show response ids", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	projected := provider.capturedHistory[2].Content
	if projected == webOutput ||
		!strings.Contains(projected, "[compacted old XELYON web_search tool result;") ||
		!strings.Contains(projected, "raw_output_ref=") ||
		strings.Contains(projected, "utm_campaign=private") ||
		strings.Contains(projected, "private-fragment") {
		t.Fatalf("provider web_search output = %q, want artifact-backed redacted placeholder", projected)
	}
	if provider.capturedHistory[5].Content != webOutput {
		t.Fatalf("later duplicate web_search output changed")
	}
	if len(provider.capturedActiveContextBlocks) != 1 {
		t.Fatalf("active context blocks = %#v, want one raw output block", provider.capturedActiveContextBlocks)
	}
	block := provider.capturedActiveContextBlocks[0]
	for _, want := range []string{
		"surface: xelyon_web_search_tool_result",
		"tool_name: web_search",
		"command_preview: web_search query=OpenAI Responses API previous_response_id documentation",
		"family: web_search",
		"classifier: web_search_result",
		"https://example.test/docs/responses?redacted#redacted",
		"safe web search snippet",
	} {
		if !strings.Contains(block.Content, want) {
			t.Fatalf("raw output active context missing %q:\n%s", want, block.Content)
		}
	}
	for _, reject := range []string{"utm_campaign=private", "private-fragment"} {
		if strings.Contains(block.Content, reject) {
			t.Fatalf("raw output active context leaked %q:\n%s", reject, block.Content)
		}
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.ReplacedCount != 1 ||
		report.RawOutputRefCount != 1 ||
		report.DataBearingCandidateCount != 1 ||
		!report.ResponsesChainDisabled {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want applied web_search artifact report", report)
	}
}

func providerHistoryRawOutputTightBudgetFixture(term string) (string, string, int, int) {
	const totalLines = 900
	const matchIndex = 450
	var b strings.Builder
	matchLine := term + " stable matched payload"
	for i := 0; i < totalLines; i++ {
		if i == matchIndex {
			b.WriteString(matchLine)
			b.WriteByte('\n')
			continue
		}
		fmt.Fprintf(&b, "context line %04d %s\n", i, strings.Repeat("context ", 10))
	}
	return b.String(), matchLine, matchIndex, totalLines
}

func providerHistoryRawOutputTightScannerBudget(t *testing.T, body string, ref rawoutputs.RawOutputRef, hints []string, matchLine string, matchIndex, totalLines int) int {
	t.Helper()
	if len(hints) == 0 {
		t.Fatal("test setup invalid: missing raw output hints")
	}
	metadataTokens := token.EstimateTokenCount(providerHistoryRawOutputContextEntryMetadata(ref))
	matchOnly := providerHistoryRawOutputRenderMatchedExcerpt([]string{matchLine}, hints[0], matchIndex, totalLines, matchIndex, matchIndex+1)
	matchOnlyTokens := token.EstimateTokenCount(matchOnly)
	for scannerBudget := metadataTokens + matchOnlyTokens + 1; scannerBudget <= metadataTokens+matchOnlyTokens+220; scannerBudget++ {
		scanner := newProviderHistoryRawOutputContextScanner(hints, scannerBudget)
		if err := scanner.Scan([]byte(body)); err != nil {
			t.Fatalf("Scan() error = %v", err)
		}
		preRendered, reason := scanner.Body()
		if reason != "" {
			continue
		}
		reducedBodyBudget := scannerBudget - metadataTokens
		preRenderedTokens := token.EstimateTokenCount(preRendered)
		if reducedBodyBudget > 0 &&
			preRenderedTokens <= scannerBudget &&
			preRenderedTokens > reducedBodyBudget &&
			matchOnlyTokens <= reducedBodyBudget {
			return scannerBudget
		}
	}
	t.Fatalf("failed to find tight scanner budget for metadata shrink regression")
	return 0
}
