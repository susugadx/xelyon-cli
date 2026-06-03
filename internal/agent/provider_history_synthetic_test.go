package agent

import (
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
)

type providerHistorySyntheticFixture struct {
	agent *Agent

	rawHistory []api.Message
	rawSession []history.MessageEntry

	readIndex    int
	searchIndex  int
	gatherIndex  int
	testIndex    int
	buildIndex   int
	lintIndex    int
	diffIndex    int
	failIndex    int
	genericIndex int
	editCall     int
}

func TestProviderHistorySyntheticMeasurementHarnessComparesModes(t *testing.T) {
	fixture := newProviderHistorySyntheticFixture(t)

	disabled := fixture.agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionDisabled})
	if !reflect.DeepEqual(disabled.History, fixture.rawHistory) {
		t.Fatalf("disabled projection changed history")
	}
	if !reflect.DeepEqual(disabled.Report, ProviderHistoryProjectionReport{}) {
		t.Fatalf("disabled report = %#v, want empty", disabled.Report)
	}
	assertProviderHistorySyntheticRawStateUnchanged(t, fixture)

	dryRun := fixture.agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionDryRun})
	if !reflect.DeepEqual(dryRun.History, fixture.rawHistory) {
		t.Fatalf("dry-run projection changed history")
	}
	if dryRun.Report.Mode != ProviderHistoryReductionDryRun ||
		dryRun.Report.CandidateCount != 3 ||
		dryRun.Report.ReplacedCount != 0 ||
		dryRun.Report.EstimatedSavedBytes <= 0 ||
		dryRun.Report.ApproxSavedTokens <= 0 ||
		dryRun.Report.ResponsesChainDisabled {
		t.Fatalf("dry-run report = %#v, want three diagnostics with provider-facing estimates and without replacement", dryRun.Report)
	}
	if dryRun.Report.KeptReasonCounts["dry_run"] != 3 ||
		dryRun.Report.KeptReasonCounts["latest_tool_result"] != 1 ||
		dryRun.Report.KeptReasonCounts["write_or_command_tool"] != 9 ||
		dryRun.Report.KeptReasonCounts["missing_tool_call_id"] != 1 ||
		dryRun.Report.KeptReasonCounts["non_contiguous_tool_call_linkage"] != 1 {
		t.Fatalf("dry-run kept reasons = %#v, want dry-run/latest/write-command/invalid linkage counts", dryRun.Report.KeptReasonCounts)
	}
	assertProviderHistorySyntheticCommandDiagnostics(t, dryRun.Report.CommandEditDryRun, 0)
	assertProviderHistorySyntheticRawStateUnchanged(t, fixture)

	apply := fixture.agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})
	if apply.Report.Mode != ProviderHistoryReductionApply ||
		apply.Report.CandidateCount != 3 ||
		apply.Report.ReplacedCount != 3 ||
		apply.Report.EstimatedSavedBytes <= 0 ||
		apply.Report.ApproxSavedTokens <= 0 ||
		!apply.Report.ResponsesChainDisabled {
		t.Fatalf("apply report = %#v, want read/search/gather replacements with savings and chain disabled", apply.Report)
	}
	if apply.Report.CommandEditDryRun.CommandReplacedCount != 3 ||
		apply.Report.CommandEditDryRun.CommandReplacementSavedBytes <= 0 ||
		apply.Report.CommandEditDryRun.ApproxCommandReplacementSavedTokens < providerHistoryCommandReplacementMinSavedTokens*3 {
		t.Fatalf("apply command/edit report = %#v, want three command replacements with savings", apply.Report.CommandEditDryRun)
	}
	assertProviderHistorySyntheticCommandDiagnostics(t, apply.Report.CommandEditDryRun, 3)
	assertProviderHistorySyntheticReadSearchGatherApplied(t, fixture, apply)
	assertProviderHistorySyntheticCommandProjection(t, fixture, apply)
	assertProviderHistorySyntheticEditArgsRaw(t, fixture, apply)
	assertProviderHistorySyntheticRawStateUnchanged(t, fixture)
}

