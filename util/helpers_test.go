package util

import (
	"os"
	"path/filepath"
	"testing"
)

// isolateEnv points HOME/BVM_DIR at a temp dir so tests never touch the
// real user environment.
func isolateEnv(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("BVM_DIR", "")
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp) // os.UserHomeDir on Windows
	return tmp
}

// root2 re-derives the isolated bvm root after a fresh isolateEnv call.
func root2(t *testing.T) string {
	t.Helper()
	r, err := BVMDir()
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// fakeInstall creates a plausible installed version directory.
func fakeInstall(t *testing.T, version, content string) {
	t.Helper()
	dir, err := VersionDir(version)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, BinaryName()), []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}
