package history

func (s *Session) unsavedMessages() []MessageEntry {
	if s == nil {
		return nil
	}
	if s.persistedCount < 0 {
		s.persistedCount = 0
	}
	if s.persistedCount >= len(s.Messages) {
		return nil
	}
	return s.Messages[s.persistedCount:]
}

func (s *Session) markPersisted() {
	if s == nil {
		return
	}
	s.persistedCount = len(s.Messages)
	s.rewriteRequired = false
}

func (s *Session) needsRewrite() bool {
	return s != nil && s.rewriteRequired
}

func (s *Session) requireRewrite() {
	if s == nil {
		return
	}
	s.rewriteRequired = true
}

func (s *Session) conversationMessageCount() int {
	count := 0
	for _, msg := range s.Messages {
		if !isConversationMessageEntry(msg) {
			continue
		}
		count++
	}
	return count
}

func (s *Session) compactedItemCount() int {
	if s == nil || !s.IsCompactedMode {
		return 0
	}
	return len(s.CompactedItems)
}

func (s *Session) storageEntries() []MessageEntry {
	if s == nil {
		return nil
	}

	entries := make([]MessageEntry, 0, len(s.Messages)+1)
	if s.IsCompactedMode && len(s.CompactedItems) > 0 {
		entries = append(entries, MessageEntry{
			Timestamp:       s.LastModified,
			EntryType:       compactedStateEntryType,
			CompactedItems:  cloneCompactedItems(s.CompactedItems),
			IsCompactedMode: true,
		})
	}
	entries = append(entries, s.Messages...)
	return entries
}

func isConversationMessageEntry(msg MessageEntry) bool {
	return msg.EntryType == ""
}
