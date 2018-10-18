// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package console

import (
	"flag"

	"github.com/urfave/cli"
)

// NOTE: console package type alias declarations using library `github.com/urfave/cli`.
// It keeps decoupled from thrid party library for aah user and also opens up avenue
// for smooth migration to new library if need be.

type (
	// App is the main structure of a console application.
	// It is recommended that an app be created with the func `console.NewApp()`.
	App = cli.App

	// Command returns the named command on App. Returns nil if the command
	// does not exist
	Command = cli.Command

	// Args contains apps console arguments
	Args = cli.Args

	// Context is a type that is passed through to each Handler action in a console application.
	// Context can be used to retrieve context-specific Args and parsed command-line options.
	Context = cli.Context

	// Flag is a common interface related to parsing flags in console. For more
	// advanced flag parsing techniques, it is recommended that this interface
	// be implemented.
	Flag = cli.Flag

	// StringFlag is a flag with type string
	StringFlag = cli.StringFlag

	// BoolFlag is a flag with type bool
	BoolFlag = cli.BoolFlag

	// IntFlag is a flag with type int
	IntFlag = cli.IntFlag

	// Int64Flag is a flag with type int64
	Int64Flag = cli.Int64Flag

	// Float64Flag is a flag with type float64
	Float64Flag = cli.Float64Flag

	// IntSlice is an opaque type for []int to satisfy flag.Value and flag.
	IntSlice = cli.IntSlice

	// StringSlice is an opaque type for []string to satisfy flag.Value and flag.
	StringSlice = cli.StringSlice

	// Author represents someone who has contributed to a console project.
	Author = cli.Author

	// FlagsByName is to sorter interface for flags
	FlagsByName = cli.FlagsByName
)

// NewApp creates a new console Application with some reasonable
// defaults for Name, Usage, Version and Action.
func NewApp() *App {
	return cli.NewApp()
}

// NewContext creates a new context. For use in when invoking an App or Command action.
func NewContext(app *App, set *flag.FlagSet, parentCtx *Context) *Context {
	return cli.NewContext(app, set, parentCtx)
}

// ShowAppHelp is an action that displays the help.
func ShowAppHelp(c *Context) error {
	return cli.ShowAppHelp(c)
}

// ShowAppHelpAndExit - Prints the list of subcommands for the app and exits with exit code.
func ShowAppHelpAndExit(c *Context, exitCode int) {
	cli.ShowAppHelpAndExit(c, exitCode)
}

// ShowCommandHelp prints help for the given command
func ShowCommandHelp(c *Context, cmd string) error {
	return cli.ShowCommandHelp(c, cmd)
}

// ShowCommandHelpAndExit - exits with code after showing help
func ShowCommandHelpAndExit(c *Context, cmd string, code int) {
	cli.ShowCommandHelpAndExit(c, cmd, code)
}

// ShowSubcommandHelp prints help for the given subcommand.
func ShowSubcommandHelp(c *Context) error {
	return cli.ShowSubcommandHelp(c)
}

// ShowVersion prints the version number of the App.
func ShowVersion(c *Context) {
	cli.ShowVersion(c)
}

// VersionFlag method customized flag name, desc for VersionFlag.
func VersionFlag(f BoolFlag) {
	cli.VersionFlag = f
}

// VersionPrinter method set custom func for version printer.
func VersionPrinter(vp func(*Context)) {
	cli.VersionPrinter = vp
}

func init() {
	cli.HelpFlag = cli.BoolFlag{
		Name:  "h, help",
		Usage: "Shows help",
	}

	cli.AppHelpTemplate = `Usage:
  {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}
{{if .Commands}}
Commands:
{{range .Commands}}{{if not .HideHelp}}  {{join .Names ", "}}{{ "\t   " }}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{end}}{{if .VisibleFlags}}
Global Options:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}
`

	cli.CommandHelpTemplate = `Name:
  {{.HelpName}} - {{.Usage}}

Usage:
  {{.HelpName}}{{if .VisibleFlags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{if .Category}}

Category:
  {{.Category}}{{end}}{{if .Description}}

Description:
  {{.Description}}{{end}}{{if .VisibleFlags}}

Options:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}
`
}
