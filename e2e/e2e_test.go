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

func bunVersionOutput(t *testing.T) string {
	t.Helper()
	name := "bun"
	if runtime.GOOS == "windows" {
		name = "bun.exe"
	}
	cmd := exec.Command(filepath.Join(homeDir, ".bun", "bin", name), "--version")
	cmd.Env = append(os.Environ(), "HOME="+homeDir, "USERPROFILE="+homeDir)
	out, err := runWithTimeout(cmd, time.Minute)
	requireSuccess(t, out, err, "activated bun --version")
	return strings.TrimSpace(out)
}

// TestE2EFlow runs the full lifecycle in order; subtests share state on
// purpose since each step builds on the previous one.
func TestE2EFlow(t *testing.T) {
	var latestVersion string

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

	t.Run("InstallPinnedVersionAndActivate", func(t *testing.T) {
		out, err := run(t, "install", pinnedVersion)
		requireSuccess(t, out, err, "bvm install "+pinnedVersion)

		if !utilIsInstalled(t, pinnedVersion) {
			t.Fatal("version directory/binary missing after install")
		}
		if got := bunVersionOutput(t); got != strings.TrimPrefix(pinnedVersion, "v") {
			t.Fatalf("activated bun --version = %q, want %q", got, strings.TrimPrefix(pinnedVersion, "v"))
		}
		// First ever install must record itself as the default alias.
		assertDefaultAlias(t, pinnedVersion)
	})

	t.Run("InstallLatestAndActivate", func(t *testing.T) {
		out, err := run(t, "install", "latest")
		requireSuccess(t, out, err, "bvm install latest")

		want := strings.TrimPrefix(latestVersion, "v")
		if got := bunVersionOutput(t); got != want {
			t.Fatalf("after 'install latest', bun --version = %q, want %q", got, want)
		}
		// A later install must not steal the default.
		assertDefaultAlias(t, pinnedVersion)
	})

	t.Run("UseSwitchesActiveVersion", func(t *testing.T) {
		out, err := run(t, "use", pinnedVersion)
		requireSuccess(t, out, err, "bvm use "+pinnedVersion)
		if got := bunVersionOutput(t); got != strings.TrimPrefix(pinnedVersion, "v") {
			t.Fatalf("after 'use', bun --version = %q, want %q", got, strings.TrimPrefix(pinnedVersion, "v"))
		}
	})

	t.Run("UseLatestResolvesHighestInstalled", func(t *testing.T) {
		out, err := run(t, "use", "latest")
		requireSuccess(t, out, err, "bvm use latest")
		if got := bunVersionOutput(t); got != strings.TrimPrefix(latestVersion, "v") {
			t.Fatalf("'use latest' gave %q, want %q", got, strings.TrimPrefix(latestVersion, "v"))
		}
		// Switch back for subsequent steps.
		if out, err := run(t, "use", pinnedVersion); err != nil {
			t.Fatalf("use %s failed: %v\n%s", pinnedVersion, err, out)
		}
	})

	t.Run("ListMarksActiveVersion", func(t *testing.T) {
		out, err := run(t, "ls")
		requireSuccess(t, out, err, "bvm ls")
		for _, line := range nonEmptyLines(out) {
			if strings.HasPrefix(line, "*") && strings.Contains(line, pinnedVersion) {
				return
			}
		}
		t.Fatalf("active marker missing for %s in:\n%s", pinnedVersion, out)
	})

	t.Run("UninstallGuardsActiveVersion", func(t *testing.T) {
		out, err := run(t, "uninstall", latestVersion)
		requireSuccess(t, out, err, "uninstall inactive "+latestVersion)
		if utilIsInstalled(t, latestVersion) {
			t.Fatal("inactive version still installed after uninstall")
		}

		out, err = run(t, "uninstall", pinnedVersion)
		requireFailure(t, out, err, "uninstall active "+pinnedVersion)
		if !utilIsInstalled(t, pinnedVersion) {
			t.Fatal("active version was removed despite guard")
		}
	})

	t.Run("UseReadsDotBvmrc", func(t *testing.T) {
		project := filepath.Join(homeDir, "project")
		os.MkdirAll(project, 0o755)

		// Both prefixed and unprefixed values must resolve.
		for _, rcValue := range []string{pinnedVersion, strings.TrimPrefix(pinnedVersion, "v")} {
			os.WriteFile(filepath.Join(project, ".bvmrc"), []byte(rcValue+"\n"), 0o644)

			out, err := runIn(t, project, "use")
			requireSuccess(t, out, err, "bvm use with .bvmrc "+rcValue)
			if got := bunVersionOutput(t); got != strings.TrimPrefix(pinnedVersion, "v") {
				t.Fatalf("after 'bvm use' via .bvmrc %q, bun --version = %q", rcValue, got)
			}
		}

		out, err := runIn(t, project, "install")
		requireSuccess(t, out, err, "bvm install with .bvmrc (already installed)")
	})

	t.Run("DefaultAliasFallback", func(t *testing.T) {
		out, err := run(t, "alias", "default", pinnedVersion)
		requireSuccess(t, out, err, "bvm alias default")

		// Plain 'bvm use' outside any project must fall back to the default.
		fresh := filepath.Join(homeDir, "fresh-project")
		os.MkdirAll(fresh, 0o755)
		out, err = runIn(t, fresh, "use")
		requireSuccess(t, out, err, "bvm use with default alias")
		if got := bunVersionOutput(t); got != strings.TrimPrefix(pinnedVersion, "v") {
			t.Fatalf("after 'bvm use' via default alias, bun --version = %q", got)
		}

		out, err = run(t, "ls")
		requireSuccess(t, out, err, "bvm ls")
		if !strings.Contains(out, "default") {
			t.Fatalf("'ls' output missing default marker:\n%s", out)
		}
	})

	t.Run("InvalidVersionsFailCleanly", func(t *testing.T) {
		out, err := run(t, "install", "not-a-version")
		requireFailure(t, out, err, "install not-a-version")
	})
}

func utilIsInstalled(t *testing.T, version string) bool {
	t.Helper()
	binary := "bun"
	if runtime.GOOS == "windows" {
		binary = "bun.exe"
	}
	_, err := os.Stat(filepath.Join(bvmRoot, "versions", version, binary))
	return err == nil
}

// assertDefaultAlias checks $BVM_DIR/default points at the expected version.
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

func nonEmptyLines(s string) []string {
	var out []string
	for _, line := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}
