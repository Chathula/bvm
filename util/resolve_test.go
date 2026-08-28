package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindRCWalksUpDirectories(t *testing.T) {
	root := isolateEnv(t)
	project := filepath.Join(root, "code", "project", "pkg", "deep")
	os.MkdirAll(project, 0o755)
	os.WriteFile(filepath.Join(root, "code", "project", ".bvmrc"), []byte("v1.4.0\n"), 0o644)

	path, version := FindRC(project)
	if path == "" || version != "v1.4.0" {
		t.Fatalf("FindRC() = (%q, %q), want v1.4.0", path, version)
	}
}

func TestFindRCNormalizesPrefixAndComments(t *testing.T) {
	dir := isolateEnv(t)
	os.WriteFile(filepath.Join(dir, rcFileName), []byte("\n# comment\n 1.4.0 \n"), 0o644)

	_, version := FindRC(dir)
	if version != "v1.4.0" {
		t.Fatalf("FindRC version = %q, want v1.4.0", version)
	}
}

func TestFindRCNoneFound(t *testing.T) {
	dir := isolateEnv(t)
	path, version := FindRC(dir)
	if path != "" || version != "" {
		t.Fatalf("FindRC() = (%q, %q), want empty", path, version)
	}

	// A file with no parseable version line is treated as absent.
	os.WriteFile(filepath.Join(dir, rcFileName), []byte("# only\nnonsense!!\n"), 0o644)
	path, version = FindRC(dir)
	if path != "" || version != "" {
		t.Fatalf("FindRC(unparseable) = (%q, %q), want empty", path, version)
	}
}

func TestResolveVersionPrefersProjectPinOverDefault(t *testing.T) {
	isolateEnv(t)
	fakeInstall(t, "v1.3.1", "a")
	fakeInstall(t, "v1.4.0", "b")
	SetDefaultVersion("v1.4.0")

	project := filepath.Join(root2(t), "project")
	os.MkdirAll(project, 0o755)
	t.Chdir(project)
	os.WriteFile(filepath.Join(project, rcFileName), []byte("v1.3.1\n"), 0o644)

	version, source, err := ResolveVersion()
	if err != nil {
		t.Fatalf("ResolveVersion() error = %v", err)
	}
	if version != "v1.3.1" {
		t.Fatalf("ResolveVersion() version = %q, want project pin v1.3.1", version)
	}
	if source == "" {
		t.Fatal("expected non-empty source")
	}
}

func TestResolveVersionFallsBackToDefault(t *testing.T) {
	isolateEnv(t)
	fakeInstall(t, "v1.4.0", "b")
	SetDefaultVersion("v1.4.0")

	t.Chdir(t.TempDir()) // no .bvmrc anywhere above

	version, source, err := ResolveVersion()
	if err != nil || version != "v1.4.0" {
		t.Fatalf("ResolveVersion() = (%q, %q, %v)", version, source, err)
	}
}

func TestResolveVersionErrors(t *testing.T) {
	t.Run("nothing installed and no default", func(t *testing.T) {
		isolateEnv(t)
		t.Chdir(t.TempDir())
		if _, _, err := ResolveVersion(); err == nil {
			t.Fatal("expected error when nothing resolves")
		}
	})
	t.Run("pinned version not installed", func(t *testing.T) {
		isolateEnv(t)
		project := t.TempDir()
		t.Chdir(project)
		os.WriteFile(filepath.Join(project, rcFileName), []byte("v9.9.9\n"), 0o644)

		if _, _, err := ResolveVersion(); err == nil {
			t.Fatal("expected error for uninstalled pin")
		}
	})
	t.Run("default not installed", func(t *testing.T) {
		isolateEnv(t)
		t.Chdir(t.TempDir())
		SetDefaultVersion("v9.9.9")

		if _, _, err := ResolveVersion(); err == nil {
			t.Fatal("expected error for uninstalled default")
		}
	})
}

func TestFindRCHereHandlesGetwdFailure(t *testing.T) {
	isolateEnv(t)
	t.Chdir(t.TempDir())
	stub(t, &getCwd, func() (string, error) { return "", fmt.Errorf("injected") })

	if path, version := FindRCHere(); path != "" || version != "" {
		t.Fatalf("FindRCHere() = (%q, %q), want empty on failure", path, version)
	}
}

func TestResolveVersionSessionOverride(t *testing.T) {
	isolateEnv(t)
	fakeInstall(t, "v1.3.1", "a")
	fakeInstall(t, "v1.4.0", "b")
	SetDefaultVersion("v1.4.0")

	project := filepath.Join(root2(t), "project")
	os.MkdirAll(project, 0o755)
	t.Chdir(project)
	os.WriteFile(filepath.Join(project, rcFileName), []byte("v1.3.1\n"), 0o644)

	// Session override beats the project pin and the default.
	t.Setenv("BVM_VERSION", "v1.4.0")
	version, source, err := ResolveVersion()
	if err != nil || version != "v1.4.0" {
		t.Fatalf("ResolveVersion() = (%q, %v), want session override v1.4.0", version, err)
	}
	if !strings.Contains(source, "session") {
		t.Fatalf("source = %q, want session mention", source)
	}

	// Unprefixed form and "latest" work too.
	t.Setenv("BVM_VERSION", "1.3.1")
	if version, _, _ = ResolveVersion(); version != "v1.3.1" {
		t.Fatalf("unprefixed override = %q", version)
	}
	t.Setenv("BVM_VERSION", "latest")
	if version, _, _ = ResolveVersion(); version != "v1.4.0" {
		t.Fatalf("'latest' override = %q, want highest installed v1.4.0", version)
	}

	// Failures are loud: explicit intent should never silently fall back.
	t.Setenv("BVM_VERSION", "v9.9.9")
	if _, _, err := ResolveVersion(); err == nil {
		t.Fatal("expected error for uninstalled override")
	}
	t.Setenv("BVM_VERSION", "bogus!!")
	if _, _, err := ResolveVersion(); err == nil {
		t.Fatal("expected error for invalid override")
	}

	// "latest" with LocalVersions failure propagates.
	stub(t, &readDir, func(string) ([]os.DirEntry, error) { return nil, fmt.Errorf("injected") })
	t.Setenv("BVM_VERSION", "latest")
	if _, _, err := ResolveVersion(); err == nil {
		t.Fatal("expected LocalVersions failure to propagate")
	}
	t.Setenv("BVM_VERSION", "")

	// "latest" with nothing installed errors.
	isolateEnv(t)
	t.Chdir(t.TempDir())
	t.Setenv("BVM_VERSION", "latest")
	if _, _, err := ResolveVersion(); err == nil {
		t.Fatal("expected latest-with-no-versions error")
	}
}

func TestWriteAndRemoveRC(t *testing.T) {
	if RCFileName() != ".bvmrc" {
		t.Fatalf("RCFileName() = %q", RCFileName())
	}

	t.Chdir(t.TempDir())

	if err := WriteRC("v1.4.0"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(rcFileName)
	if err != nil || string(data) != "v1.4.0\n" {
		t.Fatalf("rc content = %q (%v)", data, err)
	}

	if err := RemoveRC(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(rcFileName); !os.IsNotExist(err) {
		t.Fatal("rc still exists after RemoveRC")
	}
	// Removing again is a no-op.
	if err := RemoveRC(); err != nil {
		t.Fatalf("second RemoveRC() error = %v", err)
	}
}
