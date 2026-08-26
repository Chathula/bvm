package util

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// breakHome makes os.UserHomeDir fail on every platform.
func breakHome(t *testing.T) {
	t.Helper()
	t.Setenv("BVM_DIR", "")
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
}

// readOnly marks a path read-only for the duration of the test.
func readOnly(t *testing.T, path string) {
	t.Helper()
	if err := os.Chmod(path, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(path, 0o755) })
}

// --- paths.go ---

func TestPathHelpersFailWithoutHome(t *testing.T) {
	breakHome(t)

	for name, fn := range map[string]func() (string, error){
		"BVMDir":         BVMDir,
		"VersionsDir":    VersionsDir,
		"VersionDir":     func() (string, error) { return VersionDir("v1.0.0") },
		"BunBinPath":     BunBinPath,
		"BunBinDir":      BunBinDir,
		"ActiveVersion":  ActiveVersion,
		"DefaultVersion": DefaultVersion,
		"LocalVersions":  func() (string, error) { _, err := LocalVersions(); return "", err },
		"recordActive":   func() (string, error) { return "", recordActive("v1.0.0") },
		"SetDefault":     func() (string, error) { return "", SetDefaultVersion("v1.0.0") },
		"Deactivate":     func() (string, error) { return "", Deactivate() },
		"EnsurePATH":     func() (string, error) { _, err := EnsurePATH(); return "", err },
		"Activate":       func() (string, error) { return "", Activate("v1.0.0") },
	} {
		if _, err := fn(); err == nil {
			t.Errorf("%s expected error with unresolvable home", name)
		}
	}
	if IsInstalled("v1.0.0") {
		t.Error("IsInstalled should be false when home is unresolvable")
	}
}

// bvmDirValidHomeBroken covers functions whose later steps need a home even
// though $BVM_DIR short-circuits the first lookup.
func TestBvmDirValidHomeBroken(t *testing.T) {
	root := t.TempDir()
	t.Setenv("BVM_DIR", root)
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")

	fakeInstall(t, "v1.0.0", "x")
	os.MkdirAll(filepath.Join(root, ".bvm"), 0o755)

	// Activate: BunBinPath fails after VersionDir succeeds.
	if err := Activate("v1.0.0"); err == nil {
		t.Fatal("expected Activate to fail resolving ~/.bun/bin/bun")
	}
	// Deactivate: BunBinPath fails after BVMDir succeeds.
	if err := Deactivate(); err == nil {
		t.Fatal("expected Deactivate to fail resolving ~/.bun/bin/bun")
	}
}

func TestDeactivateNothingInstalledIsNoop(t *testing.T) {
	isolateEnv(t)
	if err := Deactivate(); err != nil {
		t.Fatalf("Deactivate() on clean env error = %v", err)
	}
}

func TestActivateSymlinkFailure(t *testing.T) {
	home := isolateEnv(t)
	fakeInstall(t, "v1.0.0", "x")

	binDir := filepath.Join(home, ".bun", "bin")
	os.MkdirAll(binDir, 0o755)
	readOnly(t, binDir) // no dst present -> RemoveAll nil, Symlink fails

	if err := Activate("v1.0.0"); err == nil {
		t.Fatal("expected symlink creation error in readonly bin dir")
	}
}

func TestLocalVersionsMissingDirYieldsEmpty(t *testing.T) {
	isolateEnv(t)
	versions, err := LocalVersions()
	if err != nil || versions != nil {
		t.Fatalf("LocalVersions() = (%v, %v), want (nil, nil)", versions, err)
	}
}

func TestLocalVersionsReadError(t *testing.T) {
	root := isolateEnv(t)
	versionsPath := filepath.Join(root, ".bvm", "versions")
	os.MkdirAll(filepath.Dir(versionsPath), 0o755)
	os.WriteFile(versionsPath, []byte("not a dir"), 0o644)

	if _, err := LocalVersions(); err == nil {
		t.Fatal("expected error when versions path is a file")
	}
}

// --- alias.go ---

func TestDefaultVersionReadError(t *testing.T) {
	root := isolateEnv(t)
	os.MkdirAll(filepath.Join(root, ".bvm", "default"), 0o755) // marker as directory

	if _, err := DefaultVersion(); err == nil {
		t.Fatal("expected error when default marker is unreadable")
	}
}

func TestSetDefaultVersionErrors(t *testing.T) {
	t.Run("root is a file", func(t *testing.T) {
		root := isolateEnv(t)
		os.WriteFile(filepath.Join(root, "bvmroot"), []byte("x"), 0o644)
		t.Setenv("BVM_DIR", filepath.Join(root, "bvmroot"))

		if err := SetDefaultVersion("v1.0.0"); err == nil {
			t.Fatal("expected MkdirAll error")
		}
	})
	t.Run("readonly root", func(t *testing.T) {
		root := isolateEnv(t)
		readOnly(t, root)

		if err := SetDefaultVersion("v1.0.0"); err == nil {
			t.Fatal("expected WriteFile error")
		}
	})
}

