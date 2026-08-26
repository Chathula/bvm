package util

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEnsureShimLinksOrCopiesBvmBinary(t *testing.T) {
	isolateEnv(t)
	fakeInstall(t, "v1.0.0", "x")

	if err := EnsureShim(); err != nil {
		t.Fatalf("EnsureShim() error = %v", err)
	}

	shim := mustBunPath(t)
	data, err := os.ReadFile(shim)
	if err != nil {
		t.Fatalf("shim missing: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("shim is empty")
	}

	if runtime.GOOS != "windows" {
		exe, _ := os.Executable()
		resolved, _ := filepath.EvalSymlinks(exe)
		target, err := os.Readlink(shim)
		if err != nil {
			t.Fatalf("expected symlink shim, got error: %v", err)
		}
		if filepath.Clean(target) != filepath.Clean(resolved) {
			t.Fatalf("shim target = %q, want %q", target, resolved)
		}
	}
}

func TestEnsureShimReplacesStaleShim(t *testing.T) {
	isolateEnv(t)
	shim := mustBunPath(t)
	os.MkdirAll(filepath.Dir(shim), 0o755)
	os.WriteFile(shim, []byte("stale garbage"), 0o644)

	if err := EnsureShim(); err != nil {
		t.Fatalf("EnsureShim() over stale shim error = %v", err)
	}
	data, _ := os.ReadFile(shim)
	if string(data) == "stale garbage" {
		t.Fatal("stale shim was not replaced")
	}
}

func TestEnsureShimErrorsWithoutHome(t *testing.T) {
	breakHome(t)
	if err := EnsureShim(); err == nil {
		t.Fatal("expected error with unresolvable home")
	}
}

func TestEnsureShimErrorBranches(t *testing.T) {
	t.Run("executable lookup fails", func(t *testing.T) {
		isolateEnv(t)
		stub(t, &osExecutable, func() (string, error) { return "", fmt.Errorf("injected") })
		if err := EnsureShim(); err == nil {
			t.Fatal("expected os.Executable error to propagate")
		}
	})
	t.Run("bin dir is a file", func(t *testing.T) {
		home := isolateEnv(t)
		os.MkdirAll(filepath.Join(home, ".bun"), 0o755)
		os.WriteFile(filepath.Join(home, ".bun", "bin"), []byte("file"), 0o644)
		if err := EnsureShim(); err == nil {
			t.Fatal("expected MkdirAll error when shim dir is a file")
		}
	})
	t.Run("stale shim cannot be removed", func(t *testing.T) {
		isolateEnv(t)
		stub(t, &RemoveAll, func(string) error { return fmt.Errorf("injected") })
		if err := EnsureShim(); err == nil {
			t.Fatal("expected RemoveAll error to propagate")
		}
	})
}

func TestPlaceBinaryCopyMode(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	os.WriteFile(src, []byte("payload"), 0o755)

	dst := filepath.Join(tmp, "dst-copy")
	if err := placeBinary(src, dst, false); err != nil {
		t.Fatalf("copy mode error = %v", err)
	}
	if data, _ := os.ReadFile(dst); string(data) != "payload" {
		t.Fatal("copy mode produced wrong content")
	}

	// Missing source surfaces through the copy path.
	if err := placeBinary(filepath.Join(tmp, "missing"), filepath.Join(tmp, "out"), false); err == nil {
		t.Fatal("expected copy error for missing source")
	}
}

func TestRemoveShim(t *testing.T) {
	isolateEnv(t)
	// Absent shim is a no-op.
	if err := RemoveShim(); err != nil {
		t.Fatalf("RemoveShim() on missing shim error = %v", err)
	}

	shim := mustBunPath(t)
	os.MkdirAll(filepath.Dir(shim), 0o755)
	os.WriteFile(shim, []byte("shim"), 0o755)
	if err := RemoveShim(); err != nil {
		t.Fatalf("RemoveShim() error = %v", err)
	}
	if _, err := os.Stat(shim); !os.IsNotExist(err) {
		t.Fatal("shim still present after RemoveShim")
	}

	// NotExist from removal is tolerated (returns nil).
	stub(t, &RemoveAll, func(string) error { return fs.ErrNotExist })
	if err := RemoveShim(); err != nil {
		t.Fatalf("RemoveShim() with NotExist error = %v", err)
	}

	// Other removal failures propagate.
	stub(t, &RemoveAll, func(string) error { return fmt.Errorf("injected") })
	os.WriteFile(shim, []byte("shim"), 0o755)
	if err := RemoveShim(); err == nil {
		t.Fatal("expected RemoveAll error to propagate")
	}

	breakHome(t)
	if err := RemoveShim(); err == nil {
		t.Fatal("expected error with unresolvable home")
	}
}

func TestClearDefault(t *testing.T) {
	isolateEnv(t)
	// Clearing when absent is a no-op.
	if err := ClearDefault(); err != nil {
		t.Fatalf("ClearDefault() on missing alias error = %v", err)
	}

	if err := SetDefaultVersion("v1.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := ClearDefault(); err != nil {
		t.Fatalf("ClearDefault() error = %v", err)
	}
	if def, _ := DefaultVersion(); def != "" {
		t.Fatalf("default still set: %q", def)
	}
}

func mustBunPath(t *testing.T) string {
	t.Helper()
	p, err := BunBinPath()
	if err != nil {
		t.Fatal(err)
	}
	return p
}
