package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/browsers/qutebrowser"
	"github.com/wiedzmin/toolbox/impl/redis"
	"github.com/wiedzmin/toolbox/impl/ui"
	"go.uber.org/zap"
)

var (
	logger *zap.Logger
	r      *redis.Client
)

func getCurrentTarget() (string, error) {
	l := logger.Sugar()
	value, err := r.GetValue(qutebrowser.URL_TARGET_KEYNAME)
	if err != nil {
		return "", err
	}
	target := string(value)
	if target == "" {
		target = "unknown"
	}
	l.Debugw("[qbopen.getCurrentTarget]", "key", qutebrowser.URL_TARGET_KEYNAME, "value", target)
	return target, nil
}

func perform(ctx *cli.Context) error {
	l := logger.Sugar()

	var target, openParam string

	target, err := getCurrentTarget()
	if err != nil {
		return err
	}

	switch target {
	case "tab":
		openParam = "-t"
	case "window":
		openParam = "-w"
	default:
		return fmt.Errorf("unknown url target '%s'", target)
	}

	resp := qutebrowser.Request{Commands: []string{
		fmt.Sprintf(":open %s %s", openParam, ctx.String("url")),
	}}
	l.Debugw("[perform]", "request", r)
	rb, err := resp.Marshal()
	if err != nil {
		return err
	}
	socketPath, err := qutebrowser.SocketPath()
	if err != nil {
		return err
	}
	err = impl.SendToUnixSocket(*socketPath, rb)
	if _, ok := err.(impl.FileErrNotExist); ok {
		ui.NotifyCritical("[qutebrowser]", fmt.Sprintf("cannot access socket at `%s`\nIs qutebrowser running?", *socketPath))
		os.Exit(0)
	}

	return nil
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "Qbopen"
	app.Usage = "Qutebrowser url opener, using IPC"
	app.Description = "Qbopen"
	app.Version = "0.0.1#master"

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "url",
			Aliases:  []string{"u"},
			Usage:    "URL to open",
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

	var err error

	r, err = redis.NewRedisLocal()
	if err != nil {
		l.Errorw("[main]", "err", err)
	}
	app := createCLI()
	err = app.Run(os.Args)
	if err != nil {
		l.Errorw("[main]", "err", err)
	}
}
