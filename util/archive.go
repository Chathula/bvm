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
		if file.FileInfo().IsDir() {
			continue
		}
		// Match by base name anywhere inside the archive so nested layouts
		// like "bun-windows-x64/bun.exe" work without hardcoding them.
		if filepath.Base(filepath.FromSlash(file.Name)) != binaryName {
			continue
		}
		if strings.Contains(file.Name, "..") {
			continue // zip-slip guard
		}

		src, err := file.Open()
		if err != nil {
			return "", err
		}
		dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
		if err != nil {
			src.Close()
			return "", err
		}
		_, err = io.Copy(dst, src)
		closeErr := dst.Close()
		src.Close()
		if err != nil {
			return "", err
		}
		if closeErr != nil {
			return "", closeErr
		}
		return destPath, nil
	}
	return "", fmt.Errorf("archive does not contain %s", binaryName)
}
