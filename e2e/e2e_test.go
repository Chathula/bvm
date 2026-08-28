//go:build e2e

// Package e2e exercises the built bvm binary against the real Bun release
// infrastructure. Run with: go test -tags=e2e ./e2e
package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// pinnedVersion has release archives for every platform in the CI matrix.
const pinnedVersion = "v1.1.0"

var (
	bvmBin  string // path to the freshly built bvm binary
	bvmRoot string // isolated $BVM_DIR
	homeDir string // isolated $HOME
)

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "bvm-e2e-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)

	bvmRoot = filepath.Join(tmp, ".bvm")
	homeDir = filepath.Join(tmp, "home")
	os.MkdirAll(homeDir, 0o755)

	name := "bvm"
	if runtime.GOOS == "windows" {
		name = "bvm.exe"
	}
	bvmBin = filepath.Join(tmp, name)
	build := exec.Command("go", "build", "-o", bvmBin, "github.com/chathula/bvm")
	if out, err := build.CombinedOutput(); err != nil {
		panic("failed to build bvm: " + string(out))
	}

	os.Exit(m.Run())
}

// run executes bvm in an isolated environment and returns (stdout+stderr, error).
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	return runIn(t, "", args...)
}

// runIn is run with an explicit working directory for the bvm process.
func runIn(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(bvmBin, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(),
		"BVM_DIR="+bvmRoot,
		"HOME="+homeDir,
		"USERPROFILE="+homeDir,
	)
	return runWithTimeout(cmd, 10*time.Minute)
}

// runBun invokes the installed bun shim the way a user's shell would.
func runBun(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	name := "bun"
	if runtime.GOOS == "windows" {
		name = "bun.exe"
	}
	cmd := exec.Command(filepath.Join(homeDir, ".bun", "bin", name), args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "BVM_DIR="+bvmRoot, "HOME="+homeDir, "USERPROFILE="+homeDir)
	return runWithTimeout(cmd, time.Minute)
}

func runWithTimeout(cmd *exec.Cmd, timeout time.Duration) (string, error) {
	var buf strings.Builder
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Start(); err != nil {
		return "", err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-time.After(timeout):
		cmd.Process.Kill()
		return buf.String(), fmt.Errorf("command timed out: %s", strings.Join(cmd.Args, " "))
	case err := <-done:
		return buf.String(), err
	}
}

func requireSuccess(t *testing.T, out string, err error, context string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s failed: %v\noutput:\n%s", context, err, out)
	}
}

func requireFailure(t *testing.T, out string, err error, context string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s unexpectedly succeeded\noutput:\n%s", context, out)
	}
}

func bunVersionAt(t *testing.T, dir string) string {
	t.Helper()
	out, err := runBun(t, dir, "--version")
	requireSuccess(t, out, err, "bun --version in "+dir)
	return strings.TrimSpace(out)
}

func assertDefaultAlias(t *testing.T, want string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(bvmRoot, "default"))
	if err != nil {
		t.Fatalf("default alias file missing: %v", err)
	}
	if got := strings.TrimSpace(string(data)); got != want {
		t.Fatalf("default alias = %q, want %q", got, want)
	}
}

