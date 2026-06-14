package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
)

var beforeStartNewSessionMetadataSaveForTest func()

func (a *Agent) restoreSessionConversation(session *history.Session) {
	if a == nil {
		return
	}

	a.resetProviderFacingTaskLedger()
	a.session = session
	a.lastOutputs = nil

	if session == nil {
		a.History = nil
		a.RestoreCompactedState(nil)
		a.restoreProviderResponseID("")
		return
	}

	a.History = session.ToAPIMessages()
	a.RestoreCompactedState(session)
	a.restoreProviderResponseID(session.ResponseID)
}

func (a *Agent) resetConversationState() error {
	if a == nil {
		return nil
	}

	a.History = nil
	a.lastOutputs = nil
	a.RestoreCompactedState(nil)
	a.restoreProviderResponseID("")
	a.resetProviderFacingTaskLedger()

	if a.session == nil {
		return nil
	}

	a.session.ResetConversation()
	if a.storage != nil {
		return a.storage.Rewrite(a.session)
	}
	return nil
}

func (a *Agent) syncRuntimeIdentityToSession(session *history.Session) {
	if a == nil || session == nil {
		return
	}

	session.Model = a.CurrentModel
	session.ProviderName = config.CanonicalProviderName(a.ProviderName)
	session.ProviderConfigKey = a.currentProviderConfigKey()
}

// StartNewSession は現在の session を保存してから、新しい session ID の会話を開始する。
func (a *Agent) StartNewSession() (*history.Session, error) {
	if a == nil {
		return nil, nil
	}
	if a.storage != nil && a.session != nil {
		a.syncSessionPersistenceState()
		if err := a.storage.Save(a.session); err != nil {
			return nil, fmt.Errorf("save current session: %w", err)
		}
	}

	session := history.NewSession(a.CurrentModel)
	a.syncRuntimeIdentityToSession(session)
	if a.storage != nil {
		if beforeStartNewSessionMetadataSaveForTest != nil {
			beforeStartNewSessionMetadataSaveForTest()
		}
		if err := a.storage.Save(session); err != nil {
			return nil, fmt.Errorf("save new session metadata: %w", err)
		}
	}

	a.restoreSessionConversation(session)
	if a.Stats != nil {
		a.statsMu.Lock()
		a.Stats = NewSessionStats(a.ProviderName, a.CurrentModel)
		a.statsMu.Unlock()
	}
	a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or / for commands", "リクエスト、または / でコマンド候補を入力")
	return session, nil
}

// ResumeSession は保存済み session を読み込み、保存時の provider/model へ runtime を合わせて再開する。
func (a *Agent) ResumeSession(sessionID string) (*history.Session, error) {
	if a == nil {
		return nil, fmt.Errorf("agent is nil")
	}
	if a.storage == nil {
		return nil, fmt.Errorf("history storage not available")
	}
	if a.session != nil {
		a.syncSessionPersistenceState()
		if err := a.storage.Save(a.session); err != nil {
			return nil, fmt.Errorf("save current session: %w", err)
		}
	}

	session, err := a.storage.Load(strings.TrimSpace(sessionID))
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}
	if err := a.switchRuntimeForLoadedSessionWithActiveSessionDetached(session); err != nil {
		return nil, err
	}
	a.applyLoadedSession(session)
	a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or / for commands", "リクエスト、または / でコマンド候補を入力")
	return session, nil
}

// ResumeStartupSession は interactive 起動直後の bootstrap session を保存せずに保存済み session を再開する。
func (a *Agent) ResumeStartupSession(sessionID string) (*history.Session, error) {
	if a == nil {
		return nil, fmt.Errorf("agent is nil")
	}
	if a.storage == nil {
		return nil, fmt.Errorf("history storage not available")
	}

	a.restoreSessionConversation(nil)
	session, err := a.storage.Load(strings.TrimSpace(sessionID))
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}
	if err := a.switchRuntimeForLoadedSessionWithActiveSessionDetached(session); err != nil {
		return nil, err
	}
	a.applyLoadedSession(session)
	a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or / for commands", "リクエスト、または / でコマンド候補を入力")
	return session, nil
}

// ResumeStartupLastSession は startup の bootstrap session を保存せず resume scope 内の最新 session を再開する。
func (a *Agent) ResumeStartupLastSession(opts history.ResumeListOptions) (*history.Session, error) {
	if a == nil || a.storage == nil {
		return nil, fmt.Errorf("history storage not available")
	}
	sessionID, err := a.storage.GetLastResumeSession(opts)
	if err != nil {
		return nil, err
	}
	return a.ResumeStartupSession(sessionID)
}