// --- activate.go ---

func TestActiveVersionReadError(t *testing.T) {
	root := isolateEnv(t)
	os.MkdirAll(filepath.Join(root, ".bvm", activeMarker), 0o755)

	if _, err := ActiveVersion(); err == nil {
		t.Fatal("expected error when active marker is a directory")
	}
}

func TestActivateMkdirBinDirError(t *testing.T) {
	home := isolateEnv(t)
	fakeInstall(t, "v1.0.0", "x")

	os.MkdirAll(filepath.Join(home, ".bun"), 0o755)
	os.WriteFile(filepath.Join(home, ".bun", "bin"), []byte("file"), 0o644)
	if err := Activate("v1.0.0"); err == nil {
		t.Fatal("expected MkdirAll error when bun bin dir is a file")
	}
}

func TestActivateRemoveOldBinaryError(t *testing.T) {
	home := isolateEnv(t)
	fakeInstall(t, "v1.0.0", "x")

	binDir := filepath.Join(home, ".bun", "bin")
	os.MkdirAll(binDir, 0o755)
	os.WriteFile(filepath.Join(binDir, BinaryName()), []byte("old"), 0o755)
	readOnly(t, binDir)

	if err := Activate("v1.0.0"); err == nil {
		t.Fatal("expected RemoveAll error in readonly bin dir")
	}
}

func TestRecordActiveErrors(t *testing.T) {
	t.Run("root is a file", func(t *testing.T) {
		root := isolateEnv(t)
		t.Setenv("BVM_DIR", filepath.Join(root, "as-file"))
		os.WriteFile(filepath.Join(root, "as-file"), []byte("x"), 0o644)

		if err := recordActive("v1.0.0"); err == nil {
			t.Fatal("expected MkdirAll error")
		}
	})
	t.Run("readonly root", func(t *testing.T) {
		root := isolateEnv(t)
		readOnly(t, root)

		if err := recordActive("v1.0.0"); err == nil {
			t.Fatal("expected WriteFile error")
		}
	})
}

func TestExposeBinaryBothModes(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src-bin")
	os.WriteFile(src, []byte("payload"), 0o755)

	dst := filepath.Join(tmp, "dst-copy")
	if err := exposeBinary(src, dst, false); err != nil {
		t.Fatalf("copy mode error = %v", err)
	}
	if data, _ := os.ReadFile(dst); string(data) != "payload" {
		t.Fatal("copy mode produced wrong content")
	}

	dstLink := filepath.Join(tmp, "dst-link")
	if err := exposeBinary(src, dstLink, true); err != nil {
		t.Fatalf("symlink mode error = %v", err)
	}
	if target, err := os.Readlink(dstLink); err != nil || target != src {
		t.Fatalf("symlink mode target = %q (%v)", target, err)
	}
}

func TestCopyFileErrors(t *testing.T) {
	tmp := t.TempDir()

	if err := copyFile(filepath.Join(tmp, "missing"), filepath.Join(tmp, "out")); err == nil {
		t.Fatal("expected open error for missing source")
	}

	// io.Copy from a directory fd fails with EISDIR.
	out2 := filepath.Join(tmp, "out2")
	if err := copyFile(tmp, out2); err == nil {
		t.Fatal("expected copy error when source is a directory")
	}

	readonlyDst := filepath.Join(tmp, "ro", "out3")
	os.MkdirAll(filepath.Dir(readonlyDst), 0o755)
	readOnly(t, filepath.Dir(readonlyDst))
	if err := copyFile(tmp, readonlyDst); err == nil {
		t.Fatal("expected create error in readonly destination dir")
	}
}

func TestDeactivateErrorBranches(t *testing.T) {
	breakHome(t)
	if err := Deactivate(); err == nil {
		t.Fatal("expected error with unresolvable home")
	}

	home := isolateEnv(t)
	binDir := filepath.Join(home, ".bun", "bin")
	os.MkdirAll(binDir, 0o755)
	os.WriteFile(filepath.Join(binDir, BinaryName()), []byte("x"), 0o755)
	readOnly(t, binDir)
	if err := Deactivate(); err == nil {
		t.Fatal("expected RemoveAll error in readonly bin dir")
	}

	isolateEnv(t)
	root, _ := BVMDir()
	os.MkdirAll(root, 0o755)
	os.WriteFile(filepath.Join(root, activeMarker), []byte("v1.0.0"), 0o644)
	readOnly(t, root)
	if err := Deactivate(); err == nil {
		t.Fatal("expected marker remove error in readonly root")
	}
}

