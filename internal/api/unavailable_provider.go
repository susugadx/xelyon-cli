package api

import (
	"context"
	"fmt"
	"strings"
)

// UnavailableProvider は provider setup が未完了でも interactive surface を起動するための placeholder provider。
type UnavailableProvider struct {
	providerName string
	message      string
}

// NewUnavailableProvider は provider setup 未完了を表す placeholder provider を返す。
func NewUnavailableProvider(providerName, message string) Provider {
	providerName = strings.TrimSpace(providerName)
	if providerName == "" {
		providerName = "unknown"
	}
	message = strings.TrimSpace(message)
	if message == "" {
		message = fmt.Sprintf("provider setup required for %s", providerName)
	}
	return &UnavailableProvider{
		providerName: providerName,
		message:      message,
	}
}

// IsProviderSetupRequired は provider が setup 未完了 placeholder か返す。
func IsProviderSetupRequired(provider Provider) bool {
	_, ok := provider.(*UnavailableProvider)
	return ok
}

// ProviderSetupRequiredMessage は setup 未完了 placeholder の user-facing message を返す。
func ProviderSetupRequiredMessage(provider Provider) (string, bool) {
	unavailable, ok := provider.(*UnavailableProvider)
	if !ok || unavailable == nil {
		return "", false
	}
	return unavailable.message, true
}

// Name は provider key を返す。
func (p *UnavailableProvider) Name() string {
	return p.providerName
}

// RuntimeProviderName は実行時 provider key を返す。
func (p *UnavailableProvider) RuntimeProviderName() string {
	return p.providerName
}

// ProviderConfigKey は provider_models/session owner key を返す。
func (p *UnavailableProvider) ProviderConfigKey() string {
	return p.providerName
}

// ChatWithTools は setup 未完了エラーを返す。
func (p *UnavailableProvider) ChatWithTools(context.Context, string, []Message, string) (string, error) {
	return "", p.setupError()
}

// SupportsImages は setup 未完了時に画像送信へ進ませないため false を返す。
func (p *UnavailableProvider) SupportsImages() bool {
	return false
}

// ChatWithImage は setup 未完了エラーを返す。
func (p *UnavailableProvider) ChatWithImage(context.Context, string, []Message, string, *ImageData, string) (string, error) {
	return "", p.setupError()
}

// IsFunctionCallingEnabled は placeholder では tool schema を有効にしない。
func (p *UnavailableProvider) IsFunctionCallingEnabled() bool {
	return false
}

func (p *UnavailableProvider) setupError() error {
	return fmt.Errorf("provider setup required: %s", p.message)
}
