package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/chathula/bvm/command"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

func test(str string) string {
	return color.GreenString(str)
}

func main() {
	cliApp := cli.NewApp()

	cliApp.Name = "bvm"
	cliApp.Description = "🚀 Bun Version Manager - Manage multiple bun versions easily"
	cliApp.Usage = cliApp.Name + " <COMMAND>"
	cliApp.EnableBashCompletion = true

	cliApp.Commands = []*cli.Command{
		{
			Name:    "list-remote",
			Aliases: []string{"ls-remote"},
			Usage:   "List all remote bun versions",
			Action: func(c *cli.Context) error {
				return command.ListRemote()
			},
		},
		{
			Name:      "install",
			Usage:     "Install given bun version",
			ArgsUsage: "<version>",
			Aliases:   []string{"i"},
			Action: func(c *cli.Context) error {
				if c.Args().Len() == 0 {
					return errors.New(fmt.Sprintf("require argument <%s>", color.YellowString("version")))
				}
				return command.Install(c.Args().First())
			},
		},
	}

	cli.AppHelpTemplate = color.YellowString(cliApp.Name) + ` - {{.Description}}

` + color.YellowString("USAGE:") + `
	{{.Usage}}

` + color.YellowString("COMMANDS:") + `
{{range .Commands}}{{if not .HideHelp}}	` + color.GreenString(`{{join .Names ", "}}`) + ` {{"\t"}}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{if .VisibleFlags}}{{end}}
` + color.YellowString("EXAMPLES:") + `
	{{.Name}} install latest
	{{.Name}} install 0.1.1
	{{.Name}} use latest
	{{.Name}} use 0.1.1
	{{.Name}} ls
	{{.Name}} ls-remote
`

	if err := cliApp.Run(os.Args); err != nil {
		fmt.Println(err.Error())
	}
}
