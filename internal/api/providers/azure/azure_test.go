package azure

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestChatWithTools_UsesAzureResponsesAPI(t *testing.T) {
	var received struct {
		path          string
		apiKey        string
		authorization string
		body          map[string]any
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.path = r.URL.Path
		received.apiKey = r.Header.Get("api-key")
		received.authorization = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&received.body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"type":"response.created","response":{"id":"resp_azure"}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"type":"response.output_text.delta","delta":"ok"}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"type":"response.completed","response":{"usage":{"input_tokens":7,"output_tokens":3}}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: [DONE]`)
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	p := New("azure-key")

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-gpt55-deployment",
		CatalogModel: "gpt-5.5",
	})
	ctx := azureTestContext(cfg)

	var gotUsage api.Usage
	p.SetUsageCallback(func(u api.Usage) {
		gotUsage = u
	})

	content, err := p.ChatWithTools(ctx, "system", []api.Message{{Role: "user", Content: "hello"}}, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if content != "ok" {
		t.Fatalf("content = %q, want ok", content)
	}
	if got := p.GetResponseID(); got != "resp_azure" {
		t.Fatalf("GetResponseID() = %q, want resp_azure", got)
	}

	if received.path != "/openai/v1/responses" {
		t.Fatalf("path = %q, want /openai/v1/responses", received.path)
	}
	if received.apiKey != "azure-key" {
		t.Fatalf("api-key = %q, want azure-key", received.apiKey)
	}
	if received.authorization != "" {
		t.Fatalf("Authorization = %q, want empty", received.authorization)
	}
	if received.body["model"] != "corp-gpt55-deployment" {
		t.Fatalf("model = %v, want deployment name", received.body["model"])
	}
	if got := int(received.body["max_output_tokens"].(float64)); got != 128000 {
		t.Fatalf("max_output_tokens = %d, want catalog_model gpt-5.5 limit", got)
	}
	if received.body["stream"] != true {
		t.Fatalf("stream = %v, want true", received.body["stream"])
	}
	if _, ok := received.body["prompt_cache_key"]; ok {
		t.Fatalf("prompt_cache_key should be omitted for Azure request: %#v", received.body)
	}
	if gotUsage.InputTokens != 7 || gotUsage.OutputTokens != 3 {
		t.Fatalf("usage = %+v, want input=7 output=3", gotUsage)
	}
}

func TestChatWithTools_UsesBearerTokenWhenAPIKeyMissing(t *testing.T) {
	t.Setenv(authTokenEnv, "entra-token")
	t.Setenv(authTokenCommandEnv, "")
	var received struct {
		apiKey        string
		authorization string
		body          map[string]any
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.apiKey = r.Header.Get("api-key")
		received.authorization = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&received.body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"type":"response.created","response":{"id":"resp_entra"}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"type":"response.output_text.delta","delta":"ok"}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"type":"response.completed","response":{"usage":{"input_tokens":4,"output_tokens":2}}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: [DONE]`)
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	p := New("")

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-gpt55-deployment",
		CatalogModel: "gpt-5.5",
	})

	content, err := p.ChatWithTools(azureTestContext(cfg), "system", []api.Message{{Role: "user", Content: "hello"}}, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if content != "ok" {
		t.Fatalf("content = %q, want ok", content)
	}
	if received.apiKey != "" {
		t.Fatalf("api-key = %q, want empty for Entra token auth", received.apiKey)
	}
	if received.authorization != "Bearer entra-token" {
		t.Fatalf("Authorization = %q, want Bearer entra-token", received.authorization)
	}
}

