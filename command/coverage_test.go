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

// --- install.go ---

func TestInstallHappyPathOffline(t *testing.T) {
	setupCommandEnv(t)
	quietPATH(t)

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
	if active, _ := util.ActiveVersion(); active != "v1.0.0" {
		t.Fatalf("active = %q", active)
	}
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
	t.Run("unresolvable home after resolve", func(t *testing.T) {
		setupCommandEnv(t)
		stub(t, &resolveTarget, func(string) (string, error) { return "v1.0.0", nil })
		breakHomeForCommand(t) // VersionDir fails once past resolution

		if err := Install("1.0.0"); err == nil {
			t.Fatal("expected VersionDir failure")
		}
	})
	t.Run("activate failure mid-flow", func(t *testing.T) {
		setupCommandEnv(t)
		home := os.TempDir() // not used; env-driven below
		_ = home
		stub(t, &resolveTarget, func(string) (string, error) { return "v1.0.0", nil })
		stub(t, &downloadFile, func(_ string, dest string) error {
			return os.WriteFile(dest, zipWithBinary(t, "bin"), 0o644)
		})

		// ~/.bun/bin as a file makes Activate's MkdirAll fail.
		h := filepath.Join(mustBVMDir(t), "..")
		os.MkdirAll(filepath.Join(h, ".bun"), 0o755)
		os.WriteFile(filepath.Join(h, ".bun", "bin"), []byte("x"), 0o644)

		if err := Install("1.0.0"); err == nil || !strings.Contains(err.Error(), "activate") {
			t.Fatalf("expected activate failure, got %v", err)
		}
	})
	t.Run("default-alias warning", func(t *testing.T) {
		setupCommandEnv(t)
		stub(t, &detectPlatformFn, func() (util.Platform, error) {
			return util.Platform{OS: "darwin", Arch: "x64"}, nil
		})
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

// --- use.go ---

func TestUseOutputBranches(t *testing.T) {
	setupCommandEnv(t)
	fakeInstall(t, "v1.0.0")

	// Activate failure path: pin $BVM_DIR so resolution works, then break
	// HOME so BunBinPath fails inside Activate.
	t.Setenv("BVM_DIR", mustBVMDir(t))
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	if err := Use(""); err == nil {
		t.Fatal("expected activate failure when home is unresolvable")
	}

	setupCommandEnv(t)
	fakeInstall(t, "v1.0.0")

	// Profile updated -> changed message branch.
	stub(t, &ensurePATH, func() (bool, error) { return true, nil })
	if err := Use(""); err != nil {
		t.Fatalf("Use() error = %v", err)
	}
	if active, _ := util.ActiveVersion(); active != "v1.0.0" {
		t.Fatalf("active = %q", active)
	}

	// Profile failure -> warning branch (command still succeeds).
	stub(t, &ensurePATH, func() (bool, error) { return false, fmt.Errorf("rc locked") })
	if err := Use(""); err != nil {
		t.Fatalf("profile warnings must not fail Use: %v", err)
	}
}

func TestResolveLocalEnvError(t *testing.T) {
	setupCommandEnv(t)
	breakHomeForCommand(t)
	if _, err := resolveLocal(""); err == nil {
		t.Fatal("expected LocalVersions failure propagation")
	}
}

func breakHomeForCommand(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
}

func TestResolveLocalAllBranches(t *testing.T) {
	setupCommandEnv(t)

	if _, err := resolveLocal(""); err == nil {
		t.Fatal("expected no-versions error")
	}

	fakeInstall(t, "v1.0.0")
	fakeInstall(t, "v2.0.0")

	if got, err := resolveLocal(""); err != nil || got != "v2.0.0" {
		t.Fatalf("fallback to highest = (%q, %v)", got, err)
	}
	if got, err := resolveLocal("latest"); err != nil || got != "v2.0.0" {
		t.Fatalf("'latest' = (%q, %v)", got, err)
	}
	if _, err := resolveLocal("abc!!"); err == nil {
		t.Fatal("expected normalize error")
	}
	if _, err := resolveLocal("v3.0.0"); err == nil {
		t.Fatal("expected not-installed error")
	}

	// default alias arms
	if _, err := resolveLocal("default"); err == nil {
		t.Fatal("expected missing-default error")
	}
	util.SetDefaultVersion("v9.9.9")
	if _, err := resolveLocal("default"); err == nil {
		t.Fatal("expected stale-default error")
	}
	util.SetDefaultVersion("v1.0.0")
	if got, err := resolveLocal("default"); err != nil || got != "v1.0.0" {
		t.Fatalf("'default' = (%q, %v)", got, err)
	}
	if got, err := resolveLocal(""); err != nil || got != "v1.0.0" {
		t.Fatalf("bare fallback should prefer default = (%q, %v)", got, err)
	}
}

// --- list / list-remote ---

func TestListAllBranches(t *testing.T) {
	setupCommandEnv(t)

	// Unresolvable env -> errors from every lookup.
	breakHomeForCommand(t)
	if err := List(); err == nil {
		t.Fatal("expected LocalVersions error path")
	}

	setupCommandEnv(t)
	if err := List(); err != nil {
		t.Fatalf("empty List() error = %v", err)
	}

	// Active read error with versions present.
	fakeInstall(t, "v1.0.0")
	root, _ := util.BVMDir()
	os.MkdirAll(filepath.Join(root, "active"), 0o755)
	if err := List(); err == nil {
		t.Fatal("expected ActiveVersion error path")
	}

	// Marker arms: inactive-default, active-only, both.
	setupCommandEnv(t)
	fakeInstall(t, "v1.0.0")
	fakeInstall(t, "v2.0.0")

	util.SetDefaultVersion("v1.0.0")
	out := captureStdout(t, func() { _ = List() })
	if !strings.Contains(out, "(default)") {
		t.Fatalf("missing default marker:\n%s", out)
	}

	util.Activate("v1.0.0")
	out = captureStdout(t, func() { _ = List() })
	if !strings.Contains(out, "*") || !strings.Contains(out, "(active, default)") {
		t.Fatalf("missing combined markers:\n%s", out)
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

// --- uninstall ---

func TestUninstallNormalizeError(t *testing.T) {
	setupCommandEnv(t)
	if err := Uninstall("garbage!!"); err == nil {
		t.Fatal("expected normalize error")
	}
}

func TestUninstallEnvFailureArms(t *testing.T) {
	t.Run("unresolvable home hits VersionDir", func(t *testing.T) {
		setupCommandEnv(t)
		breakHomeForCommand(t)
		if err := Uninstall("1.0.0"); err == nil {
			t.Fatal("expected VersionDir failure")
		}
	})
	t.Run("unreadable marker hits ActiveVersion", func(t *testing.T) {
		setupCommandEnv(t)
		fakeInstall(t, "v1.0.0")
		root, _ := util.BVMDir()
		os.MkdirAll(filepath.Join(root, "active"), 0o755)

		if err := Uninstall("1.0.0"); err == nil {
			t.Fatal("expected ActiveVersion failure")
		}
	})
}

func TestUninstallRemoveFails(t *testing.T) {
	setupCommandEnv(t)
	fakeInstall(t, "v1.0.0")

	stub(t, &util.RemoveAll, func(string) error { return fmt.Errorf("injected") })
	if err := Uninstall("1.0.0"); err == nil {
		t.Fatal("expected RemoveAll error")
	}
}

// --- alias remote paths ---

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

// --- doctor ---

func TestDoctorAllChecks(t *testing.T) {
	setupCommandEnv(t)
	stub(t, &probeAPI, func() (bool, string) { return true, "https://api.example.com" })

	// Fresh env: covers the "nothing active/default yet" arms; binary-link
	// and PATH checks fail, so an error is expected.
	if err := Doctor(); err == nil {
		t.Fatal("fresh environment should report failures")
	}

	quietPATH(t)
	binDir, _ := util.BunBinDir()
	os.MkdirAll(binDir, 0o755)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	fakeInstall(t, "v1.0.0")
	util.Activate("v1.0.0")
	util.SetDefaultVersion("v1.0.0")
	if err := Doctor(); err != nil {
		t.Fatalf("healthy Doctor() error = %v", err)
	}

	// Failing API probe alone flips the result.
	stub(t, &probeAPI, func() (bool, string) { return false, "refused" })
	if err := Doctor(); err == nil {
		t.Fatal("expected failures to produce error")
	}
	stub(t, &probeAPI, func() (bool, string) { return true, "https://api.example.com" })

	// Stale active + stale default details.
	os.Remove(filepath.Join(mustBVMDir(t), "versions", "v1.0.0", util.BinaryName()))
	util.SetDefaultVersion("v8.8.8")
	if err := Doctor(); err == nil {
		t.Fatal("expected stale-marker failures")
	}

	// Unresolvable home -> early check failures.
	breakHomeForCommand(t)
	if err := Doctor(); err == nil {
		t.Fatal("expected broken-home failures")
	}
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

func mustBVMDir(t *testing.T) string {
	t.Helper()
	root, err := util.BVMDir()
	if err != nil {
		t.Fatal(err)
	}
	return root
}
