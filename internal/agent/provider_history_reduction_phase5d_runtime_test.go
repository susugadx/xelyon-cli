package agent

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
)

func TestPhase5DStatusDiagnosticUsesLastProviderFacingRequestReport(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{}
	agent := newChatRequestTestAgent(t, provider, &out)
	oldRead := seedProviderHistoryReductionRequestFixture(t, agent, "call_status_old")
	agent.session.AddMessageFromAPI(agent.History[0], agent.CurrentModel)
	agent.session.AddMessageFromAPI(agent.History[1], agent.CurrentModel)
	beforeSession := append(agent.session.Messages[:0:0], agent.session.Messages...)

	if err := agent.chatInternal("next request", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	reportAfterRequest := cloneProviderHistoryProjectionReport(agent.Runtime.LastProviderHistoryProjectionReport)
	if reportAfterRequest.Mode != ProviderHistoryReductionApply || reportAfterRequest.ReplacedCount != 1 {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want normal request apply report", reportAfterRequest)
	}
	out.Reset()
	statusOutput := renderProviderHistoryStatusCommand(t, agent, &out)
	for _, want := range []string{"Provider history reduction", "provider history reduction: apply", "content_replacements=1", "total_provider_facing_saved=", "approx_total_provider_facing_saved_tokens=", "responses_chain_disabled=true"} {
		if !strings.Contains(statusOutput, want) {
			t.Fatalf("/status output missing %q:\n%s", want, statusOutput)
		}
	}
	beforeHistory := api.CloneMessages(agent.History)
	if !reflect.DeepEqual(agent.session.Messages[:len(beforeSession)], beforeSession) {
		t.Fatalf("session.Messages prefix changed after request setup:\n got %#v\nwant %#v", agent.session.Messages[:len(beforeSession)], beforeSession)
	}

	var repairCaptured []api.Message
	agent.CurrentProvider = &scriptedChatProvider{
		name: "gemini",
		chatWithToolsFn: func(_ int, _ context.Context, _ string, history []api.Message, _ string) (string, error) {
			repairCaptured = api.CloneMessages(history)
			return validAddFilePatch, nil
		},
	}
	if _, err := agent.requestGeminiApplyPatchRepair(context.Background(), "bad patch", "parse error"); err != nil {
		t.Fatalf("requestGeminiApplyPatchRepair() error = %v", err)
	}
	assertLastProviderHistoryProjectionReportPreserved(t, agent.Runtime, reportAfterRequest)
	if providerHistoryMessagesContainReductionPlaceholder(repairCaptured) {
		t.Fatalf("internal repair request history contains provider reduction placeholder: %#v", repairCaptured)
	}

	out.Reset()
	_ = renderProviderHistoryStatusCommand(t, agent, &out)
	if !reflect.DeepEqual(agent.History, beforeHistory) {
		t.Fatalf("Agent.History changed after /status or internal call:\n got %#v\nwant %#v", agent.History, beforeHistory)
	}
	if agent.session.Messages[1].Content != oldRead {
		t.Fatalf("session conversation tool content = %q, want raw old read", agent.session.Messages[1].Content)
	}
	out.Reset()
	if !handleTokensCommand(agent) {
		t.Fatal("handleTokensCommand() = false, want true")
	}
	for _, reject := range []string{"Provider history reduction", "content_replacements", "total_provider_facing_saved", "approx_total_provider_facing_saved_tokens", "responses_chain_disabled"} {
		if strings.Contains(out.String(), reject) {
			t.Fatalf("/tokens output should not contain %q:\n%s", reject, out.String())
		}
	}
}

func TestPhase5DRawStorageStaysRawAcrossRequestSurfaces(t *testing.T) {
	t.Run("normal request preserves raw runtime and session audit", func(t *testing.T) {
		disableColors(t)
		var out bytes.Buffer
		provider := &providerFacingHistoryMutationProbe{}
		agent := newChatRequestTestAgent(t, provider, &out)
		oldRead := seedProviderHistoryReductionRequestFixture(t, agent, "call_raw_normal")
		agent.session = history.NewSession(agent.CurrentModel)
		agent.session.AddMessageFromAPI(agent.History[0], agent.CurrentModel)
		agent.session.AddMessageFromAPI(agent.History[1], agent.CurrentModel)
		agent.session.AddToolExecution("read_file", map[string]string{"path": "README.md"}, oldRead, true, agent.CurrentModel)

		if err := agent.chatInternal("next request", nil); err != nil {
			t.Fatalf("chatInternal() error = %v", err)
		}

		assertProviderRequestHistoryReductionApplied(t, agent, provider, oldRead, "next request")
		if agent.session.Messages[1].Content != oldRead {
			t.Fatalf("session raw tool content = %q, want raw old read", agent.session.Messages[1].Content)
		}
		assertProviderHistoryToolExecutionPreviewPreservesRaw(t, agent.session.Messages[2].ToolExecution, "read_file", oldRead)
	})

	t.Run("image headless and plan preserve raw runtime history", func(t *testing.T) {
		surfaces := []struct {
			name string
			run  func(t *testing.T) (*Agent, *providerFacingHistoryMutationProbe, string)
		}{
			{
				name: "image",
				run: func(t *testing.T) (*Agent, *providerFacingHistoryMutationProbe, string) {
					disableColors(t)
					var out bytes.Buffer
					provider := &providerFacingHistoryMutationProbe{supportsImages: true}
					agent := newChatRequestTestAgent(t, provider, &out)
					oldRead := seedProviderHistoryReductionRequestFixture(t, agent, "call_raw_image")
					image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png", Path: "test.png", Size: 4}
					if err := agent.chatInternal("describe image", image); err != nil {
						t.Fatalf("chatInternal(image) error = %v", err)
					}
					return agent, provider, oldRead
				},
			},
			{
				name: "headless",
				run: func(t *testing.T) (*Agent, *providerFacingHistoryMutationProbe, string) {
					t.Setenv("HOME", t.TempDir())
					provider := &providerFacingHistoryMutationProbe{}
					runner := newHeadlessRunner("headless query", "test-model", provider, newProjectMapDisabledConfig())
					t.Cleanup(runner.agent.Cleanup)
					oldRead := seedProviderHistoryReductionRequestFixture(t, runner.agent, "call_raw_headless")
					if _, err := runner.requestAssistantResponse(context.Background(), 0); err != nil {
						t.Fatalf("requestAssistantResponse() error = %v", err)
					}
					return runner.agent, provider, oldRead
				},
			},
			{
				name: "plan",
				run: func(t *testing.T) (*Agent, *providerFacingHistoryMutationProbe, string) {
					disableColors(t)
					var out bytes.Buffer
					provider := &providerFacingHistoryMutationProbe{response: "investigation done"}
					agent := newChatRequestTestAgent(t, provider, &out)
					oldRead := seedProviderHistoryReductionRequestFixture(t, agent, "call_raw_plan")
					if _, err := newPlanInvestigationRunner(agent, context.Background()).requestResponse(); err != nil {
						t.Fatalf("plan requestResponse() error = %v", err)
					}
					return agent, provider, oldRead
				},
			},
		}
		for _, surface := range surfaces {
			t.Run(surface.name, func(t *testing.T) {
				agent, provider, oldRead := surface.run(t)

				if !providerHistoryMessagesContainReductionPlaceholder(provider.capturedHistory) {
					t.Fatalf("%s provider history missing reduction placeholder: %#v", surface.name, provider.capturedHistory)
				}
				if agent.History[1].Content != oldRead {
					t.Fatalf("%s Agent.History[1].Content = %q, want raw old read", surface.name, agent.History[1].Content)
				}
				if agent.Runtime.LastProviderHistoryProjectionReport.ReplacedCount != 1 {
					t.Fatalf("%s LastProviderHistoryProjectionReport = %#v, want one replacement", surface.name, agent.Runtime.LastProviderHistoryProjectionReport)
				}
			})
		}
	})
}
