package util

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// osExecutable is a var so tests can inject lookup failures portably.
var osExecutable = os.Executable

// placeBinary links or copies src to dst depending on useSymlink.
// It is a var so tests can inject failures portably.
var placeBinary = func(src, dst string, useSymlink bool) error {
	if !useSymlink {
		return copyBinary(src, dst)
	}
	return os.Symlink(src, dst)
}

// shimPaths returns each executable bvm exposes, in creation order.
func shimPaths() []func() (string, error) {
	return []func() (string, error){BunBinPath, BunxBinPath}
}

// EnsureShim exposes bvm itself as ~/.bun/bin/bun and ~/.bun/bin/bunx —
// symlinks on unix, copies on Windows. Invoked as "bun" or "bunx", bvm
// resolves the right version per directory (see ResolveVersion) and hands
// off to it, so bunx follows the same version rules as bun.
func EnsureShim() error {
	exe, err := osExecutable()
	if err != nil {
		return err
	}
	if resolved, lerr := filepath.EvalSymlinks(exe); lerr == nil {
		exe = resolved
	}

	for _, path := range shimPaths() {
		dst, err := path()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		// A stale shim pointing at a removed binary must not block us.
		if err := RemoveAll(dst); err != nil {
			return err
		}
		if err := placeBinary(exe, dst, runtime.GOOS != "windows"); err != nil {
			return err
		}
	}
	return nil
}

// RemoveShim deletes the exposed bun and bunx shims (used when nothing is
// installed).
func RemoveShim() error {
	for _, path := range shimPaths() {
		dst, err := path()
		if err != nil {
			return err
		}
		err = RemoveAll(dst)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}

// ClearDefault removes the default alias file.
func ClearDefault() error {
	root, err := BVMDir()
	if err != nil {
		return err
	}
	err = os.Remove(filepath.Join(root, defaultAliasFile))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func copyBinary(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
