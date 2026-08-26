package util

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const defaultAliasFile = "default"

// DefaultVersion returns the recorded default alias value, or "" when unset.
// The value may reference an uninstalled version; callers verify with
// IsInstalled before relying on it.
func DefaultVersion() (string, error) {
	root, err := BVMDir()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(root, defaultAliasFile))
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// SetDefaultVersion records version as the default alias.
func SetDefaultVersion(version string) error {
	root, err := BVMDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, defaultAliasFile), []byte(version), 0o644)
}
