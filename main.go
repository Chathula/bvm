package main

import (
	"log"
	"os"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

func main() {
	cliApp := cli.NewApp()

	cliApp.Name = "bvm"
	cliApp.Description = "🚀 Bun Version Manager - Manage multiple bun versions easily"
	cliApp.Usage = cliApp.Name + " <COMMAND>"
	cliApp.HideHelp = true

	cliApp.Commands = []*cli.Command{
		{
			Name:    color.GreenString("ls-remote"),
			Aliases: []string{color.GreenString("list-remote")},
			Usage:   "List released versions",
			Action: func(c *cli.Context) error {
				return nil
				// if c.Args().Len() == 0 {
				// 	return fmt.Errorf("require argument <%s>", color.RedString("file|gist-url"))
				// }
				// return command.Install(c.Args().First())
			},
		},
	}

	// 		{{range .Commands}}{{if not .HideHelp}}   {{join .Names ", "}} {{ .ArgsUsage }}{{ "\t"}}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{end}}{{if .VisibleFlags}}

	cli.AppHelpTemplate = color.YellowString(cliApp.Name) + ` - {{.Description}}

` + color.YellowString("USAGE:") + `
		{{.Usage}}

` + color.YellowString("COMMANDS:") + `
{{range .Commands}}{{if not .HideHelp}}   {{join .Names ", "}} {{"\t"}}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{if .VisibleFlags}}{{end}}
EXAMPLES:
    {{.Name}} install latest
    {{.Name}} install 0.1.1
		{{.Name}} use latest
    {{.Name}} use 0.1.1
		{{.Name}} ls
		{{.Name}} ls-remote
`

	if err := cliApp.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
