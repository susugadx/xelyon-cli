package tools

import (
	"bytes"
	"context"
	"io"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/locator"
	lsplib "github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestExecutionContext_PromptIOUsesInjectedRuntime(t *testing.T) {
	in := strings.NewReader("hello\n")
	var out bytes.Buffer
	var errOut bytes.Buffer
	runtime := ui.NewRuntime(in, &out, &errOut)

	promptIO := ExecutionContext{
		Runtime: runtime,
		Stdin:   in,
		Stdout:  &out,
		Stderr:  &errOut,
	}.PromptIO()
	line, err := promptIO.ReadSimpleLine()
	if err != nil {
		t.Fatalf("PromptIO().ReadSimpleLine() error = %v", err)
	}
	if line != "hello" {
		t.Fatalf("PromptIO().ReadSimpleLine() = %q, want %q", line, "hello")
	}
	if promptIO.Out != &out || promptIO.Err != &errOut {
		t.Fatal("PromptIO() should use runtime writers")
	}
}

func TestExecutionContext_ContextAndLocatorAccessors(t *testing.T) {
	lspClient := &lsplib.Client{}
	locators := locator.NewRegistry()
	ctx := ExecutionContext{
		Context:         context.Background(),
		LSPClient:       lspClient,
		LocatorRegistry: locators,
	}

	if got := ctx.EffectiveLSPClient(); got != lspClient {
		t.Fatal("EffectiveLSPClient() should preserve injected client")
	}
	if got := ctx.EffectiveLocatorRegistry(); got != locators {
		t.Fatal("EffectiveLocatorRegistry() should preserve injected registry")
	}

	if defaultLocators := (ExecutionContext{}).EffectiveLocatorRegistry(); defaultLocators == nil {
		t.Fatal("default EffectiveLocatorRegistry() should be non-nil")
	}
}

func TestRegistryAndConfigContextHelpers(t *testing.T) {
	baseCtx := context.Background()
	var nilCtx context.Context
	registry := NewRegistry()
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "gemini"

	if got := WithRegistry(baseCtx, nil); got != baseCtx {
		t.Fatal("WithRegistry(nil) should return the original context")
	}
	if got := RegistryFromContext(nilCtx); got != DefaultRegistry {
		t.Fatal("RegistryFromContext(nil) should return DefaultRegistry")
	}

	withRegistry := WithRegistry(baseCtx, registry)
	if got := RegistryFromContext(withRegistry); got != registry {
		t.Fatal("RegistryFromContext() should return injected registry")
	}

	withConfig := WithConfig(baseCtx, cfg)
	if got := ConfigFromContext(withConfig); got != cfg {
		t.Fatal("ConfigFromContext() should return injected config")
	}
	if got := ConfigFromContext(context.Background()); got == nil {
		t.Fatal("ConfigFromContext(default) should not be nil")
	}
}

func TestSearchCacheLifecycleHooks_NotifiesPrimaryAndExtraHooks(t *testing.T) {
	searchCacheLifecycleHooks.mu.Lock()
	origClear := searchCacheLifecycleHooks.clear
	origInvalidate := searchCacheLifecycleHooks.invalidate
	origEvicted := searchCacheLifecycleHooks.evicted
	origExtraClear := append([]func(){}, searchCacheLifecycleHooks.extraClear...)
	origExtraInvalidate := append([]func([]string){}, searchCacheLifecycleHooks.extraInvalidate...)
	origExtraEvicted := append([]func([]string){}, searchCacheLifecycleHooks.extraEvicted...)
	searchCacheLifecycleHooks.mu.Unlock()
	t.Cleanup(func() {
		searchCacheLifecycleHooks.mu.Lock()
		searchCacheLifecycleHooks.clear = origClear
		searchCacheLifecycleHooks.invalidate = origInvalidate
		searchCacheLifecycleHooks.evicted = origEvicted
		searchCacheLifecycleHooks.extraClear = origExtraClear
		searchCacheLifecycleHooks.extraInvalidate = origExtraInvalidate
		searchCacheLifecycleHooks.extraEvicted = origExtraEvicted
		searchCacheLifecycleHooks.mu.Unlock()
	})

	var events []string
	RegisterSearchCacheLifecycleHooks(
		func() { events = append(events, "clear:primary") },
		func(keys []string) { events = append(events, "invalidate:"+strings.Join(keys, ",")) },
		func(keys []string) { events = append(events, "evict:"+strings.Join(keys, ",")) },
	)
	AddSearchCacheLifecycleHooks(
		func() { events = append(events, "clear:extra") },
		func(keys []string) { events = append(events, "invalidate-extra:"+strings.Join(keys, ",")) },
		nil,
	)
	AddSearchCacheLifecycleHooks(nil, nil, func(keys []string) {
		events = append(events, "evict-extra:"+strings.Join(keys, ","))
	})

	NotifySearchCacheCleared()
	NotifySearchCacheInvalidatedKeys([]string{"a", "b"})
	NotifySearchCacheEvicted([]string{"c"})

	want := []string{
		"clear:primary",
		"clear:extra",
		"invalidate:a,b",
		"invalidate-extra:a,b",
		"evict:c",
		"evict-extra:c",
	}
	if !slices.Equal(events, want) {
		t.Fatalf("hook events = %v, want %v", events, want)
	}
}

func TestRegistryCloneAndWrapperContracts(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&captureArgsTool{name: "simple"})
	registry.SetExcludedTools([]string{"hidden", "secret"})

	cloned := registry.Clone()
	if cloned == registry {
		t.Fatal("Clone() should return a distinct registry")
	}
	if !cloned.HasTool("simple") {
		t.Fatal("Clone() should preserve tools")
	}
	names := cloned.GetExcludedTools()
	sort.Strings(names)
	if strings.Join(names, ",") != "hidden,secret" {
		t.Fatalf("GetExcludedTools() = %v, want [hidden secret]", names)
	}

	simple := &SimpleTool{
		name: "echo",
		execute: func(args map[string]string) string {
			return "echo:" + args["value"]
		},
	}
	got, change, err := simple.Run(ExecutionContext{Stdout: io.Discard, Stderr: io.Discard}, map[string]string{"value": "ok"})
	if err != nil || change != nil || got != "echo:ok" {
		t.Fatalf("SimpleTool.Run() = (%q, %+v, %v), want (%q, nil, nil)", got, change, err, "echo:ok")
	}

	modifying := &FileModifyingTool{
		name: "writer",
		execute: func(args map[string]string) (string, error) {
			return "written", nil
		},
		description: func(args map[string]string) string {
			return "updated " + args["path"]
		},
		getFilePath: func(args map[string]string) string {
			return args["path"]
		},
	}
	got, change, err = modifying.Run(ExecutionContext{}, map[string]string{"path": "main.go"})
	if err != nil || got != "written" {
		t.Fatalf("FileModifyingTool.Run() = (%q, %v), want (%q, nil)", got, err, "written")
	}
	if change == nil || change.FilePath != "main.go" || change.Tool != "writer" || change.Description != "updated main.go" {
		t.Fatalf("FileModifyingTool.Run() change = %+v", change)
	}

	RegisterBuiltinTools(cloned)
}