func TestProviderHistorySyntheticMeasurementHarnessKeepsTrailingToolSuffix(t *testing.T) {
	agent := &Agent{History: []api.Message{
		providerHistoryAssistantToolCall("call_old", "read_file"),
		providerHistoryToolResult("call_old", "read_file", strings.Repeat("old read\n", 20)),
		{Role: "assistant", Content: "after old read"},
		providerHistoryAssistantToolCalls(
			providerHistoryToolCall("call_tail_read", "read_file"),
			providerHistoryToolCall("call_tail_search", "search_code"),
		),
		providerHistoryToolResult("call_tail_read", "read_file", "tail read"),
		providerHistoryToolResult("call_tail_search", "search_code", "tail search"),
	}}

	report := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionDryRun}).Report
	if report.CandidateCount != 1 || report.KeptReasonCounts["trailing_tool_suffix"] != 2 {
		t.Fatalf("synthetic trailing report = %#v, want one old candidate and two trailing keeps", report)
	}
}

func newProviderHistorySyntheticFixture(t *testing.T) providerHistorySyntheticFixture {
	t.Helper()
	taskLedger := providerHistoryTaskLedgerWithEvidence(t,
		providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_read_old", Path: "README.md", StartLine: 1, EndLine: 60},
		providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_read_old", Path: "internal/agent/provider_history.go", StartLine: 7, EndLine: 18},
		providerHistoryEvidenceItem{ToolName: "search_code", ToolCallID: "call_search_old", Path: "internal/agent/search.go", StartLine: 20, EndLine: 30},
		providerHistoryEvidenceItem{ToolName: "gather_context", ToolCallID: "call_gather_old", Path: "internal/agent/context.go", StartLine: 40, EndLine: 55},
	)

	var messages []api.Message
	add := func(msg api.Message) int {
		index := len(messages)
		messages = append(messages, msg)
		return index
	}

	oldRead := strings.Repeat("old read_file output with source lines\n", 80)
	oldSearch := strings.Repeat("old search_code match output\n", 70)
	oldGather := strings.Repeat("old gather_context evidence output\n", 70)
	add(api.Message{Role: "user", Content: "measure synthetic provider history reduction"})
	add(providerHistoryAssistantToolCalls(
		providerHistoryToolCall("call_read_old", "read_file"),
		providerHistoryToolCall("call_search_old", "search_code"),
		providerHistoryToolCall("call_gather_old", "gather_context"),
	))
	readIndex := add(providerHistoryToolResult("call_read_old", "read_file", oldRead))
	searchIndex := add(providerHistoryToolResult("call_search_old", "", oldSearch))
	gatherIndex := add(providerHistoryToolResult("call_gather_old", "gather_context", oldGather))
	add(api.Message{Role: "assistant", Content: "old investigation evidence collected"})

	testOutput := providerHistoryLargeSuccessfulTestOutput()
	buildOutput := providerHistoryLargeSuccessfulBuildOutput()
	lintOutput := providerHistoryLargeSuccessfulLintOutput()
	add(providerHistoryAssistantToolCalls(
		providerHistoryToolCallWithJSONArguments(t, "call_test_success", "bash", map[string]string{"command": providerHistorySuccessfulTestCommand}),
		providerHistoryToolCallWithJSONArguments(t, "call_build_success", "command", map[string]string{"command": providerHistorySuccessfulBuildCommand}),
		providerHistoryToolCallWithJSONArguments(t, "call_lint_success", "bash", map[string]string{"command": providerHistorySuccessfulLintCommand}),
	))
	testIndex := add(providerHistoryToolResult("call_test_success", "bash", testOutput))
	buildIndex := add(providerHistoryToolResult("call_build_success", "command", buildOutput))
	lintIndex := add(providerHistoryToolResult("call_lint_success", "bash", lintOutput))
	add(api.Message{Role: "assistant", Content: "safe commands passed"})

	diffOutput := providerHistoryLargeCommandOutput("diff --git a/a.go b/a.go\n")
	failOutput := providerHistoryLargeCommandOutput("--- FAIL: TestSynthetic\nFAIL\t./internal/agent\n")
	genericOutput := providerHistoryLargeCommandOutput("file.txt\n")
	add(providerHistoryAssistantToolCalls(
		providerHistoryToolCallWithJSONArguments(t, "call_diff", "bash", map[string]string{"command": "git diff"}),
		providerHistoryToolCallWithJSONArguments(t, "call_test_fail", "bash", map[string]string{"command": "go test ./..."}),
		providerHistoryToolCallWithJSONArguments(t, "call_generic", "bash", map[string]string{"command": "ls -la"}),
	))
	diffIndex := add(providerHistoryToolResult("call_diff", "bash", diffOutput))
	failIndex := add(providerHistoryToolResult("call_test_fail", "bash", failOutput))
	genericIndex := add(providerHistoryToolResult("call_generic", "bash", genericOutput))
	add(api.Message{Role: "assistant", Content: "unsafe command outputs inspected"})

	writeArgs := providerHistoryJSONArguments(t, map[string]string{"path": "generated/write.go", "content": strings.Repeat("package generated\n", 120)})
	patchArgs := providerHistoryJSONArguments(t, map[string]string{"patch": strings.Repeat("*** Begin Patch\n*** Update File: a.go\n+line\n*** End Patch\n", 40)})
	replaceArgs := providerHistoryJSONArguments(t, map[string]string{
		"path":    "generated/replace.go",
		"old_str": strings.Repeat("old line\n", 80),
		"new_str": strings.Repeat("new line\n", 80),
	})
	writeCall := add(providerHistoryAssistantToolCalls(
		providerHistoryToolCallWithArguments("call_write", "write_file", writeArgs),
		providerHistoryToolCallWithArguments("call_patch", "apply_patch", patchArgs),
		providerHistoryToolCallWithArguments("call_replace", "str_replace", replaceArgs),
	))
	add(providerHistoryToolResult("call_write", "write_file", "wrote generated/write.go"))
	add(providerHistoryToolResult("call_patch", "apply_patch", "patched a.go"))
	add(providerHistoryToolResult("call_replace", "str_replace", "replaced generated/replace.go"))
	add(api.Message{Role: "assistant", Content: "edit tools completed"})

	add(api.Message{Role: "tool", ToolName: "read_file", Content: "missing id old read result"})
	add(api.Message{Role: "assistant", Content: "after missing tool id"})
	add(providerHistoryAssistantToolCall("call_non_contiguous", "read_file"))
	add(api.Message{Role: "assistant", Content: "intervening assistant turn"})
	add(providerHistoryToolResult("call_non_contiguous", "read_file", "non-contiguous old read result"))
	add(api.Message{Role: "assistant", Content: "after invalid linkage"})
	add(providerHistoryAssistantToolCall("call_latest", "read_file"))
	add(providerHistoryToolResult("call_latest", "read_file", "latest read_file output"))
	add(api.Message{Role: "assistant", Content: "final answer"})

	session := history.NewSession("synthetic-model")
	for _, msg := range messages {
		session.AddMessageFromAPI(msg, "synthetic-model")
	}
	agent := &Agent{
		Runtime:      &AgentRuntime{TaskLedger: taskLedger},
		History:      messages,
		CurrentModel: "synthetic-model",
	}
	agent.session = session

	rawSession := append([]history.MessageEntry(nil), session.Messages...)
	return providerHistorySyntheticFixture{
		agent:        agent,
		rawHistory:   api.CloneMessages(messages),
		rawSession:   rawSession,
		readIndex:    readIndex,
		searchIndex:  searchIndex,
		gatherIndex:  gatherIndex,
		testIndex:    testIndex,
		buildIndex:   buildIndex,
		lintIndex:    lintIndex,
		diffIndex:    diffIndex,
		failIndex:    failIndex,
		genericIndex: genericIndex,
		editCall:     writeCall,
	}
}