func TestChatWithTools_UsesTokenCommandWhenTokenMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("token command test uses POSIX shell")
	}
	if testing.Short() {
		t.Skip("token command test executes a local shell script")
	}

	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"type":"response.created","response":{"id":"resp_entra_command"}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"type":"response.output_text.delta","delta":"ok"}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: [DONE]`)
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	t.Setenv(authTokenEnv, "")
	t.Setenv(authTokenCommandEnv, "printf command-token")
	t.Setenv(authTokenCommandTimeoutEnv, "")

	p := New("")
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-gpt55-deployment",
		CatalogModel: "gpt-5.5",
	})

	content, err := p.ChatWithTools(azureTestContext(cfg), "system", []api.Message{{Role: "user", Content: "hello"}}, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if content != "ok" {
		t.Fatalf("content = %q, want ok", content)
	}
	if authorization != "Bearer command-token" {
		t.Fatalf("Authorization = %q, want Bearer command-token", authorization)
	}
}

func TestChatWithTools_RefreshesTokenCommandOnceAfterUnauthorized(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("token command test uses POSIX shell")
	}
	if testing.Short() {
		t.Skip("token command test executes a local shell script")
	}

	var authorizations []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorizations = append(authorizations, r.Header.Get("Authorization"))
		if len(authorizations) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"expired token"}}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"type":"response.created","response":{"id":"resp_refreshed"}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"type":"response.output_text.delta","delta":"ok"}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: [DONE]`)
	}))
	defer server.Close()

	dir := t.TempDir()
	counterPath := filepath.Join(dir, "counter")
	scriptPath := filepath.Join(dir, "token.sh")
	script := fmt.Sprintf(`#!/bin/sh
if [ ! -f %q ]; then
  echo 1 > %q
  printf expired-token
else
  printf refreshed-token
fi
`, counterPath, counterPath)
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write token script: %v", err)
	}

	t.Setenv(baseURLEnv, server.URL)
	t.Setenv(authTokenEnv, "")
	t.Setenv(authTokenCommandEnv, scriptPath)
	t.Setenv(authTokenCommandTimeoutEnv, "")

	p := New("")
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-gpt55-deployment",
		CatalogModel: "gpt-5.5",
	})

	content, err := p.ChatWithTools(azureTestContext(cfg), "system", []api.Message{{Role: "user", Content: "hello"}}, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if content != "ok" {
		t.Fatalf("content = %q, want ok", content)
	}
	if len(authorizations) != 2 {
		t.Fatalf("request count = %d, want 2", len(authorizations))
	}
	if authorizations[0] != "Bearer expired-token" || authorizations[1] != "Bearer refreshed-token" {
		t.Fatalf("authorizations = %#v, want expired then refreshed tokens", authorizations)
	}
}

