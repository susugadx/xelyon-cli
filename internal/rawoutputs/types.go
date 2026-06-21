package rawoutputs

import (
	"context"
	"io"
	"time"
)

// Root は raw output artifact store の root directory。
type Root string

// Surface は raw output ref が属する provider-facing surface。
type Surface string

const (
	// SurfaceCommandOutput は bash/command tool result の raw output。
	SurfaceCommandOutput Surface = "command_output"
	// SurfaceMCPToolResult は MCP tool result の raw output。
	SurfaceMCPToolResult Surface = "mcp_tool_result"
	// SurfaceXelyonWebSearchToolResult は XELYON web_search tool result の raw output。
	SurfaceXelyonWebSearchToolResult Surface = "xelyon_web_search_tool_result"
	// SurfaceProviderNativeBuiltinReplay は provider-native built-in replay の raw output。
	SurfaceProviderNativeBuiltinReplay Surface = "provider_native_builtin_replay"
	// SurfaceReviewProbeResult は review probe result の raw output。
	SurfaceReviewProbeResult Surface = "review_probe_result"
)

// Reason は raw output artifact lifecycle / resolver の structured reason。
type Reason string

const (
	ReasonArtifactTooLarge              Reason = "raw_output_artifact_too_large"
	ReasonSessionQuotaExceeded          Reason = "raw_output_artifact_session_quota_exceeded"
	ReasonArtifactMissing               Reason = "raw_output_artifact_missing"
	ReasonArtifactHashMismatch          Reason = "raw_output_artifact_hash_mismatch"
	ReasonArtifactQuarantined           Reason = "raw_output_artifact_quarantined"
	ReasonArtifactTombstoned            Reason = "raw_output_artifact_tombstoned"
	ReasonArtifactGCCollected           Reason = "raw_output_artifact_gc_collected"
	ReasonManifestCorrupt               Reason = "raw_output_manifest_corrupt"
	ReasonIndexCorrupt                  Reason = "raw_output_index_corrupt"
	ReasonEncryptionRequired            Reason = "raw_output_encryption_required"
	ReasonDecryptFailed                 Reason = "raw_output_decrypt_failed"
	ReasonPathInvalid                   Reason = "raw_output_path_invalid"
	ReasonRefInvalid                    Reason = "raw_output_ref_invalid"
	ReasonLegacySourceMissing           Reason = "raw_output_legacy_source_missing"
	ReasonLegacySourceAmbiguous         Reason = "raw_output_legacy_source_ambiguous"
	ReasonArtifactMaterializationFailed Reason = "raw_output_artifact_materialization_failed"
	ReasonSensitiveArtifactForbidden    Reason = "sensitive_output_artifact_forbidden"
)

const (
	recordTypeCreated          = "raw_output_artifact_created"
	recordTypeQuarantined      = "raw_output_artifact_quarantined"
	recordTypeTombstoned       = "raw_output_artifact_tombstoned"
	recordTypeGCCollected      = "raw_output_artifact_gc_collected"
	hashAlgorithmSHA256        = "sha256"
	storageEncodingRaw         = "raw"
	storageEncodingEncV1       = "enc:aes256gcm:v1"
	storageEncodingEncStreamV1 = "enc:stream:aes256gcm:v1"
	storageEncodingEncStreamV2 = "enc:stream:aes256gcm:v2"
	retentionSession           = "session"
)

// StoreOptions は raw output artifact store の lifecycle 設定。
type StoreOptions struct {
	MaxArtifactBytes  int64
	SessionQuotaBytes int64
	ChunkBytes        int
	EncryptionEnabled bool
	Passphrase        string
	Now               func() time.Time
}

// SourceMetadata は manifest/report に残せる redacted source metadata。
type SourceMetadata struct {
	Provider       string `json:"provider,omitempty"`
	Model          string `json:"model,omitempty"`
	CommandHash    string `json:"command_hash,omitempty"`
	CommandPreview string `json:"command_preview,omitempty"`
	ToolName       string `json:"tool_name,omitempty"`
	ToolCallID     string `json:"tool_call_id,omitempty"`
	EventID        string `json:"event_id,omitempty"`
	HistoryIndex   int    `json:"history_index,omitempty"`
}

