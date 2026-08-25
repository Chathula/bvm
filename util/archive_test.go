package util

import (
	"archive/zip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func createTestZip(t *testing.T, entries map[string][]byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	for name, content := range entries {
		hdr := &zip.FileHeader{Name: name, Method: zip.Deflate}
		if filepath.FromSlash(name) != filepath.Base(name) && len(name) > 0 && name[len(name)-1] == '/' {
			continue // skip explicit dir entries; zip handles them implicitly
		}
		fw, err := w.CreateHeader(hdr)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExtractBunBinaryNested(t *testing.T) {
	binary := BinaryNameForOS(runtime.GOOS)
	zipPath := createTestZip(t, map[string][]byte{
		"bun-linux-x64/README.md": []byte("docs"),
		"bun-linux-x64/" + binary: []byte("#!/bin/sh\necho bun\n"),
	})

	dest := t.TempDir()
	got, err := ExtractBunBinary(zipPath, dest)
	if err != nil {
		t.Fatalf("ExtractBunBinary() error = %v", err)
	}
	if filepath.Base(got) != binary {
		t.Fatalf("extracted to %q, want base %q", got, binary)
	}
	data, err := os.ReadFile(got)
	if err != nil || string(data) != "#!/bin/sh\necho bun\n" {
		t.Fatalf("content mismatch: %q (%v)", data, err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(got)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0o111 == 0 {
			t.Fatal("extracted binary is not executable")
		}
	}
}

func TestExtractBunBinaryMissingBinary(t *testing.T) {
	zipPath := createTestZip(t, map[string][]byte{
		"bun-linux-x64/README.md": []byte("docs only"),
	})
	if _, err := ExtractBunBinary(zipPath, t.TempDir()); err == nil {
		t.Fatal("expected error for archive without bun binary")
	}
}
