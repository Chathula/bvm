package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chathula/bvm/util"
)

func setupCommandEnv(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("BVM_DIR", "")
}

func TestUninstallRemovesInactiveVersion(t *testing.T) {
	setupCommandEnv(t)

	fakeInstall := func(version string) {
		dir, err := util.VersionDir(version)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, util.BinaryName()), []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	fakeInstall("v1.0.0")
	fakeInstall("v2.0.0")

	// Make v1.0.0 active; Uninstall must refuse to remove it.
	if err := util.Activate("v1.0.0"); err != nil {
		t.Fatalf("Activate(v1.0.0) error = %v", err)
	}

	if err := Uninstall("2.0.0"); err != nil {
		t.Fatalf("Uninstall(2.0.0) error = %v", err)
	}
	if util.IsInstalled("v2.0.0") {
		t.Fatal("v2.0.0 still installed after uninstall")
	}
	if err := Uninstall("1.0.0"); err == nil {
		t.Fatal("Uninstall(active version) expected error")
	}
	if !util.IsInstalled("v1.0.0") {
		t.Fatal("active version was removed despite guard")
	}
	if err := Uninstall("9.9.9"); err == nil {
		t.Fatal("Uninstall(not installed) expected error")
	}
}
