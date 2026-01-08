package api

import (
	"context"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// Client はLLM APIクライアント
type Client struct {
	Provider Provider
	Timeout  time.Duration
}

// NewClient は新しいClientを作成
func NewClient(provider Provider) *Client {
	return &Client{
		Provider: provider,
		Timeout:  config.DefaultHTTPTimeout,
	}
}

// ChatWithTools はProviderに委譲
func (c *Client) ChatWithTools(systemPrompt string, history []Message, model string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	return c.Provider.ChatWithTools(ctx, systemPrompt, history, model)
}
