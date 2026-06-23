package history

import "strings"

const responseContextMetadataVersion = 2

func (s *Session) ClearResponseContext() {
	if s == nil {
		return
	}

	s.ResponseID = ""
	s.ResponseModel = ""
	s.ResponseProviderName = ""
	s.ResponseProviderConfigKey = ""
	s.ResponsePromptFingerprint = ""
}

func (s *Session) ApplyResponseContext(responseID, responseModel, responseProviderName, responseProviderConfigKey string) {
	s.ApplyResponseContextWithFingerprint(responseID, responseModel, responseProviderName, responseProviderConfigKey, "")
}

// ApplyResponseContextWithFingerprint は Responses API 継続 state と prompt fingerprint を保存する。
func (s *Session) ApplyResponseContextWithFingerprint(responseID, responseModel, responseProviderName, responseProviderConfigKey, responsePromptFingerprint string) {
	if s == nil {
		return
	}

	id := strings.TrimSpace(responseID)
	if id == "" {
		s.ClearResponseContext()
		return
	}

	s.ResponseID = id
	s.ResponseModel = strings.TrimSpace(responseModel)
	if s.ResponseModel == "" {
		s.ResponseModel = s.Model
	}

	s.ResponseProviderName = strings.TrimSpace(responseProviderName)
	if s.ResponseProviderName == "" {
		s.ResponseProviderName = s.ProviderName
	}

	s.ResponseProviderConfigKey = strings.TrimSpace(responseProviderConfigKey)
	if s.ResponseProviderConfigKey == "" {
		s.ResponseProviderConfigKey = s.ProviderConfigKey
	}

	s.ResponsePromptFingerprint = strings.TrimSpace(responsePromptFingerprint)
}

func clearSavedResponseContext(session *Session) {
	if session == nil {
		return
	}
	session.ClearResponseContext()
}

func responseContextMetadataVersionForSession(session *Session) int {
	if session == nil || session.ResponseID == "" {
		return 0
	}
	return responseContextMetadataVersion
}

func restoreLoadedResponseContext(meta *SessionMetadata, session *Session) {
	if session == nil || meta == nil || session.ResponseID == "" {
		return
	}

	if meta.ResponseContextVersion >= responseContextMetadataVersion {
		return
	}

	migrateLegacyOpenAIResponseContext(session)
}

func migrateLegacyOpenAIResponseContext(session *Session) {
	if session == nil || session.ResponseID == "" {
		return
	}

	if session.ResponseModel == "" {
		session.ResponseModel = session.Model
	}
	if session.ResponseProviderName == "" {
		session.ResponseProviderName = "openai"
	}
	if session.ResponseProviderConfigKey == "" && session.ProviderConfigKey != "" {
		session.ResponseProviderConfigKey = session.ProviderConfigKey
	}
}
