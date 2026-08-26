package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chathula/bvm/util"
)

func TestUninstallRemovesVersion(t *testing.T) {
	setupCommandEnv(t)
	fakeInstall(t, "v1.0.0")
	fakeInstall(t, "v2.0.0")

	if err := Uninstall("2.0.0"); err != nil {
		t.Fatalf("Uninstall(2.0.0) error = %v", err)
	}
	if util.IsInstalled("v2.0.0") {
		t.Fatal("v2.0.0 still installed after uninstall")
	}

	if err := Uninstall("9.9.9"); err == nil {
		t.Fatal("Uninstall(not installed) expected error")
	}
	if err := Uninstall("bogus!!"); err == nil {
		t.Fatal("Uninstall(invalid) expected error")
	}
}

func TestUninstallDefaultReassignsToHighestRemaining(t *testing.T) {
	setupCommandEnv(t)
	quietPATH(t)
	fakeInstall(t, "v1.0.0")
	fakeInstall(t, "v2.0.0")
	util.SetDefaultVersion("v2.0.0")

	if err := Uninstall("2.0.0"); err != nil {
		t.Fatalf("Uninstall(default version) error = %v", err)
	}
	if def, _ := util.DefaultVersion(); def != "v1.0.0" {
		t.Fatalf("default = %q, want reassigned v1.0.0", def)
	}
}

func TestUninstallLastVersionCleansUp(t *testing.T) {
	setupCommandEnv(t)
	fakeInstall(t, "v1.0.0")
	util.SetDefaultVersion("v1.0.0")
	os.MkdirAll(filepath.Dir(mustBunShimPath(t)), 0o755)
	os.WriteFile(mustBunShimPath(t), []byte("shim"), 0o755)

	if err := Uninstall("1.0.0"); err != nil {
		t.Fatalf("Uninstall(last version) error = %v", err)
	}
	if def, _ := util.DefaultVersion(); def != "" {
		t.Fatalf("default alias still set: %q", def)
	}
	if _, err := os.Stat(mustBunShimPath(t)); !os.IsNotExist(err) {
		t.Fatal("shim still present after removing the last version")
	}
}

func TestUninstallWarnsWhenCurrentProjectPinsRemovedVersion(t *testing.T) {
	setupCommandEnv(t)
	fakeInstall(t, "v1.0.0")

	project := t.TempDir()
	t.Chdir(project)
	util.WriteRC("v1.0.0")

	out := captureStdout(t, func() { _ = Uninstall("1.0.0") })
	if !strings.Contains(out, ".bvmrc") {
		t.Fatalf("expected pin warning, got:\n%s", out)
	}
}

func TestUninstallErrorArms(t *testing.T) {
	t.Run("unresolvable home hits VersionDir", func(t *testing.T) {
		setupCommandEnv(t)
		breakHomeForCommand(t)
		if err := Uninstall("1.0.0"); err == nil {
			t.Fatal("expected VersionDir failure")
		}
	})

	t.Run("reassign propagates LocalVersions failure", func(t *testing.T) {
		setupCommandEnv(t)
		breakHomeForCommand(t)
		if err := reassignDefault("v1.0.0"); err == nil {
			t.Fatal("expected LocalVersions failure")
		}
	})

	t.Run("clear default failure", func(t *testing.T) {
		setupCommandEnv(t)
		quietShim(t)
		fakeInstall(t, "v1.0.0")
		util.SetDefaultVersion("v1.0.0")
		stub(t, &clearDefault, func() error { return fmt.Errorf("injected") })

		if err := Uninstall("1.0.0"); err == nil {
			t.Fatal("expected ClearDefault failure")
		}
	})

	t.Run("remove shim failure during cleanup", func(t *testing.T) {
		setupCommandEnv(t)
		fakeInstall(t, "v1.0.0")
		util.SetDefaultVersion("v1.0.0")
		stub(t, &util.RemoveAll, func(string) error { return fmt.Errorf("injected") })

		err := Uninstall("1.0.0")
		if err == nil || !strings.Contains(err.Error(), "injected") {
			t.Fatalf("expected RemoveShim failure, got %v", err)
		}
	})

	t.Run("set default failure during reassign", func(t *testing.T) {
		setupCommandEnv(t)
		quietShim(t)
		fakeInstall(t, "v1.0.0")
		fakeInstall(t, "v2.0.0")
		util.SetDefaultVersion("v2.0.0")
		stub(t, &setDefaultVersion, func(string) error { return fmt.Errorf("injected") })

		if err := Uninstall("2.0.0"); err == nil {
			t.Fatal("expected SetDefaultVersion failure")
		}
	})
}
