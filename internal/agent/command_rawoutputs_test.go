package agent

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/providerhistory"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func TestRawOutputsCommandSummaryIsReadOnlyForMissingRoot(t *testing.T) {
	t.Setenv(config.ProviderHistoryRawOutputArtifactRootEnvVar, "")
	t.Setenv("XELYON_ENCRYPT_HISTORY", "0")
	root := filepath.Join(t.TempDir(), "missing", "rawoutputs")
	agent, out := newRawOutputsCommandTestAgent(root)

	if !handleSpecialCommandForSurface("/rawoutputs summary", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/rawoutputs summary was not handled")
	}
	output := out.String()
	assertRawOutputsOutputContains(t, output, "Raw output artifacts", "store_exists: false", "refs: 0")
	assertRawOutputsOutputOmits(t, output, "Failed to")
	if pathExists(root) {
		t.Fatalf("/rawoutputs summary created root %s", root)
	}
}

func TestRawOutputsCommandSummaryWithEncryptedHistoryMissingKeyIsReadOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(config.ProviderHistoryRawOutputArtifactRootEnvVar, "")
	t.Setenv("XELYON_ENCRYPT_HISTORY", "1")
	root := filepath.Join(t.TempDir(), "missing", "rawoutputs")
	agent, out := newRawOutputsCommandTestAgent(root)

	if !handleSpecialCommandForSurface("/rawoutputs summary", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/rawoutputs summary was not handled")
	}
	output := out.String()
	assertRawOutputsOutputContains(t, output, "Raw output artifacts", "store_exists: false")
	assertRawOutputsOutputOmits(t, output, "Failed to")
	if pathExists(root) {
		t.Fatalf("/rawoutputs summary created root %s", root)
	}
	if pathExists(filepath.Join(home, ".xelyon")) {
		t.Fatalf("/rawoutputs summary created history key directory under %s", home)
	}
}

func TestRawOutputsCommandRefsDoesNotRenderRawBody(t *testing.T) {
	t.Setenv(config.ProviderHistoryRawOutputArtifactRootEnvVar, "")
	t.Setenv("XELYON_ENCRYPT_HISTORY", "0")
	root := filepath.Join(t.TempDir(), "rawoutputs")
	agent, out := newRawOutputsCommandTestAgent(root)
	body := "RAW_SECRET_LOOKING_BODY_BUT_NOT_CLASSIFIED_TOKEN\n"
	result := createRawOutputsCommandArtifact(t, root, agent.session.ID, "call-rawoutputs", body)
	agent.Runtime.LastProviderHistoryProjectionReport = ProviderHistoryProjectionReport{
		RawOutputRefs: []rawoutputs.RawOutputRef{result.Ref},
	}

	if !handleSpecialCommandForSurface("/rawoutputs refs", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/rawoutputs refs was not handled")
	}
	output := out.String()
	assertRawOutputsOutputContains(t, output, "ref_details:", result.Ref.RefID, "live: live", "sha256:")
	assertRawOutputsOutputOmits(t, output, body, "RAW_SECRET_LOOKING_BODY")
}

func TestRawOutputsCommandRefsIncludesContextOnlyRawOutputRefsAsLive(t *testing.T) {
	t.Setenv(config.ProviderHistoryRawOutputArtifactRootEnvVar, "")
	t.Setenv("XELYON_ENCRYPT_HISTORY", "0")
	root := filepath.Join(t.TempDir(), "rawoutputs")
	agent, out := newRawOutputsCommandTestAgent(root)
	normal := createRawOutputsCommandArtifact(t, root, agent.session.ID, "call-normal", "normal live body\n")
	contextOnly := createRawOutputsCommandMCPArtifact(t, root, agent.session.ID, "call-context", "context-only live body\n")
	stale := createRawOutputsCommandArtifact(t, root, agent.session.ID, "call-stale", "stale body\n")
	agent.Runtime.LastProviderHistoryProjectionReport = ProviderHistoryProjectionReport{
		RawOutputRefs:                 []rawoutputs.RawOutputRef{normal.Ref},
		RawOutputContextRefs:          []rawoutputs.RawOutputRef{contextOnly.Ref},
		RawOutputContextMissingRefIDs: []string{"rawout_missing"},
	}

	if !handleSpecialCommandForSurface("/rawoutputs refs", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/rawoutputs refs was not handled")
	}
	output := out.String()
	assertRawOutputsOutputContains(t, output, "live_ref_sources: 2")
	assertRawOutputsRefLineContains(t, output, normal.Ref.RefID, "live: live")
	assertRawOutputsRefLineContains(t, output, contextOnly.Ref.RefID, "live: live")
	assertRawOutputsRefLineContains(t, output, stale.Ref.RefID, "live: not_live")
	assertRawOutputsOutputOmits(t, output, "rawout_missing")
}

