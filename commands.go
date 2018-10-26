// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"fmt"

	"aahframe.work/console"
)

func (a *Application) addBulitinCliCommands() {

}

func (a *Application) cliCmdRun() console.Command {
	return console.Command{
		Name:    "run",
		Aliases: []string{"r"},
		Usage:   "Runs aah application binary",
		Description: `Runs aah application binary.
	
		Examples of short and long flags:
		<app-binary> run
		<app-binary> run -e qa
	
		<app-binary> run
		<app-binary> run -e qa
		<app-binary> run -e qa -c /path/to/config/external.conf
	
		<app-binary> run
		<app-binary> run --envprofile qa
		<app-binary> run --envprofile qa --config /path/to/config/external.conf`,
		Flags: []console.Flag{
			console.StringFlag{
				Name:  "envprofile, e",
				Usage: "Environment profile name to activate (e.g: dev, qa, prod)",
			},
			console.StringFlag{
				Name:  "config, c",
				Usage: "External config file for overriding aah configuration (*.conf) values",
			},
			console.StringFlag{
				Name:  "diagnosis, d",
				Usage: "Enabling aah application diagnosis and profiling",
			},
			console.StringFlag{
				Name:   "importpath, i",
				Usage:  "Environment profile name to activate (e.g: dev, qa, prod)",
				Hidden: true,
			},
		},
		Action: func(c *console.Context) error {
			fmt.Println("aah application bianry run called")

			fmt.Println(c.String("d"), c.String("diagnosis"))
			fmt.Println(c.String("e"), c.String("envprofile"))
			fmt.Println(c.String("c"), c.String("config"))

			if err := a.Init(c.String("importpath")); err != nil {
				return err
			}
			return errors.New("testing cli error flow")
		},
	}
}

// var cmdRun = console.Command{
// 	Name:    "run",
// 	Aliases: []string{"r"},
// 	Usage:   "Runs aah application binary",
// 	Description: `Runs aah application binary.

// 	Examples of short and long flags:
//     <app-binary> run
// 	<app-binary> run -e qa

// 	<app-binary> run
// 	<app-binary> run -e qa
// 	<app-binary> run -e qa -c /path/to/config/external.conf

//     <app-binary> run
// 	<app-binary> run --envprofile qa
// 	<app-binary> run --envprofile qa --config /path/to/config/external.conf`,
// 	Flags: []console.Flag{
// 		console.StringFlag{
// 			Name:  "envprofile, e",
// 			Usage: "Environment profile name to activate (e.g: dev, qa, prod)",
// 		},
// 		console.StringFlag{
// 			Name:  "config, c",
// 			Usage: "External config file for overriding aah configuration (*.conf) values",
// 		},
// 		console.StringFlag{
// 			Name:  "diagnosis, d",
// 			Usage: "Enabling aah application diagnosis and profiling",
// 		},
// 		console.StringFlag{
// 			Name:   "importpath, i",
// 			Usage:  "Environment profile name to activate (e.g: dev, qa, prod)",
// 			Hidden: true,
// 		},
// 	},
// 	Action: defaultApp.actionRun,
// }

// func (a *Application) actionRun(c *console.Context) error {
// 	fmt.Println("aah application bianry run called")

// 	fmt.Println(c.String("d"), c.String("diagnosis"))
// 	fmt.Println(c.String("e"), c.String("envprofile"))
// 	fmt.Println(c.String("c"), c.String("config"))

// 	if err := a.Init(c.String("importpath")); err != nil {
// 		return err
// 	}
// 	return errors.New("testing cli error flow")
// }
