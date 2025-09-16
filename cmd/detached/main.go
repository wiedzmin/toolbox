package main

import (
	"os"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/shell"
	"github.com/wiedzmin/toolbox/impl/xserver"
	"go.uber.org/zap"
)

var logger *zap.Logger

func perform(ctx *cli.Context) error {
	err := shell.RunDetached(ctx.String("command"))
	if err != nil {
		return err
	}
	if ctx.String("input") != "" {
		inp := ctx.String("input")
		err = xserver.WriteClipboard(&inp, false)
		if err != nil {
			return err
		}
	}
	return nil
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "detached"
	app.Usage = "detached"
	app.Description = "Run programs, detached from parent"
	app.Version = "0.0.1#master"

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "command",
			Aliases:  []string{"c"},
			Usage:    "command to run (with parameters)",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "input",
			Aliases:  []string{"i"},
			Usage:    "Initial input, if needed",
			Required: false,
		},
	}
	app.Action = perform
	return app
}

func main() {
	logger = impl.NewLogger()
	defer logger.Sync()
	l := logger.Sugar()
	app := createCLI()
	err := app.Run(os.Args)
	if err != nil {
		l.Errorw("[main]", "err", err)
	}
}
