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
// expose is a var so tests can inject link/copy failures portably.
var expose = exposeBinary

func Activate(version string) error {
	srcDir, err := VersionDir(version)
	if err != nil {
		return err
	}
	if !IsInstalled(version) {
		return fmt.Errorf("version %s is not installed locally", version)
	}
	dst, err := BunBinPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := RemoveAll(dst); err != nil {
		return err
	}

	src := filepath.Join(srcDir, BinaryName())
	if err := expose(src, dst, runtime.GOOS != "windows"); err != nil {
		return err
	}
	return recordActive(version)
}

// exposeBinary links or copies src to dst depending on useSymlink.
// Split from Activate so both strategies stay testable on every OS.
func exposeBinary(src, dst string, useSymlink bool) error {
	if !useSymlink {
		return copyFile(src, dst)
	}
	return os.Symlink(src, dst)
}

// recordActive writes the active-version marker under $BVM_DIR.
func recordActive(version string) error {
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
	root, err := BVMDir()
	if err != nil {
		return err
	}
	dst, err := BunBinPath()
	if err != nil {
		return err
	}
	if err := RemoveAll(dst); err != nil && !errors.Is(err, fs.ErrNotExist) {
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
