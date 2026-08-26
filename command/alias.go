package command

import (
	"fmt"
	"strings"

	"github.com/chathula/bvm/util"
	"github.com/fatih/color"
)

const defaultAliasName = "default"

// resolveTarget is a var so tests can stub remote resolution offline.
var resolveTarget = util.ResolveTarget

// Alias manages the built-in "default" version alias, mirroring
// 'nvm alias default'. Setting the alias does not switch the active binary;
// it defines what plain 'bvm use' falls back to.
func Alias(name, version string) error {
	if strings.TrimSpace(name) != defaultAliasName {
		return fail("unsupported alias %q — only %q is supported", name, defaultAliasName)
	}
	if strings.TrimSpace(version) == "" {
		return fail("require argument <%s>, e.g. 'bvm alias default 1.4.0'", color.YellowString("version"))
	}

	canonical, err := util.NormalizeVersion(version)
	if err != nil {
		return fail("%v", err)
	}
	if !util.IsInstalled(canonical) {
		// Aliasing an uninstalled version is allowed (like nvm), but the
		// value must exist remotely unless "latest".
		resolved, rerr := resolveTarget(version)
		if rerr != nil {
			return fail("%v", rerr)
		}
		canonical = resolved
	}

	if err := util.SetDefaultVersion(canonical); err != nil {
		return fail("%v", err)
	}

	fmt.Println(color.GreenString("Default bun version set to %s", canonical))
	if !util.IsInstalled(canonical) {
		fmt.Println(color.YellowString("Run 'bvm install' to install it."))
	} else {
		fmt.Println(color.HiBlackString("Activate it anytime with 'bvm use default'."))
	}
	return nil
}
