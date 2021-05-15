package cmd

import (
	"fmt"

	"github.com/fatih/color"
)

/**
* AppHelpTemplate is a formating string for:
* NAME, VERSION, DESCRIPTION, COMMANDS, GLOBAL OPTIONS, COPYRIGHT
**/
var AppHelpTemplate = `
%s:
   {{.Name}}{{if .Usage}} - {{.Usage}}{{end}}
%s:
   {{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Version}}{{if not .HideVersion}}
%s:
   {{.Version}}{{end}}{{end}}{{if .Description}}
%s:
   {{.Description | nindent 3 | trim}}{{end}}{{if len .Authors}}
AUTHOR{{with $length := len .Authors}}{{if ne 1 $length}}S{{end}}{{end}}:
   {{range $index, $author := .Authors}}{{if $index}}
   {{end}}{{$author}}{{end}}{{end}}{{if .VisibleCommands}}
%s:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{else}}{{range .VisibleCommands}}
   {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}{{end}}{{if .VisibleFlags}}
%s:
   {{range $index, $option := .VisibleFlags}}{{if $index}}
   {{end}}{{$option}}{{end}}{{end}}{{if .Copyright}}
%s:
   {{.Copyright}}{{end}}
`

// List of colors
var (
	cyan  = color.New(color.FgCyan)
	green = color.New(color.FgGreen)
)

func getColoredHeaders(diaplayColor *color.Color) string {
	name := diaplayColor.Sprint("NAME")
	version := diaplayColor.Sprint("VERSION")
	description := diaplayColor.Sprint("DESCRIPTION")
	commands := diaplayColor.Sprint("COMMANDS")
	globalOptions := diaplayColor.Sprint("GLOBAL OPTIONS")
	copyright := diaplayColor.Sprint("COPYRIGHT")

	return fmt.Sprintf(AppHelpTemplate, name, version, description, commands,
		globalOptions, copyright)
}
