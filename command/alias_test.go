package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chathula/bvm/util"
)

func fakeInstall(t *testing.T, version string) {
	t.Helper()
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

func TestEnsureDefaultSetOnlyOnFirstInstall(t *testing.T) {
	setupCommandEnv(t)
	fakeInstall(t, "v1.0.0")

	// No default yet → the first install claims it.
	t.Chdir(t.TempDir())
	if err := ensureDefaultSet("v1.0.0"); err != nil {
		t.Fatalf("ensureDefaultSet(v1.0.0) error = %v", err)
	}
	if def, _ := util.DefaultVersion(); def != "v1.0.0" {
		t.Fatalf("default = %q, want v1.0.0", def)
	}

	// Later installs must never steal the default.
	if err := ensureDefaultSet("v2.0.0"); err != nil {
		t.Fatalf("ensureDefaultSet(v2.0.0) error = %v", err)
	}
	if def, _ := util.DefaultVersion(); def != "v1.0.0" {
		t.Fatalf("default changed to %q, want it to stay v1.0.0", def)
	}
}

func TestAliasSetsAndUsesDefault(t *testing.T) {
	setupCommandEnv(t)
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

	// Plain 'bvm use' in a directory without .bvmrc must fall back to it.
	t.Chdir(t.TempDir())
	if err := Use(""); err != nil {
		t.Fatalf("Use() error = %v", err)
	}
	if active, _ := util.ActiveVersion(); active != "v2.0.0" {
		t.Fatalf("active = %q, want v2.0.0", active)
	}

	// Explicit 'bvm use default' resolves the alias too.
	if err := util.Activate("v1.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := Use("default"); err != nil {
		t.Fatalf("Use(default) error = %v", err)
	}
	if active, _ := util.ActiveVersion(); active != "v2.0.0" {
		t.Fatalf("active after 'use default' = %q, want v2.0.0", active)
	}
}

func TestStaleDefaultFallsBackToHighestInstalled(t *testing.T) {
	setupCommandEnv(t)
	fakeInstall(t, "v1.0.0")
	fakeInstall(t, "v3.0.0")

	// Default points at something no longer installed.
	if err := util.SetDefaultVersion("v9.9.9"); err != nil {
		t.Fatal(err)
	}

	t.Chdir(t.TempDir())
	if err := Use(""); err != nil {
		t.Fatalf("Use() error = %v", err)
	}
	if active, _ := util.ActiveVersion(); active != "v3.0.0" {
		t.Fatalf("active = %q, want fallback v3.0.0", active)
	}
}

func TestUseDefaultWithoutAliasErrors(t *testing.T) {
	setupCommandEnv(t)
	fakeInstall(t, "v1.0.0")

	if err := Use("default"); err == nil {
		t.Fatal("expected error when no default alias is set")
	}
}

func TestUseDefaultWhenDefaultNotInstalledErrors(t *testing.T) {
	setupCommandEnv(t)
	fakeInstall(t, "v1.0.0")
	if err := util.SetDefaultVersion("v9.9.9"); err != nil {
		t.Fatal(err)
	}

	err := Use("default")
	if err == nil {
		t.Fatal("expected error when default version is not installed")
	}
}
