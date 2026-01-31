package testutil

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// Compile-time interface verification
var _ api.Provider = (*MockProvider)(nil)

// MockProvider implements api.Provider for testing purposes.
// It provides configurable behavior through function fields and tracks
// method calls for test assertions.
type MockProvider struct {
	// NameFunc allows customizing the Name() return value
	NameFunc func() string

	// SupportsImagesFunc allows customizing the SupportsImages() return value
	SupportsImagesFunc func() bool

	// ChatWithToolsFunc allows customizing the ChatWithTools() behavior
	ChatWithToolsFunc func(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error)

	// ChatWithImageFunc allows customizing the ChatWithImage() behavior
	ChatWithImageFunc func(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error)

	// CallCount tracks how many times each method was called
	CallCount struct {
		Name           int
		SupportsImages int
		ChatWithTools  int
		ChatWithImage  int
	}

	// LastCall stores the arguments of the last call to each method
	LastCall struct {
		ChatWithTools struct {
			SystemPrompt string
			History      []api.Message
			Model        string
		}
		ChatWithImage struct {
			SystemPrompt string
			History      []api.Message
			UserMessage  string
			Image        *api.ImageData
			Model        string
		}
	}
}

// Name returns the provider name
func (m *MockProvider) Name() string {
	m.CallCount.Name++
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return "MockProvider"
}

// SupportsImages returns whether the provider supports images
func (m *MockProvider) SupportsImages() bool {
	m.CallCount.SupportsImages++
	if m.SupportsImagesFunc != nil {
		return m.SupportsImagesFunc()
	}
	return false
}

// IsFunctionCallingEnabled returns whether function calling is enabled
func (m *MockProvider) IsFunctionCallingEnabled() bool {
	return true
}

// ChatWithTools sends a chat request with tool support
func (m *MockProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	m.CallCount.ChatWithTools++
	m.LastCall.ChatWithTools.SystemPrompt = systemPrompt
	m.LastCall.ChatWithTools.History = history
	m.LastCall.ChatWithTools.Model = model

	if m.ChatWithToolsFunc != nil {
		return m.ChatWithToolsFunc(ctx, systemPrompt, history, model)
	}
	return "mock response", nil
}

// ChatWithImage sends a chat request with an image
func (m *MockProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	m.CallCount.ChatWithImage++
	m.LastCall.ChatWithImage.SystemPrompt = systemPrompt
	m.LastCall.ChatWithImage.History = history
	m.LastCall.ChatWithImage.UserMessage = userMessage
	m.LastCall.ChatWithImage.Image = image
	m.LastCall.ChatWithImage.Model = model

	if m.ChatWithImageFunc != nil {
		return m.ChatWithImageFunc(ctx, systemPrompt, history, userMessage, image, model)
	}
	return "mock image response", nil
}

// NewMockProvider creates a new MockProvider with default settings
func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

// NewMockProviderWithName creates a new MockProvider with a custom name
func NewMockProviderWithName(name string) *MockProvider {
	return &MockProvider{
		NameFunc: func() string { return name },
	}
}

// NewMockProviderWithResponse creates a new MockProvider that returns a fixed response
func NewMockProviderWithResponse(response string) *MockProvider {
	return &MockProvider{
		ChatWithToolsFunc: func(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			return response, nil
		},
	}
}

// NewMockProviderWithError creates a new MockProvider that always returns an error
func NewMockProviderWithError(err error) *MockProvider {
	return &MockProvider{
		ChatWithToolsFunc: func(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			return "", err
		},
		ChatWithImageFunc: func(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
			return "", err
		},
	}
}

// Reset clears all call counts and last call data
func (m *MockProvider) Reset() {
	m.CallCount.Name = 0
	m.CallCount.SupportsImages = 0
	m.CallCount.ChatWithTools = 0
	m.CallCount.ChatWithImage = 0
	m.LastCall.ChatWithTools.SystemPrompt = ""
	m.LastCall.ChatWithTools.History = nil
	m.LastCall.ChatWithTools.Model = ""
	m.LastCall.ChatWithImage.SystemPrompt = ""
	m.LastCall.ChatWithImage.History = nil
	m.LastCall.ChatWithImage.UserMessage = ""
	m.LastCall.ChatWithImage.Image = nil
	m.LastCall.ChatWithImage.Model = ""
}
