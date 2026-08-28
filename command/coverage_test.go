package command

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chathula/bvm/util"
)

// stub swaps a package-level seam for the duration of a test.
func stub[T any](t *testing.T, target *T, value T) {
	t.Helper()
	old := *target
	*target = value
	t.Cleanup(func() { *target = old })
}

// zipWithBinary builds a release-style archive containing one nested binary.
func zipWithBinary(t *testing.T, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	w.Create("pkg/") // directory entry
	entry, err := w.Create("pkg/" + util.BinaryName())
	if err != nil {
		t.Fatal(err)
	}
	entry.Write([]byte(content))
	w.Close()
	return buf.Bytes()
}

// quietPATH keeps the real EnsurePATH from touching actual shell profiles.
func quietPATH(t *testing.T) {
	t.Helper()
	stub(t, &ensurePATH, func() (bool, error) { return false, nil })
}

// quietShim keeps EnsureShim from linking the test binary as the shim.
func quietShim(t *testing.T) {
	t.Helper()
	stub(t, &ensureShim, func() error { return nil })
}

func TestInstallHappyPathOffline(t *testing.T) {
	setupCommandEnv(t)
	quietPATH(t)
	quietShim(t)

	stub(t, &detectPlatformFn, func() (util.Platform, error) {
		return util.Platform{OS: "darwin", Arch: "x64"}, nil
	})
	stub(t, &resolveTarget, func(string) (string, error) { return "v1.0.0", nil })
	stub(t, &downloadFile, func(_ string, dest string) error {
		return os.WriteFile(dest, zipWithBinary(t, "binary-v1"), 0o644)
	})

	if err := Install("1.0.0"); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if !util.IsInstalled("v1.0.0") {
		t.Fatal("version not installed")
	}
	// First install becomes the default (nvm semantics).
	if def, _ := util.DefaultVersion(); def != "v1.0.0" {
		t.Fatalf("first install must set default, got %q", def)
	}
	// Archive cleaned up after extraction.
	dir, _ := util.VersionDir("v1.0.0")
	if _, err := os.Stat(filepath.Join(dir, "bun-darwin-x64.zip")); !os.IsNotExist(err) {
		t.Fatal("archive was not removed")
	}
}

func TestInstallSecondTimeKeepsDefault(t *testing.T) {
	setupCommandEnv(t)
	quietPATH(t)
	quietShim(t)
	fakeInstall(t, "v1.0.0")
	util.SetDefaultVersion("v1.0.0")

	stub(t, &detectPlatformFn, func() (util.Platform, error) {
		return util.Platform{OS: "darwin", Arch: "x64"}, nil
	})
	stub(t, &resolveTarget, func(string) (string, error) { return "v2.0.0", nil })
	stub(t, &downloadFile, func(_ string, dest string) error {
		return os.WriteFile(dest, zipWithBinary(t, "binary-v2"), 0o644)
	})

	if err := Install("latest"); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if def, _ := util.DefaultVersion(); def != "v1.0.0" {
		t.Fatalf("default stolen by second install: %q", def)
	}
}

