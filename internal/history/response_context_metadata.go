package history

import "strings"

const responseContextMetadataVersion = 2

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

	if meta.ResponseContextVersion == 0 {
		migrateLegacyOpenAIResponseContext(session)
		return
	}

	repairVersion1GuessedOpenAIResponseContext(session)
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

func repairVersion1GuessedOpenAIResponseContext(session *Session) {
	if session == nil || session.ResponseID == "" {
		return
	}

	if !strings.EqualFold(session.ResponseProviderName, "openai") {
		return
	}
	if !strings.EqualFold(session.ResponseProviderConfigKey, "openai") {
		return
	}

	currentProviderConfigKey := strings.TrimSpace(session.ProviderConfigKey)
	if currentProviderConfigKey == "" || strings.EqualFold(currentProviderConfigKey, "openai") {
		return
	}

	session.ResponseProviderConfigKey = ""
}