// ResumeLastSession は resume scope 内の最新 session を再開する。
func (a *Agent) ResumeLastSession(opts history.ResumeListOptions) (*history.Session, error) {
	if a == nil || a.storage == nil {
		return nil, fmt.Errorf("history storage not available")
	}
	sessionID, err := a.storage.GetLastResumeSession(opts)
	if err != nil {
		return nil, err
	}
	return a.ResumeSession(sessionID)
}

// ResumeSessionCandidates は resume picker 用の候補を返す。
func (a *Agent) ResumeSessionCandidates(opts history.ResumeListOptions) ([]history.SessionMetadata, error) {
	if a == nil || a.storage == nil {
		return nil, fmt.Errorf("history storage not available")
	}
	return a.storage.ListResumeSessions(opts)
}

func (a *Agent) switchRuntimeForLoadedSessionWithActiveSessionDetached(session *history.Session) error {
	if a == nil {
		return nil
	}

	activeSession := a.session
	a.session = nil
	defer func() {
		a.session = activeSession
	}()
	return a.switchRuntimeForLoadedSession(session)
}

func (a *Agent) switchRuntimeForLoadedSession(session *history.Session) error {
	if a == nil || session == nil {
		return nil
	}
	providerKey := strings.TrimSpace(session.ProviderConfigKey)
	if providerKey == "" {
		providerKey = strings.TrimSpace(session.ProviderName)
	}
	if providerKey == "" {
		return nil
	}
	model := strings.TrimSpace(session.Model)
	currentKey := config.ActiveProviderConfigKey(a.currentProviderConfigKey())
	targetKey := config.ActiveProviderConfigKey(providerKey)
	if targetKey != "" && currentKey != "" && targetKey == currentKey {
		if model != "" && model != a.CurrentModel {
			a.applyRuntimeModelSelection(model, shouldResetResponseContinuationForModelSwitch(a.CurrentModel, model))
		}
		return nil
	}
	outcome, err := a.SwitchProviderModel(providerKey, model)
	if err != nil {
		return fmt.Errorf("switch to session provider/model: %w", err)
	}
	if outcome.NewModel == "" && model != "" {
		return fmt.Errorf("switch to session provider/model: resolved empty model")
	}
	return nil
}

func (a *Agent) restoreProviderResponseID(responseID string) {
	if a == nil {
		return
	}
	if !a.responsesPersistResponseIDEnabled() {
		responseID = ""
	}
	if ridProvider, ok := a.CurrentProvider.(ResponseIDCapable); ok {
		ridProvider.SetResponseID(strings.TrimSpace(responseID))
	}
}

func (a *Agent) clearResponseContinuationContext() {
	if a == nil {
		return
	}

	a.restoreProviderResponseID("")
	if a.session == nil {
		return
	}

	clearSavedResponseContext(a.session)
	a.persistSession()
}

func (a *Agent) persistLocalCompressionSuccess(messages []api.Message) {
	if a == nil {
		return
	}

	a.restoreProviderResponseID("")
	if a.session == nil {
		return
	}

	a.session.ReplaceMessagesFromAPI(messages, a.CurrentModel)
	a.session.SetCompactedState(convertToHistoryCompactedItems(a.compactedItems), a.isCompactedMode)
	clearSavedResponseContext(a.session)
	a.rewriteSessionWithWarning("⚠️  Warning: Failed to save compressed session: %v\n")
}

func (a *Agent) suspendResponseContinuationForLocalCompression(persistOnSuccess bool) func(success bool, messages []api.Message) {
	if a == nil {
		return func(bool, []api.Message) {}
	}

	var ridProvider ResponseIDCapable
	previousResponseID := ""
	if provider, ok := a.CurrentProvider.(ResponseIDCapable); ok {
		ridProvider = provider
		previousResponseID = provider.GetResponseID()
		provider.SetResponseID("")
	}

	return func(success bool, messages []api.Message) {
		if success {
			if persistOnSuccess {
				a.persistLocalCompressionSuccess(messages)
				return
			}
			a.restoreProviderResponseID("")
			return
		}
		if ridProvider != nil {
			ridProvider.SetResponseID(previousResponseID)
		}
	}
}

func (a *Agent) hasLocalConversationContext() bool {
	if a == nil {
		return false
	}

	if len(a.History) > 0 || len(a.lastOutputs) > 0 || a.isCompactedMode || len(a.compactedItems) > 0 {
		return true
	}
	if a.session == nil {
		return false
	}
	return len(a.session.Messages) > 0 ||
		len(a.session.CompactedItems) > 0 ||
		a.session.IsCompactedMode
}
