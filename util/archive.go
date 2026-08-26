package util

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractBunBinary extracts only the bun executable from a release archive
// into destDir and returns its path. Executable permission bits are set on
// unix; mode bits are ignored on Windows.
func ExtractBunBinary(zipPath, destDir string) (string, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed opening archive: %w", err)
	}
	defer reader.Close()

	binaryName := BinaryName()
	destPath := filepath.Join(destDir, binaryName)

	for _, file := range reader.File {
		if skipZipEntry(file, binaryName) {
			continue
		}

		src, err := file.Open()
		if err != nil {
			return "", err
		}
		err = writeFile(destPath, src)
		src.Close()
		if err != nil {
			return "", err
		}
		return destPath, nil
	}
	return "", fmt.Errorf("archive does not contain %s", binaryName)
}

// skipZipEntry reports whether an archive entry is not the bun binary we
// want. Entries are matched by base name anywhere inside the archive so
// nested layouts like "bun-windows-x64/bun.exe" work without hardcoding.
func skipZipEntry(file *zip.File, binaryName string) bool {
	if file.FileInfo().IsDir() {
		return true
	}
	if filepath.Base(filepath.FromSlash(file.Name)) != binaryName {
		return true
	}
	return strings.Contains(file.Name, "..") // zip-slip guard
}

// writeFile writes r to path with executable permissions.
func writeFile(path string, r io.Reader) error {
	dst, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(dst, r)
	closeErr := dst.Close()
	if copyErr == nil {
		copyErr = closeErr
	}
	return copyErr
}
