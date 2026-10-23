package command

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/chathula/bvm/util"
)

func TestRunShimArms(t *testing.T) {
	setupCommandEnv(t)

	t.Run("nothing resolves", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if err := runShim("bun"); err == nil {
			t.Fatal("expected resolution failure")
		}
	})

	t.Run("resolved binary missing", func(t *testing.T) {
		isolate := t.TempDir()
		t.Setenv("BVM_DIR", filepath.Join(isolate, ".bvm"))
		util.SetDefaultVersion("v9.9.9") // default present but binary absent
		t.Chdir(t.TempDir())

		err := runShim("bun")
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
		stub(t, &execProcessFn, func(bin string, args []string) error {
			got = bin
			return errors.New("exec-replaced")
		})

		err := runShim("bun")
		if err == nil || err.Error() != "exec-replaced" {
			t.Fatalf("expected exec stub error, got %v", err)
		}
		want := filepath.Join(mustBVMDir(t), "versions", "v1.4.0", util.BinaryName())
		if got != want {
			t.Fatalf("exec target = %q, want %q", got, want)
		}
	})
}

func TestRunBunxShimPrependsX(t *testing.T) {
	setupCommandEnv(t)
	fakeInstall(t, "v1.4.0")
	util.SetDefaultVersion("v1.4.0")
	t.Chdir(t.TempDir())

	oldArgs := os.Args
	os.Args = []string{"bunx", "cowsay", "hi"}
	t.Cleanup(func() { os.Args = oldArgs })

	var gotBin string
	var gotArgs []string
	stub(t, &execProcessFn, func(bin string, args []string) error {
		gotBin = bin
		gotArgs = args
		return nil
	})

	if err := runShim("bunx"); err != nil {
		t.Fatalf("runShim(bunx) error = %v", err)
	}
	wantBin := filepath.Join(mustBVMDir(t), "versions", "v1.4.0", util.BinaryName())
	if gotBin != wantBin {
		t.Fatalf("exec target = %q, want %q", gotBin, wantBin)
	}
	if want := []string{"x", "cowsay", "hi"}; !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("exec args = %v, want %v", gotArgs, want)
	}
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
	stub(t, &execProcessFn, func(string, []string) error { return nil }) // success: no exit

	RunShim("bun")
	if exited {
		t.Fatalf("unexpected exit on success: %s", gotMsg)
	}

	stub(t, &execProcessFn, func(string, []string) error { return errors.New("boom") })
	RunShim("bun")
	if !exited || gotMsg != "boom" {
		t.Fatalf("exitWithError = (%q, %v)", gotMsg, exited)
	}

	// VersionDir failure arm.
	stub(t, &exitWithError, func(msg string) { gotMsg = msg; exited = true })
	stub(t, &versionDir, func(string) (string, error) { return "", errors.New("injected") })
	RunShim("bun")
	if !exited || !strings.Contains(gotMsg, "injected") {
		t.Fatalf("versionDir failure not reported: %q", gotMsg)
	}
}
