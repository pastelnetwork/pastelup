package cmd

import (
	"fmt"

	"github.com/fatih/color"
)

// List of colors
var (
	blue   = color.New(color.FgHiBlue).SprintFunc()
	cyan   = color.New(color.FgCyan).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
)

// GetColoredHeaders returns the app help formatting
func GetColoredHeaders() string {
	return fmt.Sprintf(`%s {{if .Version}}{{if not .HideVersion}}{{.Version}}{{end}}{{end}}
	{{if .Usage}}{{.Usage}}{{end}}
	%s
		%s {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Description}}
	%s
		{{.Description}}{{end}}{{if len .Authors}}
	%s{{with $length := len .Authors}}{{if ne 1 $length}}%s{{end}}{{end}}%s
		{{range $index, $author := .Authors}}{{if $index}}
		{{end}}%s{{end}}{{end}}{{if .VisibleCommands}}
	%s{{range .VisibleCategories}}{{if .Name}}
		{{.Name}}:{{end}}{{range .VisibleCommands}}
		%s{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}{{if .VisibleFlags}}
	%s
		{{range $index, $option := .VisibleFlags}}{{if $index}}
		{{end}}{{$option}}{{end}}{{end}}{{if .Copyright}}
	%s{{end}}
	`, green("{{.Name}}"),
		yellow("USAGE:"),
		cyan("{{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}}"),
		yellow("DESCRIPTION:"),
		yellow("AUTHOR"),
		yellow("S"),
		yellow(":"),
		blue("{{$author}}"),
		yellow("COMMANDS:"),
		green(`{{join .Names ", "}}`),
		yellow("GLOBAL OPTIONS:"),
		red("{{.Copyright}}"))
}

// GetColoredCommandHeaders returns colored command formating
// NAME, USAGE, CATEGORY, DESCRIPTION, OPTIONS
func GetColoredCommandHeaders() string {
	return fmt.Sprintf(`%s
    {{.HelpName}} - {{.Usage}}
%s
    {{.HelpName}}{{if .VisibleFlags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{if .Category}}
%s
    {{.Category}}{{end}}{{if .Description}}
%s
    {{.Description}}{{end}}{{if .VisibleFlags}}
%s
    {{range .VisibleFlags}}{{.}}
    {{end}}{{end}}
`, yellow("NAME:"),
		yellow("USAGE:"),
		yellow("CATEGORY:"),
		yellow("DESCRIPTION:"),
		yellow("OPTIONS:"))
}

// GetColoredSubCommandHeaders returns colored formatting for subcommands
func GetColoredSubCommandHeaders() string {
	return fmt.Sprintf(`%s
    {{.HelpName}} - {{if .Description}}{{.Description}}{{else}}{{.Usage}}{{end}}
%s
    {{.HelpName}} command{{if .VisibleFlags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}
%s{{range .VisibleCategories}}{{if .Name}}
    {{.Name}}:{{end}}{{range .VisibleCommands}}
    {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}
{{end}}{{if .VisibleFlags}}
%s
    {{range .VisibleFlags}}{{.}}
    {{end}}{{end}}
`, yellow("NAME:"),
		yellow("USAGE:"),
		yellow("COMMANDS:"),
		yellow("OPTIONS:"))
}