func TestInstallFailureBranches(t *testing.T) {
	t.Run("no arg and no rc file", func(t *testing.T) {
		setupCommandEnv(t)
		t.Chdir(t.TempDir())
		if err := Install(""); err == nil {
			t.Fatal("expected error for missing version")
		}
	})
	t.Run("unsupported platform", func(t *testing.T) {
		setupCommandEnv(t)
		stub(t, &detectPlatformFn, func() (util.Platform, error) {
			return util.Platform{}, fmt.Errorf("nope")
		})
		if err := Install("1.0.0"); err == nil {
			t.Fatal("expected platform error")
		}
	})
	t.Run("resolve target fails", func(t *testing.T) {
		setupCommandEnv(t)
		stub(t, &resolveTarget, func(string) (string, error) { return "", fmt.Errorf("boom") })
		if err := Install("1.0.0"); err == nil {
			t.Fatal("expected resolve error")
		}
	})
	t.Run("already installed", func(t *testing.T) {
		setupCommandEnv(t)
		fakeInstall(t, "v1.0.0")
		stub(t, &resolveTarget, func(string) (string, error) { return "v1.0.0", nil })
		if err := Install("1.0.0"); err != nil {
			t.Fatalf("already-installed path errored: %v", err)
		}
	})
	t.Run("download failure cleans up", func(t *testing.T) {
		setupCommandEnv(t)
		stub(t, &resolveTarget, func(string) (string, error) { return "v1.0.0", nil })
		stub(t, &downloadFile, func(string, string) error { return fmt.Errorf("net down") })

		if err := Install("1.0.0"); err == nil {
			t.Fatal("expected download error")
		}
		if util.IsInstalled("v1.0.0") {
			t.Fatal("version dir left behind after failed download")
		}
	})
	t.Run("bad archive cleans up", func(t *testing.T) {
		setupCommandEnv(t)
		stub(t, &resolveTarget, func(string) (string, error) { return "v1.0.0", nil })
		stub(t, &downloadFile, func(_ string, dest string) error {
			return os.WriteFile(dest, []byte("garbage"), 0o644)
		})

		if err := Install("1.0.0"); err == nil {
			t.Fatal("expected extract error")
		}
		if util.IsInstalled("v1.0.0") {
			t.Fatal("version dir left behind after failed extraction")
		}
	})
	t.Run("bvm dir unusable", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("BVM_DIR", filepath.Join(root, "file"))
		os.WriteFile(filepath.Join(root, "file"), []byte("x"), 0o644)
		stub(t, &resolveTarget, func(string) (string, error) { return "v1.0.0", nil })

		if err := Install("1.0.0"); err == nil {
			t.Fatal("expected mkdir failure under BVM_DIR-as-file")
		}
	})
	t.Run("shim failure surfaces", func(t *testing.T) {
		setupCommandEnv(t)
		quietPATH(t)
		stub(t, &resolveTarget, func(string) (string, error) { return "v1.0.0", nil })
		stub(t, &downloadFile, func(_ string, dest string) error {
			return os.WriteFile(dest, zipWithBinary(t, "bin"), 0o644)
		})
		stub(t, &ensureShim, func() error { return fmt.Errorf("no symlink rights") })

		if err := Install("1.0.0"); err == nil || !strings.Contains(err.Error(), "shim") {
			t.Fatalf("expected shim failure, got %v", err)
		}
	})
	t.Run("default-alias warning", func(t *testing.T) {
		setupCommandEnv(t)
		quietPATH(t)
		quietShim(t)
		stub(t, &resolveTarget, func(string) (string, error) { return "v1.0.0", nil })
		stub(t, &downloadFile, func(_ string, dest string) error {
			return os.WriteFile(dest, zipWithBinary(t, "bin"), 0o644)
		})
		stub(t, &ensureDefaultSet, func(string) error { return fmt.Errorf("disk full") })

		if err := Install("1.0.0"); err != nil {
			t.Fatalf("Install should succeed despite alias warning: %v", err)
		}
	})
	t.Run("profile warning and changed message", func(t *testing.T) {
		setupCommandEnv(t)
		quietShim(t)
		stub(t, &resolveTarget, func(string) (string, error) { return "v1.0.0", nil })
		stub(t, &downloadFile, func(_ string, dest string) error {
			return os.WriteFile(dest, zipWithBinary(t, "bin"), 0o644)
		})

		// Profile update fails -> warning branch.
		stub(t, &ensurePATH, func() (bool, error) { return false, fmt.Errorf("rc locked") })
		if err := Install("1.0.0"); err != nil {
			t.Fatalf("Install should succeed despite profile warning: %v", err)
		}

		// Profile updated -> changed message.
		stub(t, &ensurePATH, func() (bool, error) { return true, nil })
		stub(t, &resolveTarget, func(string) (string, error) { return "v2.0.0", nil })
		if err := Install("2.0.0"); err != nil {
			t.Fatalf("Install() error = %v", err)
		}
	})
}