func TestChatWithTools_UsesNonStreamingForAzureProCatalogModel(t *testing.T) {
	var received struct {
		path string
		body map[string]any
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.path = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&received.body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if received.body["stream"] == true {
			t.Fatalf("stream = true, want false or omitted for gpt-5.5-pro catalog model")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_pro","output_text":"Pro response","usage":{"input_tokens":11,"output_tokens":5}}`))
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	p := New("azure-key")

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-gpt55-pro-deployment",
		CatalogModel: "gpt-5.5-pro",
	})

	var gotUsage api.Usage
	p.SetUsageCallback(func(u api.Usage) {
		gotUsage = u
	})

	content, err := p.ChatWithTools(azureTestContext(cfg), "system", []api.Message{{Role: "user", Content: "hello"}}, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if content != "Pro response" {
		t.Fatalf("content = %q, want Pro response", content)
	}
	if got := p.GetResponseID(); got != "resp_pro" {
		t.Fatalf("GetResponseID() = %q, want resp_pro", got)
	}
	if received.path != "/openai/v1/responses" {
		t.Fatalf("path = %q, want /openai/v1/responses", received.path)
	}
	if received.body["model"] != "corp-gpt55-pro-deployment" {
		t.Fatalf("model = %v, want Azure deployment name", received.body["model"])
	}
	if gotUsage.InputTokens != 11 || gotUsage.OutputTokens != 5 {
		t.Fatalf("usage = %+v, want input=11 output=5", gotUsage)
	}
}

func TestChatWithTools_StoreFalseOmitsPreviousAndDoesNotCacheResponseID(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if _, ok := received["previous_response_id"]; ok {
			t.Fatalf("previous_response_id should be omitted when responses.store=false: %#v", received)
		}
		if received["store"] != false {
			t.Fatalf("store = %#v, want false", received["store"])
		}
		input, ok := received["input"].([]any)
		if !ok || len(input) != 3 {
			t.Fatalf("input = %#v, want developer + compacted + current user", received["input"])
		}
		compacted, ok := input[1].(map[string]any)
		if !ok || compacted["type"] != "compacted" || compacted["data"] != "compact-data" {
			t.Fatalf("input[1] = %#v, want compacted item", input[1])
		}

		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"type":"response.created","response":{"id":"resp_new"}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"type":"response.output_text.delta","delta":"ok"}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: [DONE]`)
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	p := New("azure-key")
	p.SetResponseID("resp_old")

	cfg := config.DefaultConfig()
	cfg.Responses.Store = false
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-gpt55-deployment",
		CatalogModel: "gpt-5.5",
	})
	ctx := azureTestContext(cfg)
	ctx = api.WithCompactedInputItems(ctx, []api.InputItem{{Type: "compacted", Data: "compact-data"}})

	content, err := p.ChatWithTools(ctx, "system", []api.Message{{Role: "user", Content: "hello"}}, "")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if content != "ok" {
		t.Fatalf("content = %q, want ok", content)
	}
	if got := p.GetResponseID(); got != "" {
		t.Fatalf("GetResponseID() = %q, want empty when responses.store=false", got)
	}
}

func TestAzureProvider_ResponseIDContract(t *testing.T) {
	p := New("azure-key")
	if p.Name() != "Azure OpenAI" {
		t.Fatalf("Name() = %q, want Azure OpenAI", p.Name())
	}
	if p.RuntimeProviderName() != "azure" {
		t.Fatalf("RuntimeProviderName() = %q, want azure", p.RuntimeProviderName())
	}
	if p.ProviderConfigKey() != "azure" {
		t.Fatalf("ProviderConfigKey() = %q, want azure", p.ProviderConfigKey())
	}
	if p.HasCachedResponseID() {
		t.Fatal("HasCachedResponseID() = true, want false")
	}

	p.SetResponseID(" resp_azure ")
	if got := p.GetResponseID(); got != "resp_azure" {
		t.Fatalf("GetResponseID() = %q, want trimmed response ID", got)
	}
	if !p.HasCachedResponseID() {
		t.Fatal("HasCachedResponseID() = false, want true")
	}

	p.ClearCache()
	if got := p.GetResponseID(); got != "" {
		t.Fatalf("GetResponseID() after ClearCache = %q, want empty", got)
	}
}

func TestBuildChatResponsesRequest_OmitsToolsWhenFunctionCallingDisabled(t *testing.T) {
	t.Setenv("AZURE_OPENAI_FUNCTION_CALLING", "0")

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-gpt55-deployment",
		CatalogModel: "gpt-5.5",
	})

	p := New("azure-key")
	p.SetMCPTools([]api.ToolDefinition{{
		Name:        "extra_tool",
		Description: "extra",
	}})
	p.SetToolChoice("extra_tool")

	req := p.buildChatResponsesRequest(
		azureTestContext(cfg),
		"system",
		[]api.Message{{Role: "user", Content: "hello"}},
		"corp-gpt55-deployment",
	)

	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if _, ok := body["tools"]; ok {
		t.Fatalf("tools should be omitted when function calling is disabled: %s", payload)
	}
	if _, ok := body["tool_choice"]; ok {
		t.Fatalf("tool_choice should be omitted when function calling is disabled: %s", payload)
	}
}

func TestBuildChatResponsesRequest_OmitsToolsWhenToolUseDisabled(t *testing.T) {
	t.Setenv("AZURE_OPENAI_FUNCTION_CALLING", "1")

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-gpt55-deployment",
		CatalogModel: "gpt-5.5",
	})

	p := New("azure-key")
	p.SetMCPTools([]api.ToolDefinition{{Name: "extra_tool", Description: "extra"}})
	p.SetToolChoice("extra_tool")

	ctx := api.WithToolUseDisabled(azureTestContext(cfg))
	req := p.buildChatResponsesRequest(
		ctx,
		"system",
		[]api.Message{{Role: "user", Content: "hello"}},
		"corp-gpt55-deployment",
	)

	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if _, ok := body["tools"]; ok {
		t.Fatalf("tools should be omitted when tool use is disabled: %s", payload)
	}
	if _, ok := body["tool_choice"]; ok {
		t.Fatalf("tool_choice should be omitted when tool use is disabled: %s", payload)
	}
}

func TestBuildChatResponsesRequest_StoreFalseSendsFullHistoryWithoutPreviousResponseID(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Responses.Store = false
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-gpt55-deployment",
		CatalogModel: "gpt-5.5",
	})

	p := New("azure-key")
	p.SetResponseID("resp_old")
	ctx := azureTestContext(cfg)
	ctx = api.WithCompactedInputItems(ctx, []api.InputItem{{Type: "compacted", Data: "compact-data"}})
	req := p.buildChatResponsesRequest(
		ctx,
		"system",
		[]api.Message{
			{Role: "user", Content: "first"},
			{Role: "assistant", Content: "answer"},
			{Role: "user", Content: "next"},
		},
		"corp-gpt55-deployment",
	)

	if req.Store {
		t.Fatal("Store = true, want false")
	}
	if req.PreviousResponseID != "" {
		t.Fatalf("PreviousResponseID = %q, want empty", req.PreviousResponseID)
	}
	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	input, ok := body["input"].([]any)
	if !ok {
		t.Fatalf("input type = %T, want []any", body["input"])
	}
	if len(input) != 5 {
		t.Fatalf("input length = %d, want developer plus compacted item plus full history", len(input))
	}
	compacted, ok := input[1].(map[string]any)
	if !ok {
		t.Fatalf("input[1] type = %T, want compacted map", input[1])
	}
	if compacted["type"] != "compacted" || compacted["data"] != "compact-data" {
		t.Fatalf("input[1] = %#v, want compacted item from context", compacted)
	}
	if _, ok := body["previous_response_id"]; ok {
		t.Fatalf("previous_response_id should be omitted: %s", payload)
	}
}

func TestBuildChatResponsesRequest_IncludesServerCompactionContextManagementOnPreviousResponseChain(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-gpt55-deployment",
		CatalogModel: "gpt-5.5",
	})
	p := New("azure-key")
	p.SetResponseID("resp_old")

	req := p.buildChatResponsesRequest(
		azureTestContext(cfg),
		"system",
		[]api.Message{{Role: "user", Content: "hello"}},
		"corp-gpt55-deployment",
	)

	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	contextManagementRaw, ok := body["context_management"].([]any)
	if !ok || len(contextManagementRaw) != 1 {
		t.Fatalf("context_management = %#v, want one compaction item", body["context_management"])
	}
	compaction, ok := contextManagementRaw[0].(map[string]any)
	if !ok {
		t.Fatalf("context_management[0] type = %T, want map", contextManagementRaw[0])
	}
	if compaction["type"] != "compaction" {
		t.Fatalf("context_management[0].type = %#v, want compaction", compaction["type"])
	}
	threshold, ok := compaction["compact_threshold"].(float64)
	if !ok {
		t.Fatalf("compact_threshold type = %T, want float64(JSON number)", compaction["compact_threshold"])
	}
	if int(threshold) < 1000 {
		t.Fatalf("compact_threshold = %d, want >= 1000", int(threshold))
	}
	if int(threshold) == 0 {
		t.Fatal("compact_threshold = 0, want resolved non-zero value")
	}
}

func TestBuildChatResponsesRequest_OmitsServerCompactionWhenDisabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Responses.ServerCompaction.Enabled = false
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-gpt55-deployment",
		CatalogModel: "gpt-5.5",
	})
	p := New("azure-key")
	p.SetResponseID("resp_old")

	req := p.buildChatResponsesRequest(
		azureTestContext(cfg),
		"system",
		[]api.Message{{Role: "user", Content: "hello"}},
		"corp-gpt55-deployment",
	)

	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if _, ok := body["context_management"]; ok {
		t.Fatalf("context_management should be omitted when disabled: %#v", body["context_management"])
	}
}

func TestBuildChatResponsesRequest_OmitsServerCompactionWhenContextWindowUnknown(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-unknown-deployment",
	})
	p := New("azure-key")
	p.SetResponseID("resp_old")

	req := p.buildChatResponsesRequest(
		azureTestContext(cfg),
		"system",
		[]api.Message{{Role: "user", Content: "hello"}},
		"corp-unknown-deployment",
	)

	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if _, ok := body["context_management"]; ok {
		t.Fatalf("context_management should be omitted when context window is unknown: %#v", body["context_management"])
	}
}

func TestChatWithTools_ServerCompactionPayloadControlsLocalSkipState(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"type":"response.created","response":{"id":"resp_azure"}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"type":"response.output_text.delta","delta":"ok"}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: [DONE]`)
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-gpt55-deployment",
		CatalogModel: "gpt-5.5",
	})

	p := New("azure-key")
	p.SetResponseID("resp_old")
	content, err := p.ChatWithTools(azureTestContext(cfg), "system", []api.Message{{Role: "user", Content: "hello"}}, "corp-gpt55-deployment")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if content != "ok" {
		t.Fatalf("content = %q, want ok", content)
	}
	if !p.ShouldSkipLocalAutoCompressionForServerCompaction() {
		t.Fatal("ShouldSkipLocalAutoCompressionForServerCompaction() = false, want true when context_management.compaction is sent")
	}
	if _, ok := received["context_management"]; !ok {
		t.Fatalf("context_management should be present: %#v", received)
	}
}

