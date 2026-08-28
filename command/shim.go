package command

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chathula/bvm/util"
)

// execProcessFn is a var so tests can stub process replacement.
var execProcessFn = execProcess

// versionDir is a var so tests can inject lookup failures portably.
var versionDir = util.VersionDir

// exitWithError is a var so tests can capture the fatal exit path.
var exitWithError = func(msg string) {
	fmt.Fprintln(os.Stderr, "bvm:", msg)
	os.Exit(1)
}

// RunShim implements the `bun` shim: invoked as bun, bvm resolves which
// installed version applies to the current directory (nearest .bvmrc walking
// up, else the default alias) and hands the arguments over to that binary.
func RunShim() {
	if err := runShim(); err != nil {
		exitWithError(err.Error())
	}
}

func runShim() error {
	version, _, err := util.ResolveVersion()
	if err != nil {
		return err
	}

	dir, err := versionDir(version)
	if err != nil {
		return err
	}
	bin := filepath.Join(dir, util.BinaryName())
	if _, err := os.Stat(bin); err != nil {
		return fmt.Errorf("bun %s is not installed — run 'bvm install %s'", version, version)
	}
	return execProcessFn(bin, os.Args[1:])
}
