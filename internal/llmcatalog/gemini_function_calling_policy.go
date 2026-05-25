package llmcatalog

import (
	"fmt"
	"strings"
)

// GeminiFunctionCallingPolicy は Gemini の request model と catalog_model から
// function calling のローカル対応状況を解決した結果を表す。
type GeminiFunctionCallingPolicy struct {
	requestModel   string
	catalogModel   string
	policyModel    string
	support        ModelCapabilitySupport
	requestSupport ModelCapabilitySupport
	catalogSupport ModelCapabilitySupport
}

// NewGeminiFunctionCallingPolicy は Gemini function calling 判定に使う policy を返す。
// catalogModel が空の場合は requestModel を policy lookup 表示名として使う。
func NewGeminiFunctionCallingPolicy(requestModel, catalogModel string) GeminiFunctionCallingPolicy {
	requestModel = strings.TrimSpace(requestModel)
	resolvedCatalogModel := CanonicalModelNameForProvider("gemini", catalogModel)
	if resolvedCatalogModel == "" {
		resolvedCatalogModel = requestModel
	}

	policyModel := resolvedCatalogModel
	catalogSupport := GeminiFunctionCallingSupport(policyModel)
	support := catalogSupport
	requestPolicyModel := CanonicalModelNameForProvider("gemini", requestModel)
	requestSupport := GeminiFunctionCallingSupport(requestPolicyModel)
	if requestSupport.Known && !requestSupport.Supported {
		policyModel = requestPolicyModel
		support = requestSupport
	}

	return GeminiFunctionCallingPolicy{
		requestModel:   requestModel,
		catalogModel:   resolvedCatalogModel,
		policyModel:    policyModel,
		support:        support,
		requestSupport: requestSupport,
		catalogSupport: catalogSupport,
	}
}

// RequestModel は実 Gemini request に送る model 名を返す。
func (p GeminiFunctionCallingPolicy) RequestModel() string {
	return p.requestModel
}

// CatalogModel は provider policy に使う catalog model 名を返す。
func (p GeminiFunctionCallingPolicy) CatalogModel() string {
	return p.catalogModel
}

// PolicyModel は function calling 判定に使った catalog/policy model 名を返す。
func (p GeminiFunctionCallingPolicy) PolicyModel() string {
	return p.policyModel
}

// Support は function calling のローカル対応状況を返す。
func (p GeminiFunctionCallingPolicy) Support() ModelCapabilitySupport {
	return p.support
}

// RequestSupport は request model 側の function calling 対応状況を返す。
func (p GeminiFunctionCallingPolicy) RequestSupport() ModelCapabilitySupport {
	return p.requestSupport
}

// CatalogSupport は catalog_model 側の function calling 対応状況を返す。
func (p GeminiFunctionCallingPolicy) CatalogSupport() ModelCapabilitySupport {
	return p.catalogSupport
}

// Enabled は runtime が function calling payload を送ってよいか返す。
// 未知 model は custom alias 互換のため optimistic に許可する。
func (p GeminiFunctionCallingPolicy) Enabled() bool {
	return !p.support.Known || p.support.Supported
}

// UnsupportedError は既知非対応 model の function calling エラーを返す。
func (p GeminiFunctionCallingPolicy) UnsupportedError() error {
	if p.Enabled() {
		return nil
	}
	modelDetail := p.requestModel
	if modelDetail == "" {
		modelDetail = p.policyModel
	}
	if p.catalogModel != "" && p.catalogModel != modelDetail {
		modelDetail = fmt.Sprintf("%s (catalog_model=%s)", modelDetail, p.catalogModel)
	}
	replacement := strings.TrimSpace(p.support.Replacement)
	if replacement == "" {
		replacement = "gemini-3.5-flash"
	}
	reason := strings.TrimSpace(p.support.Reason)
	if reason == "" {
		reason = "the selected model is not supported by XELYON's Gemini function-calling runtime"
	}
	return fmt.Errorf("Gemini function calling is not supported for %s: %s; use %s or disable tool use only for internal text-only requests", modelDetail, reason, replacement)
}
