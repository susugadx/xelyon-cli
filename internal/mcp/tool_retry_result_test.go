package mcp

import (
	"context"
	"errors"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"testing"
)

func TestShouldRetryToolCall(t *testing.T) {
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name    string
		ctx     context.Context
		attempt int
		err     error
		result  *sdkmcp.CallToolResult
		want    bool
	}{
		{name: "temporary transport error retries", ctx: context.Background(), attempt: 1, err: errors.New("temporary"), want: true},
		{name: "context canceled error does not retry", ctx: context.Background(), attempt: 1, err: context.Canceled, want: false},
		{name: "context deadline error does not retry", ctx: context.Background(), attempt: 1, err: context.DeadlineExceeded, want: false},
		{name: "canceled request context does not retry", ctx: cancelledCtx, attempt: 1, err: errors.New("temporary"), want: false},
		{name: "tool error result retries", ctx: context.Background(), attempt: 1, result: &sdkmcp.CallToolResult{IsError: true}, want: true},
		{name: "max attempts does not retry", ctx: context.Background(), attempt: toolCallMaxAttempts, err: errors.New("temporary"), want: false},
		{name: "nil result without error does not retry", ctx: context.Background(), attempt: 1, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRetryToolCall(tt.ctx, tt.attempt, tt.err, tt.result); got != tt.want {
				t.Fatalf("shouldRetryToolCall() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToolResultTextAndErrorMessageFallbacks(t *testing.T) {
	if got := toolResultText(nil); got != "" {
		t.Fatalf("toolResultText(nil) = %q, want empty", got)
	}

	nonTextError := &sdkmcp.CallToolResult{
		IsError: true,
		Content: []sdkmcp.Content{
			&sdkmcp.ImageContent{MIMEType: "image/png", Data: []byte("base64")},
		},
	}
	if got := toolResultErrorMessage(nonTextError); got != "tool returned error" {
		t.Fatalf("toolResultErrorMessage(non-text) = %q, want fallback", got)
	}
	if got := toolResultText(nonTextError); got != "" {
		t.Fatalf("toolResultText(non-text) = %q, want empty", got)
	}

	textResult := &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{
			&sdkmcp.TextContent{Text: "first"},
			&sdkmcp.ImageContent{MIMEType: "image/png", Data: []byte("base64")},
			&sdkmcp.TextContent{Text: "second"},
		},
	}
	if got := toolResultText(textResult); got != "first\nsecond\n" {
		t.Fatalf("toolResultText(text) = %q, want text content joined with newlines", got)
	}

	textError := &sdkmcp.CallToolResult{
		IsError: true,
		Content: []sdkmcp.Content{
			&sdkmcp.TextContent{Text: "permission denied"},
		},
	}
	if got := toolResultErrorMessage(textError); got != "permission denied" {
		t.Fatalf("toolResultErrorMessage(text) = %q, want text error", got)
	}
}
