// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package console

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestConsoleMisc(t *testing.T) {
	app := NewApp()
	assert.NotNil(t, app)

	ctx := NewContext(app, nil, nil)
	assert.NotNil(t, ctx)

	VersionPrinter(func(c *Context) {
		t.Log("came here for version print")
	})

	ShowVersion(ctx)

	VersionFlag(BoolFlag{
		Name:  "v, version",
		Usage: "Prints test app version",
	})

	cli.AppHelpTemplate = ""
	cli.CommandHelpTemplate = ""
	cli.SubcommandHelpTemplate = ""
	ShowAppHelp(ctx)
	ShowCommandHelp(ctx, "help")
	ShowSubcommandHelp(ctx)
}
