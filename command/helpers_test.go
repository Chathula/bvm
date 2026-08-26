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

func breakHomeForCommand(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
}

// fakeInstall creates a plausible installed version directory.
func fakeInstall(t *testing.T, version string) {
	t.Helper()
	dir, err := util.VersionDir(version)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, util.BinaryName()), []byte("bin-"+version), 0o755); err != nil {
		t.Fatal(err)
	}
}

// mustBVMDir returns the isolated bvm root for the current test env.
func mustBVMDir(t *testing.T) string {
	t.Helper()
	root, err := util.BVMDir()
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// mustBunShimPath returns where the bun shim lives for the current env.
func mustBunShimPath(t *testing.T) string {
	t.Helper()
	p, err := util.BunBinPath()
	if err != nil {
		t.Fatal(err)
	}
	return p
}
