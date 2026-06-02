package probe

import (
	"os"
	"path/filepath"
)

type probeGeneratedFile struct {
	absPath string
	content string
}

type probeGeneratedFileValidationSpec struct {
	modeName  string
	rootLabel string
	rootDir   string
}

func validateProbeGeneratedFileTarget(spec probeGeneratedFileValidationSpec, absPath, label string) error {
	if err := validateModeExistingAncestorsWithinRoot(spec.rootDir, absPath, label, spec.modeName, spec.rootLabel); err != nil {
		return err
	}
	if _, err := os.Lstat(absPath); err == nil {
		return newBlockedCommandErrorf("%s would overwrite existing file", label)
	} else if !os.IsNotExist(err) {
		return newBlockedCommandErrorf("failed to inspect %s: %v", label, err)
	}
	return nil
}

func writeProbeGeneratedFiles(files []probeGeneratedFile) error {
	for _, file := range files {
		if err := os.MkdirAll(filepath.Dir(file.absPath), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(file.absPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return err
		}
		if _, err := out.Write([]byte(file.content)); err != nil {
			_ = out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
	}
	return nil
}
