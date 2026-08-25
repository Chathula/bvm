package command

import (
	"os"

	"github.com/chathula/bvm/util"
)

// Uninstall removes a locally installed Bun version. The active version is
// protected to avoid leaving the environment broken.
func Uninstall(arg string) error {
	version, err := util.NormalizeVersion(arg)
	if err != nil {
		return fail("%v", err)
	}
	if !util.IsInstalled(version) {
		return fail("version %s is not installed", version)
	}

	active, err := util.ActiveVersion()
	if err != nil {
		return fail("%v", err)
	}
	if active == version {
		return fail("cannot uninstall the active version — switch first with 'bvm use <other-version>'")
	}

	dir, err := util.VersionDir(version)
	if err != nil {
		return fail("%v", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		return fail("%v", err)
	}
	return nil
}