func TestChatWithTools_UnknownContextOmitsCompactionAndKeepsLocalFallbackState(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"type":"response.created","response":{"id":"resp_azure"}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"type":"response.output_text.delta","delta":"ok"}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: [DONE]`)
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-unknown-deployment",
	})

	p := New("azure-key")
	p.SetResponseID("resp_old")
	content, err := p.ChatWithTools(azureTestContext(cfg), "system", []api.Message{{Role: "user", Content: "hello"}}, "corp-unknown-deployment")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if content != "ok" {
		t.Fatalf("content = %q, want ok", content)
	}
	if p.ShouldSkipLocalAutoCompressionForServerCompaction() {
		t.Fatal("ShouldSkipLocalAutoCompressionForServerCompaction() = true, want false when compaction payload is omitted")
	}
	if _, ok := received["context_management"]; ok {
		t.Fatalf("context_management should be omitted: %#v", received["context_management"])
	}
}

func TestChatWithTools_UnknownContextOmitsCompactionAndSkipsLocalFallbackWhenDisabled(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"type":"response.created","response":{"id":"resp_azure"}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"type":"response.output_text.delta","delta":"ok"}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: [DONE]`)
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	cfg := config.DefaultConfig()
	cfg.Responses.ServerCompaction.LocalFallback = false
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-unknown-deployment",
	})

	p := New("azure-key")
	p.SetResponseID("resp_old")
	content, err := p.ChatWithTools(azureTestContext(cfg), "system", []api.Message{{Role: "user", Content: "hello"}}, "corp-unknown-deployment")
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if content != "ok" {
		t.Fatalf("content = %q, want ok", content)
	}
	if !p.ShouldSkipLocalAutoCompressionForServerCompaction() {
		t.Fatal("ShouldSkipLocalAutoCompressionForServerCompaction() = false, want true when local fallback is disabled")
	}
	if _, ok := received["context_management"]; ok {
		t.Fatalf("context_management should be omitted: %#v", received["context_management"])
	}
}

