package rawoutputs

import (
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	defaultMaxArtifactBytes  = 64 * 1024 * 1024
	defaultSessionQuotaBytes = 1024 * 1024 * 1024
	defaultChunkBytes        = 1024 * 1024
	schemaVersion            = 1
	encryptedLinePrefix      = "enc:"
	refIDPrefix              = "rawout_"
	maxPreviewRunes          = 120
)

var safeIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type lifecycleState struct {
	created     ManifestRecord
	quarantined bool
	tombstoned  bool
	collected   bool
}

func openStore(root Root, opts StoreOptions) (*Store, error) {
	if strings.TrimSpace(string(root)) == "" {
		return nil, reasonError(ReasonPathInvalid, "root is empty")
	}
	opts = normalizeStoreOptions(opts)
	if opts.EncryptionEnabled && strings.TrimSpace(opts.Passphrase) == "" {
		return nil, reasonError(ReasonEncryptionRequired, "encryption enabled without passphrase")
	}
	cleanRoot, err := secureCleanRoot(string(root))
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cleanRoot, 0o700); err != nil {
		return nil, reasonError(ReasonPathInvalid, "create root: %w", err)
	}
	if err := ensureExistingDirIsNotSymlink(cleanRoot); err != nil {
		return nil, err
	}
	return &Store{root: Root(cleanRoot), opts: opts}, nil
}

func normalizeStoreOptions(opts StoreOptions) StoreOptions {
	if opts.MaxArtifactBytes <= 0 {
		opts.MaxArtifactBytes = defaultMaxArtifactBytes
	}
	if opts.SessionQuotaBytes <= 0 {
		opts.SessionQuotaBytes = defaultSessionQuotaBytes
	}
	if opts.ChunkBytes <= 0 {
		opts.ChunkBytes = defaultChunkBytes
	}
	if opts.ChunkBytes > int(opts.MaxArtifactBytes) {
		opts.ChunkBytes = int(opts.MaxArtifactBytes)
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return opts
}
