package util

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type shellProfile struct {
	shell string
	file  string
	line  string
}

// shellProfiles lists the rc files to update per OS. Windows profiles are
// never auto-edited; doctor reports the PATH state instead.
func shellProfiles(goos, home, binDir string) []shellProfile {
	if goos == "windows" {
		return nil
	}
	return []shellProfile{
		{"zsh", filepath.Join(home, ".zshrc"), exportLine(binDir)},
		{"bash", filepath.Join(home, ".bashrc"), exportLine(binDir)},
		{"fish", filepath.Join(home, ".config", "fish", "config.fish"), "fish_add_path " + binDir},
	}
}

// EnsurePATH appends the bun bin directory to the user's shell profile when
// it is missing from PATH. Returns true when a profile was modified.
func EnsurePATH() (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}
	binDir := filepath.Join(home, ".bun", "bin")
	if PathContains(binDir) {
		return false, nil
	}
	return applyProfiles(shellProfiles(runtime.GOOS, home, binDir))
}

// applyProfiles writes each profile's line when its shell exists on the
// machine. Split from EnsurePATH so every branch is testable on any OS.
func applyProfiles(profiles []shellProfile) (bool, error) {
	changed := false
	for _, c := range profiles {
		if _, err := exec.LookPath(c.shell); err != nil {
			continue // shell not installed on this machine
		}
		appended, err := appendLineIfMissing(c.file, c.line)
		if err != nil {
			return changed, err
		}
		changed = changed || appended
	}
	return changed, nil
}

func BunBinDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".bun", "bin"), nil
}

// PathContains reports whether PATH holds exactly the given entry.
func PathContains(dir string) bool {
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p == dir {
			return true
		}
	}
	return false
}

func exportLine(binDir string) string {
	return `export PATH="$PATH:` + binDir + `"`
}

// openAppend is a var so tests can inject file-open failures portably.
var openAppend = func(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
}

// appendLineIfMissing writes line to path once; reports whether it wrote.
func appendLineIfMissing(path, line string) (bool, error) {
	data, readErr := os.ReadFile(path)
	switch {
	case readErr == nil && strings.Contains(string(data), line):
		return false, nil
	case readErr != nil && !errors.Is(readErr, fs.ErrNotExist):
		return false, readErr
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	f, err := openAppend(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	_, err = f.WriteString("\n# added by bvm\n" + line + "\n")
	return true, err
}
