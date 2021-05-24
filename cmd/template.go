package cmd

import (
	"fmt"

	"github.com/fatih/color"
)

// List of colors
var (
	cyan  = color.New(color.FgCyan)
	green = color.New(color.FgGreen)
	blue  = color.New(color.FgBlue)
)

/**
* AppHelpTemplate is a app formating string for:
* NAME, VERSION, USAGE, DESCRIPTION, COMMANDS, GLOBAL OPTIONS, COPYRIGHT
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

/**
* CommandHelpTemplate is a command formating string for:
* NAME, USAGE, CATEGORY, DESCRIPTION, OPTIONS
**/
var CommandHelpTemplate = `
%s:
   {{.HelpName}} - {{.Usage}}
%s:
   {{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}}{{if .VisibleFlags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Category}}
%s:
   {{.Category}}{{end}}{{if .Description}}
%s:
   {{.Description | nindent 3 | trim}}{{end}}{{if .VisibleFlags}}
%s:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{else}}{{range .VisibleCommands}}
   {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}{{end}}{{if .VisibleFlags}}
%s:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}
`

func GetColoredCommandHeaders(displaycolor *color.Color) string {
	name := displaycolor.Sprint("NAME")
	usage := displaycolor.Sprint("USAGE")
	category := displaycolor.Sprint("CATEGORY")
	description := displaycolor.Sprint("DESCRIPTION")
	commands := displaycolor.Sprint("COMMANDS")
	options := displaycolor.Sprint("OPTIONS")

	return fmt.Sprintf(CommandHelpTemplate, name, usage, category, description, commands, options)
}

/**
* SubCommandHelpTemplate is a formating string for:
* NAME, USAGE, DESCRIPTION, COMMANDS, OPTIONS
**/
var SubCommandHelpTemplate = `
%s:
   {{.HelpName}} - {{.Usage}}
%s:
   {{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}} command{{if .VisibleFlags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Description}}
%s:
   {{.Description | nindent 3 | trim}}{{end}}
%s:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{else}}{{range .VisibleCommands}}
   {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}{{if .VisibleFlags}}
%s:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}
`

func GetColoredSubCommandHeaders(displaycolor *color.Color) string {
	name := displaycolor.Sprint("NAME")
	usage := displaycolor.Sprint("USAGE")
	description := displaycolor.Sprint("DESCRIPTION")
	commands := displaycolor.Sprint("COMMANDS")
	options := displaycolor.Sprint("OPTIONS")

	return fmt.Sprintf(SubCommandHelpTemplate, name, usage, description, commands, options)
}
