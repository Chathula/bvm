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

// EnsurePATH appends the bun bin directory to the user's shell profile when
// it is missing from PATH. Windows is left untouched; doctor reports it.
// Returns true when a profile was modified.
func EnsurePATH() (bool, error) {
	binDir, err := BunBinDir()
	if err != nil {
		return false, err
	}
	if pathContains(binDir) {
		return false, nil
	}
	if runtime.GOOS == "windows" {
		return false, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}

	type profile struct{ shell, file, line string }
	candidates := []profile{
		{"zsh", filepath.Join(home, ".zshrc"), exportLine(binDir)},
		{"bash", filepath.Join(home, ".bashrc"), exportLine(binDir)},
		{"fish", filepath.Join(home, ".config", "fish", "config.fish"), "fish_add_path " + binDir},
	}

	changed := false
	for _, c := range candidates {
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

func pathContains(dir string) bool {
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

// appendLineIfMissing writes line to path once; reports whether it wrote.
func appendLineIfMissing(path, line string) (bool, error) {
	if data, err := os.ReadFile(path); err == nil && strings.Contains(string(data), line) {
		return false, nil
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return false, err
	}
	defer f.Close()
	_, err = f.WriteString("\n# added by bvm\n" + line + "\n")
	return true, err
}
