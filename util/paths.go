package util

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
)

// readDir is a var so tests can inject directory-listing failures portably.
var readDir = os.ReadDir

// statPath is a var so tests can inject stat failures portably.
var statPath = os.Stat

// RemoveAll is a var so tests can inject deletion failures portably.
var RemoveAll = os.RemoveAll

// ErrUnsupportedPlatform is returned when the host OS/arch has no Bun build.
var ErrUnsupportedPlatform = errors.New("unsupported platform")

// BinaryName returns the bun executable name on the host OS.
func BinaryName() string {
	return BinaryNameForOS(runtime.GOOS)
}

// BinaryNameForOS returns the bun executable name for a given GOOS.
func BinaryNameForOS(goos string) string {
	if goos == "windows" {
		return "bun.exe"
	}
	return "bun"
}

// BVMDir returns the bvm root directory ($BVM_DIR, default ~/.bvm).
func BVMDir() (string, error) {
	if dir := os.Getenv("BVM_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".bvm"), nil
}

// VersionsDir returns the directory holding installed Bun versions.
func VersionsDir() (string, error) {
	root, err := BVMDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "versions"), nil
}

// VersionDir returns the install directory of a single version ("v1.1.0").
func VersionDir(version string) (string, error) {
	dir, err := VersionsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, version), nil
}

// BunBinPath returns where the active bun binary is exposed (~/.bun/bin/bun).
func BunBinPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".bun", "bin", BinaryName()), nil
}

// IsInstalled reports whether a version has its binary in place locally.
func IsInstalled(version string) bool {
	dir, err := VersionDir(version)
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(dir, BinaryName()))
	return err == nil
}

// LocalVersions returns installed versions sorted ascending (oldest first).
func LocalVersions() ([]string, error) {
	dir, err := VersionsDir()
	if err != nil {
		return nil, err
	}
	info, err := statPath(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	entries, err := readDir(dir)
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, e := range entries {
		if e.IsDir() {
			versions = append(versions, e.Name())
		}
	}
	sort.Strings(versions)
	return versions, nil
}
