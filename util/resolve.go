package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// rcFileName is the per-project version pin file, analogous to .nvmrc.
const rcFileName = ".bvmrc"

// FindRC walks up from dir looking for a .bvmrc file, returning its path
// and the normalized version inside it. Empty when none is found.
func FindRC(dir string) (path, version string) {
	for {
		candidate := filepath.Join(dir, rcFileName)
		if data, err := os.ReadFile(candidate); err == nil {
			if v := firstVersionLine(string(data)); v != "" {
				return candidate, v
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "" // reached the root
		}
		dir = parent
	}
}

// firstVersionLine returns the first parseable version in a .bvmrc,
// skipping blank lines and # comments.
func firstVersionLine(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if v, err := NormalizeVersion(line); err == nil {
			return v
		}
	}
	return ""
}

// getCwd is a var so tests can inject lookup failures portably.
var getCwd = os.Getwd

// FindRCHere walks up from the current working directory.
func FindRCHere() (path, version string) {
	wd, err := getCwd()
	if err != nil {
		return "", ""
	}
	return FindRC(wd)
}

// ResolveVersion determines which bun version applies to the current
// directory: the nearest .bvmrc walking up, otherwise the default alias.
// Returns the version and where it came from ("project pin" / "default").
func ResolveVersion() (version, source string, err error) {
	if rcPath, v := FindRCHere(); rcPath != "" {
		if !IsInstalled(v) {
			return "", "", fmt.Errorf("bun %s is pinned in %s but not installed — run 'bvm install %s'", v, rcPath, v)
		}
		return v, fmt.Sprintf("project pin (%s)", rcPath), nil
	}

	def, err := DefaultVersion()
	if err != nil {
		return "", "", err
	}
	if def == "" {
		return "", "", fmt.Errorf("no bun version resolved — no %s here and no default set; run 'bvm install latest'", rcFileName)
	}
	if !IsInstalled(def) {
		return "", "", fmt.Errorf("default bun %s is not installed — run 'bvm install %s'", def, def)
	}
	return def, "default", nil
}

// RCFileName is the name of the per-project pin file (".bvmrc").
func RCFileName() string { return rcFileName }

// WriteRC pins version for the current directory.
func WriteRC(version string) error {
	return os.WriteFile(rcFileName, []byte(version+"\n"), 0o644)
}

// RemoveRC deletes the local pin file when present.
func RemoveRC() error {
	err := os.Remove(rcFileName)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