// --- platform.go ---

func TestNewPlatformUnsupportedCombinations(t *testing.T) {
	if _, err := NewPlatform("darwin", "386"); err == nil {
		t.Fatal("expected unsupported arch error")
	}
	if _, err := NewPlatform("plan9", "amd64"); err == nil {
		t.Fatal("expected unsupported OS error")
	}
	if _, err := DetectPlatform(); err != nil {
		t.Fatalf("DetectPlatform on host failed: %v", err)
	}
}

// --- versions.go ---

func TestProbeReleases(t *testing.T) {
	t.Run("reachable", func(t *testing.T) {
		setReleasesAPIURL(t, httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `[{"name":"bun-v1.0.0"}]`)
		})).URL)

		ok, detail := ProbeReleases()
		if !ok || detail == "" {
			t.Fatalf("ProbeReleases() = (%v, %q)", ok, detail)
		}
	})
	t.Run("unreachable", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		setReleasesAPIURL(t, srv.URL)
		srv.Close()

		ok, _ := ProbeReleases()
		if ok {
			t.Fatal("expected unreachable API")
		}
	})
}

func TestAPIGetRequestBuildError(t *testing.T) {
	if _, err := APIGet("://missing-scheme"); err == nil {
		t.Fatal("expected NewRequest error for malformed URL")
	}
}

func TestAPIGetTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // refuse connections

	if _, err := APIGet(url); err == nil {
		t.Fatal("expected transport error against closed server")
	}
}

func TestRemoteVersionsErrorPaths(t *testing.T) {
	t.Run("transport failure", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		setReleasesAPIURL(t, srv.URL)
		srv.Close()

		if _, err := RemoteVersions(); err == nil {
			t.Fatal("expected request failure")
		}
	})
	t.Run("non-200 status", func(t *testing.T) {
		setReleasesAPIURL(t, httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})).URL)

		if _, err := RemoteVersions(); err == nil {
			t.Fatal("expected status error")
		}
	})
	t.Run("invalid json", func(t *testing.T) {
		setReleasesAPIURL(t, httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "definitely not json")
		})).URL)

		if _, err := RemoteVersions(); err == nil {
			t.Fatal("expected json parse error")
		}
	})
	t.Run("body read interrupted", func(t *testing.T) {
		// Lie about Content-Length, then drop the connection mid-body.
		setReleasesAPIURL(t, httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "1000")
			w.(http.Flusher).Flush()
			fmt.Fprint(w, "short")
			panic("connection dropped")
		})).URL)

		if _, err := RemoteVersions(); err == nil {
			t.Fatal("expected body read error")
		}
	})
}

func TestValidRemoteVersionFetchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	setReleasesAPIURL(t, srv.URL)
	srv.Close()

	if err := ValidRemoteVersion("v1.0.0"); err == nil {
		t.Fatal("expected fetch error propagation")
	}
}

func TestResolveTargetEdgeCases(t *testing.T) {
	setReleasesAPIURL(t, mockTagsAPI(t, nil).URL) // empty remote list

	if _, err := ResolveTarget("latest"); err == nil {
		t.Fatal("expected 'no remote versions' error")
	}

	setReleasesAPIURL(t, mockTagsAPI(t, []string{"bun-v1.0.0"}).URL)
	got, err := ResolveTarget(" latest ")
	if err != nil || got != "v1.0.0" {
		t.Fatalf("ResolveTarget trimmed = (%q, %v)", got, err)
	}
}

// --- pathenv.go ---

func TestShellProfilesByOS(t *testing.T) {
	if got := shellProfiles("windows", "/h", "/bin"); got != nil {
		t.Fatalf("windows profiles = %v, want nil", got)
	}
	profiles := shellProfiles("linux", "/h", "/bin")
	if len(profiles) != 3 {
		t.Fatalf("unix profiles = %d, want 3", len(profiles))
	}
	if profiles[2].shell != "fish" || profiles[2].line != "fish_add_path /bin" {
		t.Fatalf("unexpected fish profile: %+v", profiles[2])
	}
}

