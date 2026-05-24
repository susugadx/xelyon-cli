package llmcatalog

import "strings"

// GeminiFunctionCallingPolicy は Gemini の request model と catalog_model から
// function calling のローカル対応状況を解決した結果を表す。
type GeminiFunctionCallingPolicy struct {
	requestModel string
	policyModel  string
	support      ModelCapabilitySupport
}

// NewGeminiFunctionCallingPolicy は Gemini function calling 判定に使う policy を返す。
// catalogModel が空の場合は requestModel を policy lookup 表示名として使う。
func NewGeminiFunctionCallingPolicy(requestModel, catalogModel string) GeminiFunctionCallingPolicy {
	policyModel := CanonicalModelNameForProvider("gemini", catalogModel)
	if policyModel == "" {
		policyModel = strings.TrimSpace(requestModel)
	}
	return GeminiFunctionCallingPolicy{
		requestModel: strings.TrimSpace(requestModel),
		policyModel:  policyModel,
		support:      GeminiFunctionCallingSupport(policyModel),
	}
}

// RequestModel は実 Gemini request に送る model 名を返す。
func (p GeminiFunctionCallingPolicy) RequestModel() string {
	return p.requestModel
}

// PolicyModel は function calling 判定に使った catalog/policy model 名を返す。
func (p GeminiFunctionCallingPolicy) PolicyModel() string {
	return p.policyModel
}

// Support は function calling のローカル対応状況を返す。
func (p GeminiFunctionCallingPolicy) Support() ModelCapabilitySupport {
	return p.support
}

// Enabled は runtime が function calling payload を送ってよいか返す。
// 未知 model は custom alias 互換のため optimistic に許可する。
func (p GeminiFunctionCallingPolicy) Enabled() bool {
	return !p.support.Known || p.support.Supported
}
