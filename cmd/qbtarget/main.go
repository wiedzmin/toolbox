package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/browsers/qutebrowser"
	"github.com/wiedzmin/toolbox/impl/ui"
	"go.uber.org/zap"
)

const URL_TARGET_SETTING = "new_instance_open_target"

var logger *zap.Logger

func perform(ctx *cli.Context) error {
	l := logger.Sugar()
	targetParam := ctx.String("target")
	var target string
	switch targetParam {
	case "tab":
		target = "tab"
	case "window":
		target = "window"
	default:
		return fmt.Errorf("unknown url target '%s'", targetParam)
	}
	socketPath, err := qutebrowser.SocketPath()
	if err != nil {
		return err
	}
	r := qutebrowser.Request{Commands: []string{
		fmt.Sprintf(":set %s %s", URL_TARGET_SETTING, target),
	}}

	l.Debugw("[perform]", "request", r)
	rb, err := r.Marshal()
	if err != nil {
		return err
	}
	err = impl.SendToUnixSocket(*socketPath, rb)
	if _, ok := err.(impl.FileErrNotExist); ok {
		ui.NotifyCritical("[qutebrowser]", fmt.Sprintf("cannot access socket at `%s`\nIs qutebrowser running?", *socketPath))
		os.Exit(0)
	}
	l.Debugw("[qbtarget.perform]", "status", fmt.Sprintf("url target set to `%s`", target))
	ui.NotifyNormal("[qbtarget]", fmt.Sprintf("url target set to `%s`", target))

	return nil
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "Qbtarget"
	app.Usage = "Qutebrowser url target switching tool"
	app.Description = "Qbtarget"
	app.Version = "0.0.1#master"

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "target",
			Aliases:  []string{"t"},
			Usage:    "URL target to use from now on",
			Required: true,
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
