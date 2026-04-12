package history

const responseContextMetadataVersion = 1

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
	if session.ResponseProviderConfigKey == "" {
		if session.ProviderConfigKey != "" {
			session.ResponseProviderConfigKey = session.ProviderConfigKey
		} else {
			session.ResponseProviderConfigKey = "openai"
		}
	}
}