// ClassificationMetadata は commandoutputs/providerhistory 側の分類 metadata。
type ClassificationMetadata struct {
	SemanticRole string `json:"semantic_role,omitempty"`
	Family       string `json:"family,omitempty"`
	Subfamily    string `json:"subfamily,omitempty"`
	Classifier   string `json:"classifier,omitempty"`
	Sensitive    bool   `json:"sensitive,omitempty"`
}

// RetentionPolicy は artifact retention policy。
type RetentionPolicy struct {
	Policy    string    `json:"policy"`
	CreatedAt time.Time `json:"created_at"`
}

// RawOutputRef は provider-facing placeholder から raw artifact を解決する ref metadata。
type RawOutputRef struct {
	RefID          string `json:"ref_id"`
	Surface        string `json:"surface"`
	SessionID      string `json:"session_id"`
	EventID        string `json:"event_id,omitempty"`
	HistoryIndex   int    `json:"history_index,omitempty"`
	ToolName       string `json:"tool_name,omitempty"`
	ToolCallID     string `json:"tool_call_id,omitempty"`
	CommandHash    string `json:"command_hash,omitempty"`
	CommandPreview string `json:"command_preview,omitempty"`
	ArtifactID     string `json:"artifact_id"`
	Family         string `json:"family,omitempty"`
	Subfamily      string `json:"subfamily,omitempty"`
	SemanticRole   string `json:"semantic_role,omitempty"`
	Classifier     string `json:"classifier,omitempty"`
	ContentHash    string `json:"content_hash"`
	ByteSize       int    `json:"byte_size"`
	RuneSize       int    `json:"rune_size"`
	ApproxTokens   int    `json:"approx_tokens"`
}

// RawOutputArtifact は artifact object の lifecycle metadata。
type RawOutputArtifact struct {
	ArtifactID      string `json:"artifact_id"`
	HashAlgorithm   string `json:"hash_algorithm"`
	ContentHash     string `json:"content_hash"`
	ByteSize        int    `json:"byte_size"`
	StorageEncoding string `json:"storage_encoding"`
	Encrypted       bool   `json:"encrypted"`
	RelativePath    string `json:"relative_path"`
}

