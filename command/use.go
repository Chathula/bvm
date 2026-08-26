package command

import (
	"fmt"

	"github.com/chathula/bvm/util"
	"github.com/fatih/color"
)

// Use activates a locally installed Bun version. The version may be given
// explicitly, be "latest" (highest installed), or come from a .bvmrc file.
func Use(arg string) error {
	version, err := resolveLocal(resolveVersionArg(arg))
	if err != nil {
		return fail("%v", err)
	}

	if err := util.Activate(version); err != nil {
		return fail("%v", err)
	}

	changed, err := ensurePATH()
	if err != nil {
		fmt.Println(color.YellowString("Warning: could not update shell profile: %v", err))
	} else if changed {
		fmt.Println(color.YellowString("Added ~/.bun/bin to your PATH — restart your shell or source your profile."))
	}

	fmt.Println(color.GreenString("Now using bun %s", version))
	return nil
}

func resolveLocal(arg string) (string, error) {
	installed, err := util.LocalVersions()
	if err != nil {
		return "", err
	}
	if len(installed) == 0 {
		return "", fmt.Errorf("no versions installed yet — run 'bvm install latest'")
	}
	highest := installed[len(installed)-1]

	switch arg {
	case "":
		// No explicit version and no .bvmrc: fall back to the default
		// alias, then to the highest installed version.
		def, _ := util.DefaultVersion()
		if def != "" && util.IsInstalled(def) {
			return def, nil
		}
		return highest, nil
	case "latest":
		return highest, nil
	case "default":
		def, _ := util.DefaultVersion()
		if def == "" {
			return "", fmt.Errorf("no default version set — run 'bvm alias default <version>'")
		}
		if !util.IsInstalled(def) {
			return "", fmt.Errorf("default version %s is not installed — run 'bvm install'", def)
		}
		return def, nil
	}

	version, err := util.NormalizeVersion(arg)
	if err != nil {
		return "", err
	}
	for _, v := range installed {
		if v == version {
			return version, nil
		}
	}
	return "", fmt.Errorf("version %s is not installed — run 'bvm install %s'", version, version)
}
