package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chathula/bvm/util"
)

func TestResolveVersionArgExplicitWins(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	os.WriteFile(filepath.Join(tmp, rcFileName), []byte("1.0.0\n"), 0o644)

	if got := resolveVersionArg("2.0.0"); got != "2.0.0" {
		t.Fatalf("resolveVersionArg(2.0.0) = %q, want explicit arg to win", got)
	}
}

func TestResolveVersionArgReadsRCFile(t *testing.T) {
	cases := map[string]string{
		"1.2.3\n":                 "1.2.3",
		"\n# comment\n v1.4.0 \n": "v1.4.0",
		"latest":                  "latest",
	}
	for content, want := range cases {
		tmp := t.TempDir()
		t.Chdir(tmp)
		os.WriteFile(filepath.Join(tmp, rcFileName), []byte(content), 0o644)

		if got := resolveVersionArg(""); got != want {
			t.Fatalf("rc %q resolved to %q, want %q", content, got, want)
		}
	}
}

func TestResolveVersionArgEmptyCases(t *testing.T) {
	t.Run("no file yields empty", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if got := resolveVersionArg(""); got != "" {
			t.Fatalf("resolveVersionArg() = %q, want empty", got)
		}
	})
	t.Run("comment-only file yields empty", func(t *testing.T) {
		tmp := t.TempDir()
		t.Chdir(tmp)
		os.WriteFile(filepath.Join(tmp, rcFileName), []byte("\n# only comments\n"), 0o644)
		if got := resolveVersionArg(""); got != "" {
			t.Fatalf("resolveVersionArg() = %q, want empty", got)
		}
	})
}

func TestUseReadsRCWithAndWithoutVPrefix(t *testing.T) {
	setupCommandEnv(t)

	// Install a fake version to activate.
	dir, err := util.VersionDir("v1.4.0")
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, util.BinaryName()), []byte("x"), 0o755)

	for _, rcValue := range []string{"v1.4.0", "1.4.0"} {
		t.Run(rcValue, func(t *testing.T) {
			tmp := t.TempDir()
			t.Chdir(tmp)
			os.WriteFile(filepath.Join(tmp, rcFileName), []byte(rcValue+"\n"), 0o644)

			if err := Use(""); err != nil {
				t.Fatalf("Use() with .bvmrc %q error = %v", rcValue, err)
			}
			active, err := util.ActiveVersion()
			if err != nil || active != "v1.4.0" {
				t.Fatalf("active = (%q, %v), want v1.4.0", active, err)
			}
		})
	}
}