func TestDownloadErrorBranches(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "out.bin")

	if err := download("://bad", dest); err == nil {
		t.Fatal("expected request-build error")
	}

	srv404 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv404.Close()
	if err := download(srv404.URL, dest); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 404-specific error, got %v", err)
	}

	srv500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv500.Close()
	if err := download(srv500.URL, dest); err == nil {
		t.Fatal("expected 500 error")
	}

	srvOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "payload")
	}))
	defer srvOK.Close()

	srvAbort := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.(http.Flusher).Flush()
		fmt.Fprint(w, "short")
		panic("drop")
	}))
	defer srvAbort.Close()
	if err := download(srvAbort.URL, dest); err == nil {
		t.Fatal("expected mid-body copy error")
	}
	if _, err := os.Stat(dest + ".part"); !os.IsNotExist(err) {
		t.Fatal(".part file left behind after failed copy")
	}

	// Destination's temp file pre-occupied by a directory fails everywhere.
	occupied := filepath.Join(t.TempDir(), "out")
	os.MkdirAll(occupied+".part", 0o755)
	if err := download(srvOK.URL, occupied); err == nil {
		t.Fatal("expected create error for directory-occupied .part path")
	}

	if err := download(srvOK.URL, dest); err != nil {
		t.Fatalf("happy-path download error = %v", err)
	}
	if data, _ := os.ReadFile(dest); string(data) != "payload" {
		t.Fatal("downloaded content mismatch")
	}
}

func TestEnsureDefaultSetError(t *testing.T) {
	root := t.TempDir()
	t.Setenv("BVM_DIR", root)

	// First call succeeds and writes the marker.
	if err := ensureDefaultSet("v1.0.0"); err != nil {
		t.Fatalf("ensureDefaultSet() error = %v", err)
	}
	// Point BVM_DIR at an unwritable location for the error arm.
	t.Setenv("BVM_DIR", filepath.Join(root, "as-file"))
	os.WriteFile(filepath.Join(root, "as-file"), []byte("x"), 0o644)
	if err := ensureDefaultSet("v9.9.9"); err == nil {
		t.Fatal("expected write error")
	}
}

func TestUsePinsCurrentDirectory(t *testing.T) {
	setupCommandEnv(t)
	quietShim(t)
	fakeInstall(t, "v1.3.1")
	fakeInstall(t, "v1.4.0")
	util.SetDefaultVersion("v1.4.0")

	project := t.TempDir()
	t.Chdir(project)

	// Plain use never creates the pin file — .bvmrc is strictly opt-in.
	if err := Use(false, false, "1.3.1"); err != nil {
		t.Fatalf("Use(1.3.1) error = %v", err)
	}
	if path, _ := util.FindRCHere(); path != "" {
		t.Fatal(".bvmrc created without --save")
	}

	// Pin the project to v1.3.1 with --save.
	if err := Use(false, true, "1.3.1"); err != nil {
		t.Fatalf("Use(--save, 1.3.1) error = %v", err)
	}
	if _, v := util.FindRCHere(); v != "v1.3.1" {
		t.Fatalf(".bvmrc pin = %q, want v1.3.1", v)
	}

	// No-arg reports the pin.
	out := captureStdout(t, func() { _ = Use(false, false, "") })
	if !strings.Contains(out, "v1.3.1") {
		t.Fatalf("resolution output missing version:\n%s", out)
	}

	// 'default' clears the pin.
	if err := Use(false, false, "default"); err != nil {
		t.Fatalf("Use(default) error = %v", err)
	}
	if path, _ := util.FindRCHere(); path != "" {
		t.Fatal(".bvmrc still present after 'use default'")
	}

	// Uninstalled version errors.
	if err := Use(false, false, "v9.9.9"); err == nil {
		t.Fatal("expected error for uninstalled version")
	}
}

func TestExecRunsSpecificVersion(t *testing.T) {
	setupCommandEnv(t)
	fakeInstall(t, "v1.1.0")
	fakeInstall(t, "v1.4.0")

	var gotBin string
	var gotArgs []string
	stub(t, &execProcessFn, func(bin string, args []string) error {
		gotBin = bin
		gotArgs = args
		return nil
	})

	// Leading "bun" is tolerated and stripped.
	if err := Exec("1.1.0", []string{"bun", "test"}); err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	want := filepath.Join(mustBVMDir(t), "versions", "v1.1.0", util.BinaryName())
	if gotBin != want {
		t.Fatalf("exec target = %q, want %q", gotBin, want)
	}
	if len(gotArgs) != 1 || gotArgs[0] != "test" {
		t.Fatalf("exec args = %v, want [test]", gotArgs)
	}

	if err := Exec("1.4.0", nil); err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	if gotBin != filepath.Join(mustBVMDir(t), "versions", "v1.4.0", util.BinaryName()) {
		t.Fatalf("exec target = %q", gotBin)
	}

	if err := Exec("v9.9.9", nil); err == nil {
		t.Fatal("expected error for uninstalled version")
	}
	if err := Exec("bogus!!", nil); err == nil {
		t.Fatal("expected normalize error")
	}
}