func TestAppendLineIfMissingErrorBranches(t *testing.T) {
	tmp := t.TempDir()

	// Read error other than NotExist: target path is a directory.
	dirPath := filepath.Join(tmp, "im-a-dir")
	os.MkdirAll(dirPath, 0o755)
	if _, err := appendLineIfMissing(dirPath, "line"); err == nil {
		t.Fatal("expected read error for directory target")
	}

	// MkdirAll error: target lives under a read-only directory, so ReadFile
	// yields NotExist but creating the parent fails.
	roParent := filepath.Join(tmp, "ro")
	os.MkdirAll(roParent, 0o755)
	readOnly(t, roParent)
	if _, err := appendLineIfMissing(filepath.Join(roParent, "sub", "rc"), "line"); err == nil {
		t.Fatal("expected mkdir error under readonly parent")
	}

	// OpenFile error: target directory is read-only.
	roDir := filepath.Join(tmp, "ro")
	os.MkdirAll(roDir, 0o755)
	readOnly(t, roDir)
	if _, err := appendLineIfMissing(filepath.Join(roDir, "rc"), "line"); err == nil {
		t.Fatal("expected open error in readonly dir")
	}
}

func TestEnsurePATHPropagatesProfileError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("profile editing is a no-op on Windows")
	}
	home := isolateEnv(t)

	fakeBin := filepath.Join(home, "fakebin")
	os.MkdirAll(fakeBin, 0o755)
	os.WriteFile(filepath.Join(fakeBin, "zsh"), []byte("#!/bin/sh\n"), 0o755)
	os.WriteFile(filepath.Join(fakeBin, "bash"), []byte("#!/bin/sh\n"), 0o755)
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	// zsh appends fine; bash's rc is a directory -> hard error mid-loop.
	os.WriteFile(filepath.Join(home, ".zshrc"), []byte(""), 0o644)
	os.MkdirAll(filepath.Join(home, ".bashrc"), 0o755)

	changed, err := EnsurePATH()
	if err == nil {
		t.Fatal("expected profile error propagation")
	}
	if !changed {
		t.Fatal("expected changed=true for the profile written before the error")
	}
}

func TestPathContainsExactMatchOnly(t *testing.T) {
	t.Setenv("PATH", "/a:/b:/c")
	if !PathContains("/b") {
		t.Fatal("/b should be found")
	}
	if PathContains("/bb") {
		t.Fatal("/bb must not match /b")
	}
}

// --- archive.go ---

func TestExtractBunBinaryErrorPaths(t *testing.T) {
	binary := BinaryNameForOS(runtime.GOOS)

	t.Run("unreadable archive", func(t *testing.T) {
		bad := filepath.Join(t.TempDir(), "bad.zip")
		os.WriteFile(bad, []byte("this is not a zip file"), 0o644)
		if _, err := ExtractBunBinary(bad, t.TempDir()); err == nil {
			t.Fatal("expected OpenReader error")
		}
	})

	t.Run("unsupported compression method", func(t *testing.T) {
		var buf bytes.Buffer
		w := zip.NewWriter(&buf)
		hdr := &zip.FileHeader{Name: "nested/" + binary, Method: 99}
		if _, err := w.CreateRaw(hdr); err != nil {
			t.Fatal(err)
		}
		w.Close()
		zipPath := filepath.Join(t.TempDir(), "method.zip")
		os.WriteFile(zipPath, buf.Bytes(), 0o644)

		if _, err := ExtractBunBinary(zipPath, t.TempDir()); err == nil {
			t.Fatal("expected file.Open error for unknown method")
		}
	})

	t.Run("corrupt payload checksum", func(t *testing.T) {
		data := []byte("real content here")
		var buf bytes.Buffer
		w := zip.NewWriter(&buf)
		hdr := &zip.FileHeader{Name: "pkg/" + binary, Method: zip.Store}
		hdr.CRC32++ // deliberately wrong
		hdr.CompressedSize64 = uint64(len(data))
		hdr.UncompressedSize64 = uint64(len(data))
		if _, err := w.CreateRaw(hdr); err != nil {
			t.Fatal(err)
		}
		w.Flush()
		buf.Write(data)
		w.Close()
		zipPath := filepath.Join(t.TempDir(), "crc.zip")
		os.WriteFile(zipPath, buf.Bytes(), 0o644)

		if _, err := ExtractBunBinary(zipPath, t.TempDir()); err == nil {
			t.Fatal("expected checksum/copy error")
		}
	})

	t.Run("readonly destination", func(t *testing.T) {
		var buf bytes.Buffer
		w := zip.NewWriter(&buf)
		entry, err := w.Create("pkg/" + binary)
		if err != nil {
			t.Fatal(err)
		}
		entry.Write([]byte("content"))
		w.Close()
		zipPath := filepath.Join(t.TempDir(), "ok.zip")
		os.WriteFile(zipPath, buf.Bytes(), 0o644)

		dest := t.TempDir()
		readOnly(t, dest)
		if _, err := ExtractBunBinary(zipPath, dest); err == nil {
			t.Fatal("expected create error in readonly dest")
		}
	})
}
