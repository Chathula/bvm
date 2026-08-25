package util

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestActivateAndDeactivate(t *testing.T) {
	home := isolateEnv(t)
	fakeInstall(t, "v1.0.0", "binary-v1")
	fakeInstall(t, "v2.0.0", "binary-v2")

	if err := Activate("v1.0.0"); err != nil {
		t.Fatalf("Activate(v1.0.0) error = %v", err)
	}

	dst := filepath.Join(home, ".bun", "bin", BinaryName())
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("activated binary missing: %v", err)
	}
	if string(data) != "binary-v1" {
		t.Fatalf("activated binary content = %q", data)
	}
	if active, _ := ActiveVersion(); active != "v1.0.0" {
		t.Fatalf("ActiveVersion() = %q, want v1.0.0", active)
	}

	// Switching versions must replace the exposed binary.
	if err := Activate("v2.0.0"); err != nil {
		t.Fatalf("Activate(v2.0.0) error = %v", err)
	}
	data, _ = os.ReadFile(dst)
	if string(data) != "binary-v2" {
		t.Fatalf("after switch, binary content = %q", data)
	}

	if runtime.GOOS != "windows" {
		target, err := os.Readlink(dst)
		if err != nil {
			t.Fatalf("expected symlink, got error: %v", err)
		}
		wantDir, _ := VersionDir("v2.0.0")
		if filepath.Clean(target) != filepath.Join(wantDir, BinaryName()) {
			t.Fatalf("symlink target = %q, want %q", target, filepath.Join(wantDir, BinaryName()))
		}
	}

	if err := Deactivate(); err != nil {
		t.Fatalf("Deactivate() error = %v", err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("activated binary still exists after Deactivate")
	}
	if active, _ := ActiveVersion(); active != "" {
		t.Fatalf("active marker still set: %q", active)
	}
}

func TestActivateNotInstalled(t *testing.T) {
	isolateEnv(t)
	if err := Activate("v9.9.9"); err == nil {
		t.Fatal("Activate(uninstalled) expected error")
	}
}

func TestLocalVersionsSorted(t *testing.T) {
	isolateEnv(t)
	fakeInstall(t, "v1.10.0", "x")
	fakeInstall(t, "v1.2.0", "x")
	fakeInstall(t, "v1.9.0", "x")

	got, err := LocalVersions()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"v1.10.0", "v1.2.0", "v1.9.0"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("LocalVersions() = %v, want %v", got, want)
		}
	}
}