func TestUseResetGuidance(t *testing.T) {
	setupCommandEnv(t)
	out := captureStdout(t, func() { _ = Use(true, false, "--reset") })
	if !strings.Contains(out, "BVM_VERSION") {
		t.Fatalf("expected unset guidance, got:\n%s", out)
	}
}

func TestUseBranches(t *testing.T) {
	setupCommandEnv(t)
	quietShim(t)
	fakeInstall(t, "v1.0.0")

	t.Run("invalid version", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if err := Use(false, false, "bogus!!"); err == nil {
			t.Fatal("expected normalize error")
		}
	})
	t.Run("use default without pin", func(t *testing.T) {
		t.Chdir(t.TempDir())
		out := captureStdout(t, func() { _ = Use(false, false, "default") })
		if !strings.Contains(out, "already follows the default") {
			t.Fatalf("unexpected output:\n%s", out)
		}
	})
	t.Run("write rc failure", func(t *testing.T) {
		t.Chdir(t.TempDir())
		os.MkdirAll(filepath.Join(t.TempDir(), "placeholder"), 0o755)
		os.MkdirAll(filepath.Join(".", rcFileNameAsDir()), 0o755) // .bvmrc as directory
		if err := Use(false, true, "v1.0.0"); err == nil {
			t.Fatal("expected WriteRC failure")
		}
	})
	t.Run("remove rc failure", func(t *testing.T) {
		t.Chdir(t.TempDir())
		os.MkdirAll(filepath.Join(".", rcFileNameAsDir(), "keep"), 0o755) // non-empty dir
		if err := Use(false, false, "default"); err == nil {
			t.Fatal("expected RemoveRC failure")
		}
	})
	t.Run("shim warning", func(t *testing.T) {
		t.Chdir(t.TempDir())
		stub(t, &ensureShim, func() error { return fmt.Errorf("no rights") })
		out := captureStdout(t, func() { _ = Use(false, true, "v1.0.0") })
		if !strings.Contains(out, "shim") {
			t.Fatalf("expected shim warning, got:\n%s", out)
		}
	})
	t.Run("resolution failure", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if err := Use(false, false, ""); err == nil {
			t.Fatal("expected resolution failure with nothing installed")
		}
	})
}

// rcFileNameAsDir is a helper name so tests can occupy the .bvmrc path with
// a directory without importing util.
func rcFileNameAsDir() string { return ".bvmrc" }

func TestListMarksDefaultAndProjectPin(t *testing.T) {
	setupCommandEnv(t)
	fakeInstall(t, "v1.3.1")
	fakeInstall(t, "v1.4.0")
	util.SetDefaultVersion("v1.4.0")

	project := t.TempDir()
	t.Chdir(project)
	util.WriteRC("v1.3.1")

	out := captureStdout(t, func() { _ = List() })
	if !strings.Contains(out, "default") || !strings.Contains(out, "this project") {
		t.Fatalf("markers missing in:\n%s", out)
	}

	// Empty env prints the hint without error.
	setupCommandEnv(t)
	out = captureStdout(t, func() { _ = List() })
	if !strings.Contains(out, "no versions installed") {
		t.Fatalf("empty-list hint missing:\n%s", out)
	}
}

