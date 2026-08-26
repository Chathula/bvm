package command

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chathula/bvm/util"
)

func TestRunShimArms(t *testing.T) {
	setupCommandEnv(t)

	t.Run("nothing resolves", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if err := runShim(); err == nil {
			t.Fatal("expected resolution failure")
		}
	})

	t.Run("resolved binary missing", func(t *testing.T) {
		isolate := t.TempDir()
		t.Setenv("BVM_DIR", filepath.Join(isolate, ".bvm"))
		util.SetDefaultVersion("v9.9.9") // default present but binary absent
		t.Chdir(t.TempDir())

		err := runShim()
		if err == nil || !strings.Contains(err.Error(), "not installed") {
			t.Fatalf("expected missing-binary error, got %v", err)
		}
	})

	t.Run("hands off to resolved binary", func(t *testing.T) {
		setupCommandEnv(t)
		fakeInstall(t, "v1.4.0")
		util.SetDefaultVersion("v1.4.0")
		t.Chdir(t.TempDir())

		var got string
		stub(t, &execBunFn, func(bin string) error {
			got = bin
			return errors.New("exec-replaced")
		})

		err := runShim()
		if err == nil || err.Error() != "exec-replaced" {
			t.Fatalf("expected exec stub error, got %v", err)
		}
		want := filepath.Join(mustBVMDir(t), "versions", "v1.4.0", util.BinaryName())
		if got != want {
			t.Fatalf("exec target = %q, want %q", got, want)
		}
	})
}

func TestRunShimExitPath(t *testing.T) {
	setupCommandEnv(t)
	fakeInstall(t, "v1.4.0")
	util.SetDefaultVersion("v1.4.0")
	t.Chdir(t.TempDir())

	var gotMsg string
	var exited bool
	stub(t, &exitWithError, func(msg string) {
		gotMsg = msg
		exited = true
	})
	stub(t, &execBunFn, func(string) error { return nil }) // success: no exit

	RunShim()
	if exited {
		t.Fatalf("unexpected exit on success: %s", gotMsg)
	}

	stub(t, &execBunFn, func(string) error { return errors.New("boom") })
	RunShim()
	if !exited || gotMsg != "boom" {
		t.Fatalf("exitWithError = (%q, %v)", gotMsg, exited)
	}

	// VersionDir failure arm.
	stub(t, &exitWithError, func(msg string) { gotMsg = msg; exited = true })
	stub(t, &versionDir, func(string) (string, error) { return "", errors.New("injected") })
	RunShim()
	if !exited || !strings.Contains(gotMsg, "injected") {
		t.Fatalf("versionDir failure not reported: %q", gotMsg)
	}
}
