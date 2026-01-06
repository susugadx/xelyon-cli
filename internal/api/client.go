package api

import (
	"context"
	"time"
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
		Timeout:  30 * time.Second,
	}
}

// ChatWithTools はProviderに委譲
func (c *Client) ChatWithTools(systemPrompt string, history []Message, model string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	return c.Provider.ChatWithTools(ctx, systemPrompt, history, model)
}