func TestListEnvError(t *testing.T) {
	setupCommandEnv(t)
	breakHomeForCommand(t)
	if err := List(); err == nil {
		t.Fatal("expected LocalVersions error path")
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestListRemoteBranches(t *testing.T) {
	stub(t, &remoteVersions, func() ([]string, error) { return nil, fmt.Errorf("api down") })
	if err := ListRemote(); err == nil {
		t.Fatal("expected error propagation")
	}

	stub(t, &remoteVersions, func() ([]string, error) {
		return []string{"v1.0.0", "v2.0.0"}, nil
	})
	out := captureStdout(t, func() { _ = ListRemote() })
	if !strings.Contains(out, "v2.0.0 (latest)") {
		t.Fatalf("missing latest marker:\n%s", out)
	}
}

func TestUninstallRemoveFails(t *testing.T) {
	setupCommandEnv(t)
	fakeInstall(t, "v1.0.0")

	stub(t, &util.RemoveAll, func(string) error { return fmt.Errorf("injected") })
	if err := Uninstall("1.0.0"); err == nil {
		t.Fatal("expected RemoveAll error")
	}
}

func TestDoctorReportsChecks(t *testing.T) {
	setupCommandEnv(t)
	stub(t, &probeAPI, func() (bool, string) { return true, "https://api.example.com" })

	// Fresh env: several checks fail, error expected, output non-empty.
	out := captureStdout(t, func() { _ = Doctor() })
	if out == "" {
		t.Fatal("expected progressive output")
	}
	if err := Doctor(); err == nil {
		t.Fatal("fresh environment should report failures")
	}

	// Healthy env: installed, defaulted, shim on PATH, API reachable.
	// Chdir away from the repo so walk-up resolution can't trip over a
	// developer's own .bvmrc.
	t.Chdir(t.TempDir())
	binDir, _ := util.BunBinDir()
	os.MkdirAll(binDir, 0o755)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	fakeInstall(t, "v1.0.0")
	util.SetDefaultVersion("v1.0.0")
	quietShim(t)
	if err := EnsureShimForTest(t); err != nil {
		t.Skipf("cannot create shim here: %v", err)
	}
	out = captureStdout(t, func() { _ = Doctor() })
	if err := Doctor(); err != nil {
		t.Fatalf("healthy Doctor() error = %v\noutput:\n%s", err, out)
	}

	// Failing API probe alone flips the result.
	stub(t, &probeAPI, func() (bool, string) { return false, "refused" })
	if err := Doctor(); err == nil {
		t.Fatal("expected failures to produce error")
	}

	// Broken home -> early check failures.
	breakHomeForCommand(t)
	if err := Doctor(); err == nil {
		t.Fatal("expected broken-home failures")
	}
}

// EnsureShimForTest creates the real shim using the test binary.
func EnsureShimForTest(t *testing.T) error {
	t.Helper()
	return util.EnsureShim()
}

func TestPathHint(t *testing.T) {
	if got := pathHint("darwin", false, "/bin"); got != "" {
		t.Fatalf("unix hint = %q, want empty", got)
	}
	if got := pathHint("windows", true, "/bin"); got != "" {
		t.Fatalf("on-path hint = %q, want empty", got)
	}
	want := "add '/bin' to your PATH manually"
	if got := pathHint("windows", false, "/bin"); got != want {
		t.Fatalf("hint = %q, want %q", got, want)
	}
}

func TestLinkDetail(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "real-bin")
	os.WriteFile(src, []byte("x"), 0o755)

	// Windows binaries are copies — never a link suffix, regardless of host.
	if got := linkDetail("windows", src, true); got != "" {
		t.Fatalf("windows detail = %q", got)
	}
	if got := linkDetail("windows", src, false); got != "" {
		t.Fatalf("windows missing detail = %q", got)
	}

	// Missing binary -> no suffix on any OS.
	if got := linkDetail("darwin", src, false); got != "" {
		t.Fatalf("missing-binary detail = %q", got)
	}

	// Regular files have no link target; symlinks report theirs. Both arms
	// are exercised on every OS because goos is just a parameter here.
	if got := linkDetail("darwin", src, true); got != "" {
		t.Fatalf("regular-file detail = %q", got)
	}
	link := filepath.Join(tmp, "linked")
	if err := os.Symlink(src, link); err != nil {
		t.Fatalf("symlink creation failed: %v", err)
	}
	want := " -> " + src
	if got := linkDetail("darwin", link, true); got != want {
		t.Fatalf("link detail = %q, want %q", got, want)
	}
}