func TestRawOutputsCommandGCDryRunUsesCallerLiveRefs(t *testing.T) {
	t.Setenv(config.ProviderHistoryRawOutputArtifactRootEnvVar, "")
	t.Setenv("XELYON_ENCRYPT_HISTORY", "0")
	root := filepath.Join(t.TempDir(), "rawoutputs")
	agent, out := newRawOutputsCommandTestAgent(root)
	result := createRawOutputsCommandArtifact(t, root, agent.session.ID, "call-live", "live body\n")
	_ = createRawOutputsCommandArtifact(t, root, agent.session.ID, "call-dead", "dead body\n")
	agent.Runtime.LastProviderHistoryProjectionReport = ProviderHistoryProjectionReport{
		RawOutputRefs: []rawoutputs.RawOutputRef{result.Ref},
	}

	if !handleSpecialCommandForSurface("/rawoutputs gc --dry-run", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/rawoutputs gc --dry-run was not handled")
	}
	output := out.String()
	assertRawOutputsOutputContains(t, output, "gc_dry_run:", "tombstone_refs: 1", "collect_artifacts: 1", "kept_artifacts: 1")
}

func TestRawOutputsCommandGCDryRunKeepsContextOnlyRawOutputRefs(t *testing.T) {
	t.Setenv(config.ProviderHistoryRawOutputArtifactRootEnvVar, "")
	t.Setenv("XELYON_ENCRYPT_HISTORY", "0")
	root := filepath.Join(t.TempDir(), "rawoutputs")
	agent, out := newRawOutputsCommandTestAgent(root)
	normal := createRawOutputsCommandArtifact(t, root, agent.session.ID, "call-live", "normal live body\n")
	contextOnly := createRawOutputsCommandMCPArtifact(t, root, agent.session.ID, "call-context", "context-only live body\n")
	_ = createRawOutputsCommandArtifact(t, root, agent.session.ID, "call-dead", "dead body\n")
	agent.Runtime.LastProviderHistoryProjectionReport = ProviderHistoryProjectionReport{
		RawOutputRefs:                 []rawoutputs.RawOutputRef{normal.Ref},
		RawOutputContextRefs:          []rawoutputs.RawOutputRef{contextOnly.Ref},
		RawOutputContextMissingRefIDs: []string{"rawout_missing"},
	}

	if !handleSpecialCommandForSurface("/rawoutputs gc --dry-run", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/rawoutputs gc --dry-run was not handled")
	}
	output := out.String()
	assertRawOutputsOutputContains(t, output, "live_ref_sources: 2", "gc_dry_run:", "tombstone_refs: 1", "collect_artifacts: 1", "kept_artifacts: 2")
	assertRawOutputsOutputOmits(t, output, "rawout_missing")
}

func TestRawOutputsCommandsKeepRuntimeMCPRefLiveWhenActiveContextUnavailable(t *testing.T) {
	t.Setenv(config.ProviderHistoryRawOutputArtifactRootEnvVar, "")
	t.Setenv("XELYON_ENCRYPT_HISTORY", "0")
	root := filepath.Join(t.TempDir(), "rawoutputs")
	agent, out := newRawOutputsCommandTestAgent(root)
	contextOnly := createRawOutputsCommandMCPArtifact(t, root, agent.session.ID, "call-context-unavailable", "context-only live body\n")
	stale := createRawOutputsCommandArtifact(t, root, agent.session.ID, "call-stale", "stale body\n")
	store, err := rawoutputs.OpenStore(rawoutputs.Root(root), rawoutputs.StoreOptions{})
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	placeholder := buildMCPRuntimeResultPlaceholder(contextOnly.Ref, "", "context-only live body\n")
	history := []api.Message{
		providerHistoryAssistantToolCall("call-context-unavailable", "mcp_context7_get_library_docs"),
		providerHistoryToolResult("call-context-unavailable", "mcp_context7_get_library_docs", placeholder),
	}
	projection := providerhistory.Project(providerhistory.ProjectionInput{
		Messages: history,
		Policy: providerhistory.Policy{
			Mode:                             providerhistory.Apply,
			RawOutputArtifactsMode:           providerhistory.RawOutputArtifactsApply,
			RawOutputArtifactStore:           store,
			SessionID:                        agent.session.ID,
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  false,
		},
	})
	agent.Runtime.LastProviderHistoryProjectionReport = projection.Report

	if !handleSpecialCommandForSurface("/rawoutputs refs", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/rawoutputs refs was not handled")
	}
	output := out.String()
	assertRawOutputsOutputContains(t, output, "live_ref_sources: 1")
	assertRawOutputsRefLineContains(t, output, contextOnly.Ref.RefID, "live: live")
	assertRawOutputsRefLineContains(t, output, stale.Ref.RefID, "live: not_live")

	out.Reset()
	if !handleSpecialCommandForSurface("/rawoutputs gc --dry-run", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/rawoutputs gc --dry-run was not handled")
	}
	output = out.String()
	assertRawOutputsOutputContains(t, output, "gc_dry_run:", "collect_artifacts: 1", "kept_artifacts: 1")
	assertRawOutputsOutputOmits(t, output, stale.Ref.RefID)
}

