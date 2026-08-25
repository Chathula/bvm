package command

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveVersionArgExplicitWins(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	os.WriteFile(filepath.Join(tmp, rcFileName), []byte("1.0.0\n"), 0o644)

	got, err := resolveVersionArg("2.0.0")
	if err != nil || got != "2.0.0" {
		t.Fatalf("resolveVersionArg(2.0.0) = (%q, %v), want explicit arg to win", got, err)
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

		got, err := resolveVersionArg("")
		if err != nil {
			t.Fatalf("resolveVersionArg(%q) error = %v", content, err)
		}
		if got != want {
			t.Fatalf("rc %q resolved to %q, want %q", content, got, want)
		}
	}
}

func TestResolveVersionArgErrors(t *testing.T) {
	t.Run("no file", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if _, err := resolveVersionArg(""); err == nil {
			t.Fatal("expected error when no arg and no .bvmrc")
		}
	})
	t.Run("empty file", func(t *testing.T) {
		tmp := t.TempDir()
		t.Chdir(tmp)
		os.WriteFile(filepath.Join(tmp, rcFileName), []byte("\n# only comments\n"), 0o644)
		if _, err := resolveVersionArg(""); err == nil {
			t.Fatal("expected error for .bvmrc without a version line")
		}
	})
}
