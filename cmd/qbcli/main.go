package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/browsers/qutebrowser"
	"github.com/wiedzmin/toolbox/impl/redis"
	"github.com/wiedzmin/toolbox/impl/ui"
	"github.com/wiedzmin/toolbox/impl/xserver/xkb"
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

func query(ctx *cli.Context, prompt string) (string, error) {
	xkb.EnsureEnglishKeyboardLayout()
	value, err := ui.GetSelection([]string{}, prompt, ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), true, false)
	if err != nil {
		return "", err
	}
	return value, nil
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

	var url string
	if ctx.Bool("query") {
		url, err = query(ctx, "URL")
		if err != nil {
			return err
		}
	} else {
		url = ctx.String("url")
	}

	return qutebrowser.Execute([]string{
		fmt.Sprintf(":open %s %s", openParam, url),
	})
}

func saveSession(ctx *cli.Context) error {
	if ctx.Bool("query") {
		sessionName, err := query(ctx, "save as")
		if err != nil {
			return err
		}
		return qutebrowser.SaveSessionInternal(sessionName)
	}
	return qutebrowser.SaveSessionInternal(ctx.String("name"))
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
				&cli.BoolFlag{
					Name:     "query",
					Usage:    "Whether to show selector for argument input",
					Required: false,
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
				&cli.BoolFlag{
					Name:     "query",
					Usage:    "Whether to show selector for argument input",
					Required: false,
				},
			},
		},
	}
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     impl.SelectorFontFlagName,
			Aliases:  []string{"f"},
			EnvVars:  []string{impl.EnvPrefix + "_SELECTOR_FONT"},
			Usage:    "Font to use for selector application, e.g. dmenu, rofi, etc.",
			Required: false,
		},
		&cli.StringFlag{
			Name:     ui.SelectorToolFlagName,
			Aliases:  []string{"T"},
			EnvVars:  []string{impl.EnvPrefix + "_SELECTOR_TOOL"},
			Value:    ui.SelectorToolDefault,
			Usage:    "Selector tool to use, e.g. dmenu, rofi, etc.",
			Required: false,
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
