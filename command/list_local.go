package command

import (
	"fmt"
	"strings"

	"github.com/chathula/bvm/util"
	"github.com/fatih/color"
)

// List prints locally installed Bun versions, marking the global default
// and the version pinned to the current directory (if any).
func List() error {
	versions, err := util.LocalVersions()
	if err != nil {
		return fail("%v", err)
	}
	if len(versions) == 0 {
		fmt.Println("no versions installed yet — run 'bvm install latest'")
		return nil
	}

	def, err := util.DefaultVersion()
	if err != nil {
		return fail("%v", err)
	}
	_, pinned := util.FindRCHere()

	for _, version := range versions {
		marker := "  "
		name := version
		if version == def {
			marker = "*"
			name = color.GreenString(version)
		}

		var tags []string
		if version == def {
			tags = append(tags, "default")
		}
		if pinned != "" && version == pinned {
			tags = append(tags, "this project")
		}
		suffix := ""
		if len(tags) > 0 {
			suffix = color.New(color.FgHiBlack).Sprintf("(%s)", strings.Join(tags, ", "))
		}

		fmt.Printf("%s %s %s\n", marker, name, suffix)
	}
	return nil
}
