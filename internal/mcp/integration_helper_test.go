package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpHelperEchoInput struct {
	Name string `json:"name"`
}

func TestMCPHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_XELYON_MCP_HELPER") != "1" {
		return
	}
	if err := runMCPHelperServer(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func runMCPHelperServer() error {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "xelyon-mcp-helper",
		Version: "test",
	}, nil)
	addMCPHelperFailureMiddleware(server)

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "echo",
		Description: "Return a stable greeting for integration tests",
	}, func(_ context.Context, _ *sdkmcp.CallToolRequest, input mcpHelperEchoInput) (*sdkmcp.CallToolResult, any, error) {
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{
				&sdkmcp.TextContent{Text: "Hello " + input.Name},
				&sdkmcp.TextContent{Text: "From helper"},
			},
		}, nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "hidden",
		Description: "Filtered-out helper tool",
	}, func(_ context.Context, _ *sdkmcp.CallToolRequest, _ map[string]any) (*sdkmcp.CallToolResult, any, error) {
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{
				&sdkmcp.TextContent{Text: "hidden"},
			},
		}, nil, nil
	})

	return server.Run(context.Background(), &sdkmcp.StdioTransport{})
}

func addMCPHelperFailureMiddleware(server *sdkmcp.Server) {
	failMethod := os.Getenv("GO_WANT_XELYON_MCP_HELPER_FAIL")
	if failMethod == "" {
		return
	}
	server.AddReceivingMiddleware(func(next sdkmcp.MethodHandler) sdkmcp.MethodHandler {
		return func(ctx context.Context, method string, req sdkmcp.Request) (sdkmcp.Result, error) {
			if method == failMethod {
				return nil, fmt.Errorf("helper %s failed with arg %s env %s", method, mcpHelperSecretArg(), os.Getenv("MCP_SECRET_ENV"))
			}
			return next(ctx, method, req)
		}
	})
}

func mcpHelperSecretArg() string {
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "--secret=") {
			return strings.TrimPrefix(arg, "--secret=")
		}
	}
	return ""
}
