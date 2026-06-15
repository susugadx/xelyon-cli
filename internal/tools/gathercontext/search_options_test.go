package gathercontext

import (
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestBuildSearchOptions_AttachesLSPAdapterWithInvocationCWD(t *testing.T) {
	root := t.TempDir()
	invocationCWD := filepath.Join(root, "workspace")

	opts := buildSearchOptions(
		newGatherContextExecCtx(root, func(execCtx *tools.ExecutionContext) {
			execCtx.InvocationCWD = invocationCWD
			execCtx.LSPClient = &lsp.Client{}
		}),
		searchPlan{query: "Build", preferImpact: true},
	)

	adapter, ok := opts.LSPClient.(*navigation.LSPAdapter)
	if !ok {
		t.Fatalf("LSPClient = %T, want *navigation.LSPAdapter", opts.LSPClient)
	}
	if adapter.RootDir != invocationCWD {
		t.Fatalf("adapter.RootDir = %q, want invocation CWD %q", adapter.RootDir, invocationCWD)
	}
	if opts.Intent != "impact" {
		t.Fatalf("Intent = %q, want impact", opts.Intent)
	}
}
