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
%s{{with $length := len .Authors}}{{if ne 1 $length}}S{{end}}{{end}}:
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
	blue  = color.New(color.FgBlue)
)

func GetColoredHeaders(displaycolor *color.Color) string {
	name := displaycolor.Sprint("NAME")
	usage := displaycolor.Sprint("USAGE")
	version := displaycolor.Sprint("VERSION")
	description := displaycolor.Sprint("DESCRIPTION")
	author := displaycolor.Sprint("AUTHOR")
	commands := displaycolor.Sprint("COMMANDS")
	globalOptions := displaycolor.Sprint("GLOBAL OPTIONS")
	copyright := displaycolor.Sprint("COPYRIGHT")

	return fmt.Sprintf(AppHelpTemplate, name, usage, version, description, author, commands, globalOptions, copyright)
}
