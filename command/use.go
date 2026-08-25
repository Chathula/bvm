package command

import (
	"fmt"

	"github.com/chathula/bvm/util"
	"github.com/fatih/color"
)

// Use activates a locally installed Bun version ("latest" resolves to the
// highest installed version).
func Use(arg string) error {
	version, err := resolveLocal(arg)
	if err != nil {
		return fail("%v", err)
	}

	if err := util.Activate(version); err != nil {
		return fail("%v", err)
	}

	changed, err := util.EnsurePATH()
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

	if arg == "" || arg == "latest" {
		return installed[len(installed)-1], nil
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
