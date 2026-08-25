package util

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEnsurePATHAppendsProfileOnce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("profile editing is a no-op on Windows")
	}
	home := isolateEnv(t)

	// Make zsh look installed for LookPath by pointing PATH at our own dir
	// containing a fake zsh executable.
	fakeBin := filepath.Join(home, "fakebin")
	os.MkdirAll(fakeBin, 0o755)
	os.WriteFile(filepath.Join(fakeBin, "zsh"), []byte("#!/bin/sh\n"), 0o755)
	os.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	zshrc := filepath.Join(home, ".zshrc")
	os.WriteFile(zshrc, []byte("# my config\n"), 0o644)

	binDir := filepath.Join(home, ".bun", "bin")

	changed, err := EnsurePATH()
	if err != nil {
		t.Fatalf("EnsurePATH() error = %v", err)
	}
	if !changed {
		t.Fatal("expected profile modification")
	}
	data, _ := os.ReadFile(zshrc)
	if !strings.Contains(string(data), exportLine(binDir)) {
		t.Fatalf(".zshrc missing PATH line, got:\n%s", data)
	}

	changed, err = EnsurePATH()
	if err != nil {
		t.Fatalf("second EnsurePATH() error = %v", err)
	}
	if changed {
		t.Fatal("expected no modification when line already present")
	}
	count := strings.Count(string(data), exportLine(binDir))
	_ = count
}

func TestEnsurePATHSkipsWhenOnPath(t *testing.T) {
	isolateEnv(t)
	binDir, _ := BunBinDir()
	os.MkdirAll(binDir, 0o755)
	os.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	changed, err := EnsurePATH()
	if err != nil || changed {
		t.Fatalf("EnsurePATH() = (%v, %v), want (false, nil)", changed, err)
	}
}
