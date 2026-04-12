package history

import "testing"

func TestResponseContextMetadataVersionForSession(t *testing.T) {
	t.Run("session without response id does not write version", func(t *testing.T) {
		session := NewSession("test-model")
		if got := responseContextMetadataVersionForSession(session); got != 0 {
			t.Fatalf("responseContextMetadataVersionForSession() = %d, want 0", got)
		}
	})

	t.Run("session with response id writes current version", func(t *testing.T) {
		session := NewSession("test-model")
		session.ResponseID = "resp_123"
		if got := responseContextMetadataVersionForSession(session); got != responseContextMetadataVersion {
			t.Fatalf("responseContextMetadataVersionForSession() = %d, want %d", got, responseContextMetadataVersion)
		}
	})
}

func TestRestoreLoadedResponseContext(t *testing.T) {
	t.Run("legacy metadata migrates openai response context", func(t *testing.T) {
		meta := &SessionMetadata{ResponseID: "resp_legacy"}
		session := &Session{
			Model:             "saved-model",
			ProviderName:      "openai",
			ProviderConfigKey: "openai",
			ResponseID:        "resp_legacy",
		}

		restoreLoadedResponseContext(meta, session)

		if session.ResponseModel != "saved-model" {
			t.Fatalf("session.ResponseModel = %q, want %q", session.ResponseModel, "saved-model")
		}
		if session.ResponseProviderName != "openai" || session.ResponseProviderConfigKey != "openai" {
			t.Fatalf(
				"session response provider identity = (%q, %q), want (%q, %q)",
				session.ResponseProviderName,
				session.ResponseProviderConfigKey,
				"openai",
				"openai",
			)
		}
	})

	t.Run("versioned metadata does not guess missing provider identity", func(t *testing.T) {
		meta := &SessionMetadata{
			ResponseID:             "resp_versioned",
			ResponseContextVersion: responseContextMetadataVersion,
		}
		session := &Session{
			Model:         "saved-model",
			ResponseID:    "resp_versioned",
			ResponseModel: "saved-model",
		}

		restoreLoadedResponseContext(meta, session)

		if session.ResponseProviderName != "" || session.ResponseProviderConfigKey != "" {
			t.Fatalf(
				"session response provider identity = (%q, %q), want left empty",
				session.ResponseProviderName,
				session.ResponseProviderConfigKey,
			)
		}
	})
}
