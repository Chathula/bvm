// Command bvm manages multiple Bun versions.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chathula/bvm/command"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

// version is injected at release time via -ldflags "-X main.version=x.y.z".
var version = "dev"

// isShimInvocation reports whether this binary was invoked as the bun shim
// (i.e. linked/copied to ~/.bun/bin/bun).
func isShimInvocation() bool {
	base := strings.ToLower(filepath.Base(os.Args[0]))
	return base == "bun" || base == "bun.exe"
}

func main() {
	// Invoked as "bun" (the shim): resolve the version for this directory
	// and hand off to it instead of running the CLI.
	if isShimInvocation() {
		command.RunShim()
		return
	}

	cliApp := cli.NewApp()
	cliApp.Name = "bvm"
	cliApp.Version = version
	cliApp.Description = "🚀 Bun Version Manager - Manage multiple bun versions easily"
	cliApp.Usage = cliApp.Name + " <COMMAND>"
	cliApp.EnableBashCompletion = true

	cliApp.Commands = []*cli.Command{
		{
			Name:      "install",
			Usage:     "Install given bun version (defaults to .bvmrc)",
			ArgsUsage: "[version]",
			Aliases:   []string{"i"},
			Action: func(c *cli.Context) error {
				return command.Install(c.Args().First())
			},
		},
		{
			Name:      "use",
			Usage:     "Pin a bun version to the current directory (.bvmrc); 'default' clears the pin; no arg shows what applies here",
			ArgsUsage: "[version]",
			Action: func(c *cli.Context) error {
				return command.Use(c.Args().First())
			},
		},
		{
			Name:    "list",
			Usage:   "List installed bun versions",
			Aliases: []string{"ls"},
			Action: func(c *cli.Context) error {
				return command.List()
			},
		},
		{
			Name:    "list-remote",
			Usage:   "List all remote bun versions",
			Aliases: []string{"ls-remote"},
			Action: func(c *cli.Context) error {
				return command.ListRemote()
			},
		},
		{
			Name:      "uninstall",
			Usage:     "Remove an installed bun version",
			ArgsUsage: "<version>",
			Aliases:   []string{"rm"},
			Action: func(c *cli.Context) error {
				if !c.Args().Present() {
					return fmt.Errorf("require argument <%s>", color.YellowString("version"))
				}
				return command.Uninstall(c.Args().First())
			},
		},
		{
			Name:      "alias",
			Usage:     "Set the default bun version used when no version is given",
			ArgsUsage: "default <version>",
			Action: func(c *cli.Context) error {
				return command.Alias(c.Args().First(), c.Args().Get(1))
			},
		},
		{
			Name:  "doctor",
			Usage: "Diagnose your bvm and bun installation",
			Action: func(c *cli.Context) error {
				return command.Doctor()
			},
		},
	}

	cli.AppHelpTemplate = color.YellowString(cliApp.Name) + ` - {{.Description}}

` + color.YellowString("USAGE:") + `
	{{.Usage}}

` + color.YellowString("VERSION:") + `
	{{.Version}}

` + color.YellowString("COMMANDS:") + `
{{range .Commands}}	` + color.GreenString(`{{join .Names ", "}}`) + ` {{"\t"}}{{.Usage}}{{ "\n" }}{{end}}
` + color.YellowString("EXAMPLES:") + `
	{{.Name}} install latest
	{{.Name}} install 1.1.0
	{{.Name}} install        # installs the version from .bvmrc
	{{.Name}} use 1.1.0
	{{.Name}} use            # uses the version from .bvmrc, or the default
	{{.Name}} alias default 1.1.0
	{{.Name}} ls
	{{.Name}} ls-remote
	{{.Name}} uninstall 1.1.0
	{{.Name}} doctor
`

	if err := cliApp.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