// TestE2EFlow runs the full lifecycle in order; subtests share state on
// purpose since each step builds on the previous one.
func TestE2EFlow(t *testing.T) {
	var latestVersion string
	freshDir := filepath.Join(homeDir, "elsewhere") // no .bvmrc anywhere above
	os.MkdirAll(freshDir, 0o755)

	t.Run("ListRemoteShowsVersions", func(t *testing.T) {
		out, err := run(t, "list-remote")
		requireSuccess(t, out, err, "bvm list-remote")
		lines := nonEmptyLines(out)
		if len(lines) < 2 {
			t.Fatalf("expected many versions, got:\n%s", out)
		}
		last := lines[len(lines)-1]
		if !strings.Contains(last, "(latest)") {
			t.Fatalf("last line missing '(latest)' marker: %q", last)
		}
		latestVersion = strings.TrimSpace(strings.TrimSuffix(last, "(latest)"))
	})

	t.Run("FirstInstallBecomesDefault", func(t *testing.T) {
		out, err := run(t, "install", pinnedVersion)
		requireSuccess(t, out, err, "bvm install "+pinnedVersion)

		assertDefaultAlias(t, pinnedVersion)
		// New shells / unpinned directories get the default.
		if got := bunVersionAt(t, freshDir); got != strings.TrimPrefix(pinnedVersion, "v") {
			t.Fatalf("fresh dir bun --version = %q, want default %q", got, strings.TrimPrefix(pinnedVersion, "v"))
		}
	})

	t.Run("SecondInstallDoesNotStealDefault", func(t *testing.T) {
		out, err := run(t, "install", "latest")
		requireSuccess(t, out, err, "bvm install latest")

		assertDefaultAlias(t, pinnedVersion)
		if got := bunVersionAt(t, freshDir); got != strings.TrimPrefix(pinnedVersion, "v") {
			t.Fatalf("default changed by second install: %q", got)
		}
	})

	t.Run("AliasDefaultSwitchesUnpinnedDirs", func(t *testing.T) {
		out, err := run(t, "alias", "default", latestVersion)
		requireSuccess(t, out, err, "bvm alias default")

		if got := bunVersionAt(t, freshDir); got != strings.TrimPrefix(latestVersion, "v") {
			t.Fatalf("fresh dir bun --version = %q, want %q", got, strings.TrimPrefix(latestVersion, "v"))
		}
	})

	t.Run("UsePinsProjectDirectory", func(t *testing.T) {
		project := filepath.Join(homeDir, "project")
		os.MkdirAll(project, 0o755)

		out, err := runIn(t, project, "use", pinnedVersion)
		requireSuccess(t, out, err, "bvm use (no --save) must not create .bvmrc")
		if _, err := os.Stat(filepath.Join(project, ".bvmrc")); !os.IsNotExist(err) {
			t.Fatal(".bvmrc created without --save")
		}

		out, err = runIn(t, project, "use", "--save", pinnedVersion)
		requireSuccess(t, out, err, "bvm use --save "+pinnedVersion)

		// Project resolves to the pin; everywhere else keeps the default.
		if got := bunVersionAt(t, project); got != strings.TrimPrefix(pinnedVersion, "v") {
			t.Fatalf("project bun --version = %q, want %q", got, strings.TrimPrefix(pinnedVersion, "v"))
		}
		if got := bunVersionAt(t, freshDir); got != strings.TrimPrefix(latestVersion, "v") {
			t.Fatalf("fresh dir bun --version = %q, want default %q", got, strings.TrimPrefix(latestVersion, "v"))
		}

		// 'use default' clears the pin.
		out, err = runIn(t, project, "use", "default")
		requireSuccess(t, out, err, "bvm use default")
		if got := bunVersionAt(t, project); got != strings.TrimPrefix(latestVersion, "v") {
			t.Fatalf("after 'use default', project bun --version = %q, want %q", got, strings.TrimPrefix(latestVersion, "v"))
		}

		// Re-pin for the ls marker check.
		if out, err := runIn(t, project, "use", "--save", pinnedVersion); err != nil {
			t.Fatalf("re-pin failed: %v\n%s", err, out)
		}
	})

	t.Run("UseReadsDotBvmrcPrefixVariants", func(t *testing.T) {
		project := filepath.Join(homeDir, "project")
		for _, rcValue := range []string{pinnedVersion, strings.TrimPrefix(pinnedVersion, "v")} {
			os.WriteFile(filepath.Join(project, ".bvmrc"), []byte(rcValue+"\n"), 0o644)
			if got := bunVersionAt(t, project); got != strings.TrimPrefix(pinnedVersion, "v") {
				t.Fatalf(".bvmrc %q resolved to %q", rcValue, got)
			}
		}
	})

	t.Run("ListShowsMarkers", func(t *testing.T) {
		project := filepath.Join(homeDir, "project")
		out, err := runIn(t, project, "ls")
		requireSuccess(t, out, err, "bvm ls")
		if !strings.Contains(out, "default") || !strings.Contains(out, "this project") {
			t.Fatalf("markers missing in:\n%s", out)
		}
	})

	t.Run("UninstallDefaultReassigns", func(t *testing.T) {
		out, err := run(t, "uninstall", latestVersion)
		requireSuccess(t, out, err, "uninstall default "+latestVersion)

		assertDefaultAlias(t, pinnedVersion)
		if got := bunVersionAt(t, freshDir); got != strings.TrimPrefix(pinnedVersion, "v") {
			t.Fatalf("after default uninstall, bun --version = %q, want %q", got, strings.TrimPrefix(pinnedVersion, "v"))
		}
	})

	t.Run("UninstallLastVersionCleansUp", func(t *testing.T) {
		out, err := run(t, "uninstall", pinnedVersion)
		requireSuccess(t, out, err, "uninstall last "+pinnedVersion)

		shim := "bun"
		if runtime.GOOS == "windows" {
			shim = "bun.exe"
		}
		if _, err := os.Stat(filepath.Join(homeDir, ".bun", "bin", shim)); !os.IsNotExist(err) {
			t.Fatal("shim still present after removing the last version")
		}
		_, err = runBun(t, freshDir, "--version")
		requireFailure(t, "", err, "bun --version with nothing installed")
	})

	t.Run("DoctorRunsAfterReinstall", func(t *testing.T) {
		out, err := run(t, "install", pinnedVersion)
		requireSuccess(t, out, err, "reinstall "+pinnedVersion)

		// Put the shim dir on PATH so the PATH check passes too.
		cmd := exec.Command(bvmBin, "doctor")
		cmd.Dir = freshDir
		cmd.Env = append(os.Environ(),
			"BVM_DIR="+bvmRoot,
			"HOME="+homeDir,
			"USERPROFILE="+homeDir,
			"PATH="+filepath.Join(homeDir, ".bun", "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
		)
		out, err = runWithTimeout(cmd, 5*time.Minute)
		requireSuccess(t, out, err, "bvm doctor")
		if !strings.Contains(out, "releases API reachable") {
			t.Fatalf("doctor output missing checks:\n%s", out)
		}
	})

	t.Run("InvalidVersionsFailCleanly", func(t *testing.T) {
		out, err := run(t, "install", "not-a-version")
		requireFailure(t, out, err, "install not-a-version")
	})
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, line := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}