// ManifestRecord は raw output artifact manifest の append-only event。
type ManifestRecord struct {
	SchemaVersion  int                    `json:"schema_version"`
	RecordType     string                 `json:"record_type"`
	Ref            RawOutputRef           `json:"ref,omitempty"`
	Source         SourceMetadata         `json:"source,omitempty"`
	Classification ClassificationMetadata `json:"classification,omitempty"`
	Artifact       RawOutputArtifact      `json:"artifact,omitempty"`
	Retention      RetentionPolicy        `json:"retention,omitempty"`
	Reason         string                 `json:"reason,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

// CreateRequest は raw body artifact 作成入力。
type CreateRequest struct {
	Surface        Surface
	SessionID      string
	Source         SourceMetadata
	Classification ClassificationMetadata
	Body           io.Reader
	SizeHintBytes  int64
	Retention      RetentionPolicy
}

// CreateResult は artifact 作成結果。
type CreateResult struct {
	Ref      RawOutputRef
	Artifact RawOutputArtifact
	Record   ManifestRecord
}

// ResolvedArtifact は hash verification 済み raw body stream。
type ResolvedArtifact struct {
	Ref         RawOutputRef
	Body        io.ReadCloser
	SizeBytes   int64
	ContentHash string
}

// VerifyResult は body を返さない artifact verification 結果。
type VerifyResult struct {
	Ref         RawOutputRef
	OK          bool
	Reason      Reason
	ContentHash string
	SizeBytes   int64
}

// ChunkScanner は raw output body chunk を順次受け取る。
type ChunkScanner interface {
	Scan(chunk []byte) error
}

// ScanRequest は raw output body を materialize せず検証しながら scan する入力。
type ScanRequest struct {
	Ref     RawOutputRef
	Scanner ChunkScanner
}

// ScanResult は streaming scan の検証結果。
type ScanResult struct {
	Ref         RawOutputRef
	ContentHash string
	SizeBytes   int64
}

// LegacyMaterializeRequest は legacy history/session source から artifact を materialize する入力。
type LegacyMaterializeRequest struct {
	CreateRequest
	ExactSourceID string
	Ambiguous     bool
}

// GCRequest は caller-provided live refs による GC 入力。
type GCRequest struct {
	SessionID string
	LiveRefs  []RawOutputRef
	DryRun    bool
}

// GCResult は GC の結果。
type GCResult struct {
	DryRun               bool
	TombstonedRefIDs     []string
	CollectedArtifactIDs []string
	KeptArtifactIDs      []string
}

// DiagnosticsRequest は read-only store diagnostics の入力。
type DiagnosticsRequest struct {
	SessionID       string
	LiveRefs        []RawOutputRef
	IncludeVerify   bool
	IncludeRefs     bool
	RefLimit        int
	IncludeGCDryRun bool
}

// DiagnosticsResult は raw output store の read-only diagnostics 結果。
type DiagnosticsResult struct {
	Root                      string
	SessionID                 string
	StoreExists               bool
	RefCount                  int
	ArtifactCount             int
	LiveRefSourceCount        int
	ByteSize                  int64
	MissingObjects            int
	HashMismatches            int
	DecryptFailures           int
	PathFailures              int
	QuarantinedRefs           int
	TombstonedRefs            int
	CollectedRefs             int
	Refs                      []RefDiagnostic
	GCDryRun                  GCResult
	GCDryRunAvailable         bool
	GCDryRunUnavailableReason string
}

// RefDiagnostic は raw output ref 単位の read-only diagnostics 結果。
type RefDiagnostic struct {
	Ref          RawOutputRef
	Artifact     RawOutputArtifact
	Lifecycle    string
	LiveStatus   string
	VerifyReason Reason
}

// IndexResult は manifest から rebuild した index summary。
type IndexResult struct {
	RecordCount int
	LiveRefs    int
}

// Store は raw output artifact storage / resolver / lifecycle ledger。
type Store struct {
	root Root
	opts StoreOptions
}

// OpenStore は raw output artifact store を開く。
func OpenStore(root Root, opts StoreOptions) (*Store, error) {
	return openStore(root, opts, true)
}

// OpenStoreReadOnly は root や session directory を作らず raw output artifact store を開く。
func OpenStoreReadOnly(root Root, opts StoreOptions) (*Store, error) {
	return openStore(root, opts, false)
}

// Create は raw body を content-addressed artifact として保存する。
func (s *Store) Create(ctx context.Context, req CreateRequest) (CreateResult, error) {
	return s.create(ctx, req)
}

// Resolve は raw output ref を検証し body stream を返す。
func (s *Store) Resolve(ctx context.Context, ref RawOutputRef) (ResolvedArtifact, error) {
	return s.resolve(ctx, ref)
}

// Verify は body を返さず artifact lifecycle/hash を検証する。
func (s *Store) Verify(ctx context.Context, ref RawOutputRef) (VerifyResult, error) {
	return s.verify(ctx, ref)
}

// LookupRef は session manifest から live RawOutputRef metadata を read-only で取得する。
func (s *Store) LookupRef(ctx context.Context, sessionID, refID string) (RawOutputRef, error) {
	return s.lookupRef(ctx, sessionID, refID)
}

// Scan は raw output ref を検証し、body を chunk 単位で scanner に渡す。
func (s *Store) Scan(ctx context.Context, req ScanRequest) (ScanResult, error) {
	return s.scan(ctx, req)
}

// MaterializeLegacy は exact legacy source から artifact を作成する。
func (s *Store) MaterializeLegacy(ctx context.Context, req LegacyMaterializeRequest) (CreateResult, error) {
	return s.materializeLegacy(ctx, req)
}

// CollectGarbage は caller-provided live refs から session-local GC を実行する。
func (s *Store) CollectGarbage(ctx context.Context, req GCRequest) (GCResult, error) {
	return s.collectGarbage(ctx, req)
}

// RebuildIndex は manifest から rebuildable index を再作成する。
func (s *Store) RebuildIndex(ctx context.Context) (IndexResult, error) {
	return s.rebuildIndex(ctx)
}

// Diagnostics は manifest と object 状態を read-only で集計する。
func (s *Store) Diagnostics(ctx context.Context, req DiagnosticsRequest) (DiagnosticsResult, error) {
	return s.diagnostics(ctx, req)
}