func assertProviderHistorySyntheticCommandDiagnostics(t *testing.T, report ProviderHistoryCommandEditDryRunReport, wantReplaced int) {
	t.Helper()
	if report.CommandCandidates != 6 ||
		report.EditArgCandidates != 3 ||
		report.CommandOriginalBytes <= 0 ||
		report.EditArgOriginalBytes <= 0 ||
		report.ApproxCommandSavedTokens <= 0 ||
		report.EditArgEstimatedSavedBytes != 0 ||
		report.ApproxEditArgSavedTokens != 0 ||
		report.CommandReplacedCount != wantReplaced {
		t.Fatalf("CommandEditDryRun = %#v, want six command and three edit diagnostics with %d command replacements and no edit estimates", report, wantReplaced)
	}
	wantReasons := map[string]int{
		"test_success_output":    1,
		"build_success_output":   1,
		"lint_success_output":    1,
		"git_diff_output":        1,
		"test_failure_output":    1,
		"command_success_output": 1,
		"write_file_content":     1,
		"apply_patch_patch":      1,
		"str_replace_strings":    1,
	}
	for reason, want := range wantReasons {
		if got := report.CandidateReasonCounts[reason]; got != want {
			t.Fatalf("CandidateReasonCounts[%q] = %d in %#v, want %d", reason, got, report.CandidateReasonCounts, want)
		}
	}
}