func TestBuildImageResponsesRequest_UsesInputImage(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "vision-deployment",
		CatalogModel: "gpt-5.4",
	})

	req := New("azure-key").buildImageResponsesRequest(
		azureTestContext(cfg),
		"system",
		nil,
		"what is this?",
		&api.ImageData{MediaType: "image/png", Base64: "abc123"},
		"vision-deployment",
	)

	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	if !strings.Contains(string(payload), `"type":"input_image"`) {
		t.Fatalf("payload = %s, want input_image", payload)
	}
	if !strings.Contains(string(payload), `data:image/png;base64,abc123`) {
		t.Fatalf("payload = %s, want image data URL", payload)
	}
}

func TestNewLongRunningResponsesHTTPClient_DisablesResponseHeaderTimeout(t *testing.T) {
	baseTransport := http.DefaultTransport.(*http.Transport).Clone()
	baseTransport.ResponseHeaderTimeout = 2 * time.Second
	base := &http.Client{Transport: baseTransport}

	client := newLongRunningResponsesHTTPClient(base)
	if client == base {
		t.Fatal("newLongRunningResponsesHTTPClient() should clone client instance")
	}

	gotTransport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport type = %T, want *http.Transport", client.Transport)
	}
	if gotTransport.ResponseHeaderTimeout != 0 {
		t.Fatalf("ResponseHeaderTimeout = %s, want 0", gotTransport.ResponseHeaderTimeout)
	}
	if baseTransport.ResponseHeaderTimeout != 2*time.Second {
		t.Fatalf("base transport mutated: ResponseHeaderTimeout = %s, want 2s", baseTransport.ResponseHeaderTimeout)
	}
}
