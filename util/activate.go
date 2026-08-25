package util

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const activeMarker = "active"

// ActiveVersion returns the version recorded as active, or "" if none is.
func ActiveVersion() (string, error) {
	root, err := BVMDir()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(root, activeMarker))
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// Activate exposes an installed version as ~/.bun/bin/bun — symlink on
// unix, copy on Windows (symlinks need elevated rights there) — and records
// it in $BVM_DIR/active.
func Activate(version string) error {
	if !IsInstalled(version) {
		return fmt.Errorf("version %s is not installed locally", version)
	}

	srcDir, err := VersionDir(version)
	if err != nil {
		return err
	}
	dst, err := BunBinPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.RemoveAll(dst); err != nil {
		return err
	}

	src := filepath.Join(srcDir, BinaryName())
	if runtime.GOOS == "windows" {
		if err := copyFile(src, dst); err != nil {
			return err
		}
	} else if err := os.Symlink(src, dst); err != nil {
		return err
	}

	root, err := BVMDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, activeMarker), []byte(version), 0o644)
}

// Deactivate removes the exposed bun binary and the active-version marker.
func Deactivate() error {
	dst, err := BunBinPath()
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dst); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	root, err := BVMDir()
	if err != nil {
		return err
	}
	err = os.Remove(filepath.Join(root, activeMarker))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
