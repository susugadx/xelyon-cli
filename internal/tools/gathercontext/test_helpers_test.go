package gathercontext

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func withGatherContextWorkingDir(t *testing.T, dir string) {
	t.Helper()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})
}

type gatherContextExecCtxOption func(*tools.ExecutionContext)

func newGatherContextExecCtx(root string, opts ...gatherContextExecCtxOption) tools.ExecutionContext {
	execCtx := tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    locator.NewRegistry(),
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}
	for _, opt := range opts {
		opt(&execCtx)
	}
	return execCtx
}

func withGatherContextInvocationCWD(dir string) gatherContextExecCtxOption {
	return func(execCtx *tools.ExecutionContext) {
		execCtx.InvocationCWD = dir
	}
}

func withGatherContextLocatorRegistry(reg *locator.Registry) gatherContextExecCtxOption {
	return func(execCtx *tools.ExecutionContext) {
		execCtx.LocatorRegistry = reg
	}
}

func withGatherContextModel(provider, model string) gatherContextExecCtxOption {
	return func(execCtx *tools.ExecutionContext) {
		execCtx.ProviderName = provider
		execCtx.Model = model
	}
}

func writeGatherContextFiles(t *testing.T, files map[string]string) {
	t.Helper()
	for path, body := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func runGatherContext(t *testing.T, execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange) {
	t.Helper()
	result, change, err := (&Tool{}).Run(execCtx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return result, change
}

func assertGatherContextContainsAll(t *testing.T, result string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(result, want) {
			t.Fatalf("expected %q in result, got:\n%s", want, result)
		}
	}
}

func assertGatherContextExcludesAll(t *testing.T, result string, excludes ...string) {
	t.Helper()
	for _, exclude := range excludes {
		if strings.Contains(result, exclude) {
			t.Fatalf("did not expect %q in result, got:\n%s", exclude, result)
		}
	}
}