func assertProviderHistorySyntheticReadSearchGatherApplied(t *testing.T, fixture providerHistorySyntheticFixture, result providerHistoryProjectionResult) {
	t.Helper()
	for _, item := range []struct {
		callID string
		index  int
	}{
		{callID: "call_read_old", index: fixture.readIndex},
		{callID: "call_search_old", index: fixture.searchIndex},
		{callID: "call_gather_old", index: fixture.gatherIndex},
	} {
		if result.History[item.index].Content == fixture.rawHistory[item.index].Content {
			t.Fatalf("tool result %s was not replaced", item.callID)
		}
		candidate := candidateByToolCallID(result.Report, item.callID)
		if candidate == nil || !candidate.ReplacementApplied || len(candidate.EvidencePointers) == 0 {
			t.Fatalf("candidate %s = %#v, want applied candidate with evidence pointers", item.callID, candidate)
		}
	}
	readCandidate := candidateByToolCallID(result.Report, "call_read_old")
	if readCandidate == nil || len(readCandidate.EvidencePointers) != 2 {
		t.Fatalf("read candidate = %#v, want two matched evidence pointers", readCandidate)
	}
}

func assertProviderHistorySyntheticCommandProjection(t *testing.T, fixture providerHistorySyntheticFixture, result providerHistoryProjectionResult) {
	t.Helper()
	for _, item := range []struct {
		index int
		label string
	}{
		{index: fixture.testIndex, label: providerHistorySuccessfulTestReplacementLabel},
		{index: fixture.buildIndex, label: providerHistorySuccessfulBuildReplacementLabel},
		{index: fixture.lintIndex, label: providerHistorySuccessfulLintReplacementLabel},
	} {
		assertProviderHistoryCommandContentReplacement(t, result.History[item.index].Content, fixture.rawHistory[item.index].Content, item.label)
	}
	for _, index := range []int{fixture.diffIndex, fixture.failIndex, fixture.genericIndex} {
		if result.History[index].Content != fixture.rawHistory[index].Content {
			t.Fatalf("unsafe/generic command at history[%d] changed:\n got %q\nwant %q", index, result.History[index].Content, fixture.rawHistory[index].Content)
		}
	}
}

func assertProviderHistorySyntheticEditArgsRaw(t *testing.T, fixture providerHistorySyntheticFixture, result providerHistoryProjectionResult) {
	t.Helper()
	if !reflect.DeepEqual(result.History[fixture.editCall].ToolCalls, fixture.rawHistory[fixture.editCall].ToolCalls) {
		t.Fatalf("edit tool arguments changed at history[%d]:\n got %#v\nwant %#v", fixture.editCall, result.History[fixture.editCall].ToolCalls, fixture.rawHistory[fixture.editCall].ToolCalls)
	}
}

func assertProviderHistorySyntheticRawStateUnchanged(t *testing.T, fixture providerHistorySyntheticFixture) {
	t.Helper()
	if !reflect.DeepEqual(fixture.agent.History, fixture.rawHistory) {
		t.Fatalf("Agent.History changed after projection:\n got %#v\nwant %#v", fixture.agent.History, fixture.rawHistory)
	}
	if fixture.agent.session == nil || !reflect.DeepEqual(fixture.agent.session.Messages, fixture.rawSession) {
		t.Fatalf("Session.Messages changed after projection:\n got %#v\nwant %#v", fixture.agent.session.Messages, fixture.rawSession)
	}
}
