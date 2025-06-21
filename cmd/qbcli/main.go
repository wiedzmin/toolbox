package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/browsers/qutebrowser"
	"github.com/wiedzmin/toolbox/impl/redis"
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

func open(ctx *cli.Context) error {
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

	return qutebrowser.Execute([]string{
		fmt.Sprintf(":open %s %s", openParam, ctx.String("url")),
	})
}

func saveSession(ctx *cli.Context) error {
	var sessionName string
	if ctx.String("name") != "" {
		sessionName = ctx.String("name")
	} else {
		sessionName = fmt.Sprintf("session-%s", impl.CommonNowTimestamp(false))
	}

	return qutebrowser.Execute([]string{
		fmt.Sprintf(":session-save --quiet %s", sessionName),
		":session-save --quiet",
	})
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "Qbcli"
	app.Usage = "CLI for Qutebrowser, with help of IPC"
	app.Description = "Qbcli"
	app.Version = "0.0.1#master"

	app.Commands = cli.Commands{
		{
			Name:   "open",
			Usage:  "Open URL",
			Action: open,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "url",
					Aliases:  []string{"u"},
					Usage:    "URL to open",
					Required: true,
				},
			},
		},
		{
			Name:   "save-session",
			Usage:  "Save current browser session",
			Action: saveSession,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "name",
					Aliases:  []string{"n"},
					Usage:    "Session name",
					Required: false,
				},
			},
		},
	}
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
