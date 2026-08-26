package command

import (
	"fmt"
	"strings"

	"github.com/chathula/bvm/util"
	"github.com/fatih/color"
)

// ensureShim is a var so tests can stub shim installation.
var ensureShim = util.EnsureShim

// Use pins a bun version to the current directory by writing .bvmrc —
// nvm-style "active in this project". 'default' removes the local pin so
// the directory follows the global default again; no argument reports the
// version that currently applies here.
func Use(arg string) error {
	arg = strings.TrimSpace(arg)

	if arg == "" {
		return showResolution()
	}

	if arg == "default" {
		rcPath, _ := util.FindRCHere()
		if err := util.RemoveRC(); err != nil {
			return fail("%v", err)
		}
		if rcPath == "" {
			fmt.Println("No local pin here — this directory already follows the default.")
			return nil
		}
		def, _ := util.DefaultVersion()
		fmt.Printf("Removed %s — this directory now follows the default (%s).\n", rcPath, def)
		return nil
	}

	version, err := util.NormalizeVersion(arg)
	if err != nil {
		return fail("%v", err)
	}
	if !util.IsInstalled(version) {
		return fail("version %s is not installed — run 'bvm install %s'", version, version)
	}

	if err := util.WriteRC(version); err != nil {
		return fail("%v", err)
	}
	if err := ensureShim(); err != nil {
		fmt.Println(color.YellowString("Warning: could not install the bun shim: %v", err))
	}

	fmt.Println(color.GreenString("bun %s pinned to this directory (%s)", version, util.RCFileName()))
	return nil
}

func showResolution() error {
	version, source, err := util.ResolveVersion()
	if err != nil {
		return fail("%v", err)
	}
	fmt.Printf("bun %s (%s)\n", color.GreenString(version), source)
	return nil
}
