package command

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/chathula/bvm/util"
)

func TestAliasSetsAndUsesDefault(t *testing.T) {
	setupCommandEnv(t)
	quietPATH(t)
	fakeInstall(t, "v1.0.0")
	fakeInstall(t, "v2.0.0")

	if err := Alias("other-alias", "2.0.0"); err == nil {
		t.Fatal("expected error for unsupported alias name")
	}
	if err := Alias("default", ""); err == nil {
		t.Fatal("expected error when version missing")
	}
	if err := Alias("default", "not-a-version"); err == nil {
		t.Fatal("expected error for invalid version string")
	}

	if err := Alias("default", "2.0.0"); err != nil {
		t.Fatalf("Alias(default, 2.0.0) error = %v", err)
	}
	if def, _ := util.DefaultVersion(); def != "v2.0.0" {
		t.Fatalf("DefaultVersion() = %q, want v2.0.0", def)
	}

	// Unpinned directory resolves to the default via the shim.
	project := filepath.Join(t.TempDir(), "deep")
	os.MkdirAll(project, 0o755)
	t.Chdir(project)

	version, source, err := util.ResolveVersion()
	if err != nil || version != "v2.0.0" {
		t.Fatalf("ResolveVersion() = (%q, %q, %v), want default v2.0.0", version, source, err)
	}
}

func TestAliasRemoteResolutionArms(t *testing.T) {
	setupCommandEnv(t)

	stub(t, &resolveTarget, func(string) (string, error) { return "", fmt.Errorf("offline") })
	if err := Alias("default", "2.0.0"); err == nil {
		t.Fatal("expected remote resolve error")
	}

	stub(t, &resolveTarget, func(string) (string, error) { return "v2.0.0", nil })
	if err := Alias("default", "2.0.0"); err != nil {
		t.Fatalf("Alias() error = %v", err)
	}
	if def, _ := util.DefaultVersion(); def != "v2.0.0" {
		t.Fatalf("default = %q", def)
	}
}

func TestAliasSetDefaultWriteError(t *testing.T) {
	root := t.TempDir()
	t.Setenv("BVM_DIR", filepath.Join(root, "as-file"))
	os.WriteFile(filepath.Join(root, "as-file"), []byte("x"), 0o644)

	if err := Alias("default", "1.0.0"); err == nil {
		t.Fatal("expected SetDefaultVersion error")
	}
}