func TestRawOutputsCommandRejectsMutatingSubcommands(t *testing.T) {
	t.Setenv(config.ProviderHistoryRawOutputArtifactRootEnvVar, "")
	t.Setenv("XELYON_ENCRYPT_HISTORY", "0")
	agent, out := newRawOutputsCommandTestAgent(filepath.Join(t.TempDir(), "rawoutputs"))

	for _, input := range []string{"/rawoutputs gc --apply", "/rawoutputs delete", "/rawoutputs repair"} {
		out.Reset()
		if !handleSpecialCommandForSurface(input, agent, commandcatalog.CommandSurfaceClassic) {
			t.Fatalf("%s was not handled", input)
		}
		assertRawOutputsOutputContains(t, out.String(), "not supported", "Usage: /rawoutputs")
	}
}

func newRawOutputsCommandTestAgent(root string) (*Agent, *bytes.Buffer) {
	out := &bytes.Buffer{}
	session := history.NewSession("gpt-test")
	session.ID = "session-rawoutputs"
	cfg := config.DefaultProviderHistoryRawOutputArtifactsConfig()
	cfg.Root = root
	return &Agent{
		Runtime: &AgentRuntime{
			Options: RuntimeOptions{
				ProviderHistoryRawOutputArtifacts: cfg,
			},
			UI: uiruntime.NewRuntime(strings.NewReader(""), out, out),
		},
		agentConversationState: agentConversationState{
			session: session,
		},
	}, out
}

func createRawOutputsCommandArtifact(t *testing.T, root, sessionID, callID, body string) rawoutputs.CreateResult {
	t.Helper()
	return createRawOutputsCommandArtifactForSurface(t, root, sessionID, callID, rawoutputs.SurfaceCommandOutput, "bash", "network", "network_response", body)
}

func createRawOutputsCommandMCPArtifact(t *testing.T, root, sessionID, callID, body string) rawoutputs.CreateResult {
	t.Helper()
	return createRawOutputsCommandArtifactForSurface(t, root, sessionID, callID, rawoutputs.SurfaceMCPToolResult, "mcp_context7_get_library_docs", "mcp", "mcp_runtime_large_result", body)
}

func createRawOutputsCommandArtifactForSurface(t *testing.T, root, sessionID, callID string, surface rawoutputs.Surface, toolName, family, classifier, body string) rawoutputs.CreateResult {
	t.Helper()
	store, err := rawoutputs.OpenStore(rawoutputs.Root(root), rawoutputs.StoreOptions{})
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	result, err := store.Create(context.Background(), rawoutputs.CreateRequest{
		Surface:   surface,
		SessionID: sessionID,
		Source: rawoutputs.SourceMetadata{
			ToolName:     toolName,
			ToolCallID:   callID,
			CommandHash:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			HistoryIndex: 1,
		},
		Classification: rawoutputs.ClassificationMetadata{
			SemanticRole: "data_bearing",
			Family:       family,
			Classifier:   classifier,
		},
		Body:          strings.NewReader(body),
		SizeHintBytes: int64(len(body)),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return result
}

func assertRawOutputsOutputContains(t *testing.T, output string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(output, fragment) {
			t.Fatalf("/rawoutputs output missing %q:\n%s", fragment, output)
		}
	}
}

func assertRawOutputsOutputOmits(t *testing.T, output string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if strings.Contains(output, fragment) {
			t.Fatalf("/rawoutputs output should not contain %q:\n%s", fragment, output)
		}
	}
}

func assertRawOutputsRefLineContains(t *testing.T, output, refID string, fragments ...string) {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(line, refID) {
			continue
		}
		for _, fragment := range fragments {
			if !strings.Contains(line, fragment) {
				t.Fatalf("/rawoutputs line for %s missing %q:\n%s", refID, fragment, line)
			}
		}
		return
	}
	t.Fatalf("/rawoutputs output missing ref %s:\n%s", refID, output)
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
