package command

import (
	"fmt"
	"strings"

	"github.com/chathula/bvm/util"
	"github.com/fatih/color"
)

// ensureShim is a var so tests can stub shim installation.
var ensureShim = util.EnsureShim

// Use reports which version applies to the current directory, or changes
// it: 'default' removes an existing .bvmrc pin, --reset explains how to
// clear a session override, and --save writes one. The pin file is always
// opt-in — it is never created implicitly.
func Use(reset, save bool, arg string) error {
	arg = strings.TrimSpace(arg)

	if arg == "" {
		return showResolution()
	}

	if reset {
		fmt.Println("Session overrides live in the BVM_VERSION environment variable.")
		fmt.Println("Clear it in your shell:  unset BVM_VERSION   (fish: set -e BVM_VERSION)")
		return nil
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

	if save {
		if err := util.WriteRC(version); err != nil {
			return fail("%v", err)
		}
		if err := ensureShim(); err != nil {
			fmt.Println(color.YellowString("Warning: could not install the bun shim: %v", err))
		}
		fmt.Println(color.GreenString("bun %s pinned to this directory (%s)", version, util.RCFileName()))
		return nil
	}

	def, _ := util.DefaultVersion()
	if version == def {
		fmt.Println(color.GreenString("bun %s is already the default — it applies here unless a %s or $BVM_VERSION overrides it.", version, util.RCFileName()))
		return nil
	}
	fmt.Println(color.YellowString("bun %s is installed but applies nowhere by default.", version))
	fmt.Println("To pin it to this project:")
	fmt.Printf("  bvm use --save %s      # writes %s in this directory\n", version, util.RCFileName())
	fmt.Println("To use it temporarily in this shell session (nvm-style):")
	fmt.Printf("  bvm use %s             # sets $BVM_VERSION until you close the shell\n", version)
	fmt.Println("To make it the global default:")
	fmt.Printf("  bvm alias default %s\n", version)
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
