package rawoutputs

import (
	"context"
	"os"
	"strings"
)

func (s *Store) create(ctx context.Context, req CreateRequest) (CreateResult, error) {
	if err := ctx.Err(); err != nil {
		return CreateResult{}, err
	}
	if err := s.validateCreateRequest(req); err != nil {
		return CreateResult{}, err
	}
	if req.Classification.Sensitive {
		return CreateResult{}, reasonError(ReasonSensitiveArtifactForbidden, "sensitive output is not stored in normal raw output artifact store")
	}
	if req.SizeHintBytes > s.opts.MaxArtifactBytes {
		return CreateResult{}, reasonError(ReasonArtifactTooLarge, "size hint %d > max %d", req.SizeHintBytes, s.opts.MaxArtifactBytes)
	}

	tempPath, stats, err := s.writeTempPreparedObject(ctx, req.SessionID, req.Body)
	if err != nil {
		return CreateResult{}, err
	}
	if tempPath != "" {
		defer func() {
			_ = os.Remove(tempPath)
		}()
	}
	if err := s.ensureSessionDirs(req.SessionID); err != nil {
		return CreateResult{}, err
	}
	if stats.byteSize > s.opts.MaxArtifactBytes {
		return CreateResult{}, reasonError(ReasonArtifactTooLarge, "artifact %d > max %d", stats.byteSize, s.opts.MaxArtifactBytes)
	}

	ref, artifact, record := s.createdRecordForRequest(req, stats)
	states, err := s.loadLifecycle(req.SessionID)
	if err != nil {
		return CreateResult{}, err
	}
	if existing, ok := states[ref.RefID]; ok && existing.created.RecordType != "" {
		if err := lifecycleUsable(existing); err != nil {
			return CreateResult{}, err
		}
		if err := validateRefMatchesRecord(ref, existing.created); err != nil {
			return CreateResult{}, err
		}
		if err := s.ensureArtifactObject(req.SessionID, tempPath, existing.created.Artifact); err != nil {
			return CreateResult{}, err
		}
		return CreateResult{Ref: existing.created.Ref, Artifact: existing.created.Artifact, Record: existing.created}, nil
	}

	additionalBytes := stats.byteSize
	if liveArtifactExists(states, artifact.ArtifactID) {
		additionalBytes = 0
	}
	used := sessionArtifactBytesFromLifecycle(states)
	if used+additionalBytes > s.opts.SessionQuotaBytes {
		return CreateResult{}, reasonError(ReasonSessionQuotaExceeded, "session quota exceeded: %d + %d > %d", used, additionalBytes, s.opts.SessionQuotaBytes)
	}

	if err := s.ensureArtifactObject(req.SessionID, tempPath, artifact); err != nil {
		return CreateResult{}, err
	}

	if err := s.appendManifestRecord(record); err != nil {
		return CreateResult{}, err
	}
	if _, err := s.rebuildSessionIndex(ctx, req.SessionID); err != nil {
		return CreateResult{}, err
	}
	return CreateResult{Ref: ref, Artifact: artifact, Record: record}, nil
}

func (s *Store) ensureArtifactObject(sessionID, tempPath string, artifact RawOutputArtifact) error {
	objectExists, err := s.artifactObjectExists(sessionID, artifact)
	if err != nil {
		return err
	}
	if objectExists {
		return nil
	}
	if strings.TrimSpace(tempPath) == "" {
		return reasonError(ReasonArtifactMissing, "prepared object is missing")
	}
	return s.commitPreparedObject(sessionID, tempPath, artifact)
}

func (s *Store) createdRecordForRequest(req CreateRequest, stats tempWriteStats) (RawOutputRef, RawOutputArtifact, ManifestRecord) {
	contentHash := "sha256:" + stats.sha256Hex
	artifact := RawOutputArtifact{
		ArtifactID:      contentHash,
		HashAlgorithm:   hashAlgorithmSHA256,
		ContentHash:     contentHash,
		ByteSize:        int(stats.byteSize),
		StorageEncoding: storageEncodingRaw,
		Encrypted:       false,
		RelativePath:    objectRelativePath(stats.sha256Hex),
	}
	if s.opts.EncryptionEnabled {
		artifact.StorageEncoding = storageEncodingEncStreamV2
		artifact.Encrypted = true
	}
	ref := RawOutputRef{
		RefID:          buildRefID(req, contentHash),
		Surface:        string(req.Surface),
		SessionID:      req.SessionID,
		EventID:        trimDisplay(req.Source.EventID, maxPreviewRunes),
		HistoryIndex:   req.Source.HistoryIndex,
		ToolName:       trimDisplay(req.Source.ToolName, maxPreviewRunes),
		ToolCallID:     trimDisplay(req.Source.ToolCallID, maxPreviewRunes),
		CommandHash:    trimDisplay(req.Source.CommandHash, maxPreviewRunes),
		CommandPreview: trimDisplay(req.Source.CommandPreview, maxPreviewRunes),
		ArtifactID:     artifact.ArtifactID,
		Family:         trimDisplay(req.Classification.Family, maxPreviewRunes),
		Subfamily:      trimDisplay(req.Classification.Subfamily, maxPreviewRunes),
		SemanticRole:   trimDisplay(req.Classification.SemanticRole, maxPreviewRunes),
		Classifier:     trimDisplay(req.Classification.Classifier, maxPreviewRunes),
		ContentHash:    contentHash,
		ByteSize:       int(stats.byteSize),
		RuneSize:       stats.runeSize,
		ApproxTokens:   approxTokensFromBytes(stats.byteSize),
	}
	retention := req.Retention
	if strings.TrimSpace(retention.Policy) == "" {
		retention.Policy = retentionSession
	}
	if retention.CreatedAt.IsZero() {
		retention.CreatedAt = s.opts.Now().UTC()
	}
	record := ManifestRecord{
		SchemaVersion:  schemaVersion,
		RecordType:     recordTypeCreated,
		Ref:            ref,
		Source:         sanitizeSourceMetadata(req.Source),
		Classification: sanitizeClassificationMetadata(req.Classification),
		Artifact:       artifact,
		Retention:      retention,
		CreatedAt:      s.opts.Now().UTC(),
	}
	return ref, artifact, record
}

func liveArtifactExists(states map[string]lifecycleState, artifactID string) bool {
	for _, state := range states {
		if state.created.RecordType == "" || state.created.Artifact.ArtifactID != artifactID {
			continue
		}
		if lifecycleUsable(state) == nil {
			return true
		}
	}
	return false
}

func (s *Store) artifactObjectExists(sessionID string, artifact RawOutputArtifact) (bool, error) {
	path, err := s.safeExistingSessionFilePath(sessionID, artifact.RelativePath)
	if err != nil {
		return false, err
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, reasonError(ReasonPathInvalid, "stat object: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false, reasonError(ReasonPathInvalid, "object path is symlink")
	}
	return true, nil
}

func (s *Store) validateCreateRequest(req CreateRequest) error {
	if req.Body == nil {
		return reasonError(ReasonRefInvalid, "body is nil")
	}
	if err := validateSessionID(req.SessionID); err != nil {
		return err
	}
	if err := validateSurface(req.Surface); err != nil {
		return err
	}
	if req.SizeHintBytes < 0 {
		return reasonError(ReasonRefInvalid, "size hint must not be negative")
	}
	return nil
}
