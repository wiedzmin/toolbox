package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/browsers/qutebrowser"
	"github.com/wiedzmin/toolbox/impl/emacs"
	"github.com/wiedzmin/toolbox/impl/redis"
	"github.com/wiedzmin/toolbox/impl/ui"
	"go.uber.org/zap"
)

const (
	WATCH_FILE                        = "/tmp/qbtarget"
	BROWSE_URL_USE_TABS_VARIABLE_NAME = "browse-url-qutebrowser-new-window-is-tab"
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
	l.Debugw("[qbtarget.getCurrentTarget]", "key", qutebrowser.URL_TARGET_KEYNAME, "value", target)
	return target, nil
}

func perform(ctx *cli.Context) error {
	l := logger.Sugar()

	var target, emacsUseTabs string

	if ctx.Bool("notify-status") {
		target, err := getCurrentTarget()
		if err != nil {
			return err
		}
		ui.NotifyNormal("[qbtarget]", fmt.Sprintf("url target is `%s`", target))
		return nil
	} else {
		targetParam := ctx.String("target")
		switch targetParam {
		case "tab":
			target = "tab"
			emacsUseTabs = "t"
		case "window":
			target = "window"
			emacsUseTabs = "nil"
		case "":
			target, err := getCurrentTarget()
			if err != nil {
				return err
			}
			targetStr := strings.ToUpper(target)
			if ctx.Bool("colorize") && ctx.String("foreground") != "" {
				targetStr = fmt.Sprintf("<span foreground=\"%s\">%s</span>", ctx.String("foreground"), targetStr)
			}
			io.WriteString(os.Stdout, fmt.Sprintf("%s\n", targetStr))
			return nil
		default:
			return fmt.Errorf("unknown url target '%s'", targetParam)
		}
		socketPath, err := qutebrowser.SocketPath()
		if err != nil {
			return err
		}
		resp := qutebrowser.Request{Commands: []string{
			fmt.Sprintf(":set %s %s", qutebrowser.URL_TARGET_SETTING, target),
		}}
		err = emacs.SendToServer(fmt.Sprintf("(setq %s %s)", BROWSE_URL_USE_TABS_VARIABLE_NAME, emacsUseTabs), false)
		if err != nil {
			return err
		}

		l.Debugw("[perform]", "request", r)
		rb, err := resp.Marshal()
		if err != nil {
			return err
		}
		err = impl.SendToUnixSocket(*socketPath, rb)
		if _, ok := err.(impl.FileErrNotExist); ok {
			ui.NotifyCritical("[qutebrowser]", fmt.Sprintf("cannot access socket at `%s`\nIs qutebrowser running?", *socketPath))
			os.Exit(0)
		}

		err = r.SetValue(qutebrowser.URL_TARGET_KEYNAME, target)
		if err != nil {
			return err
		}

		watchData := []byte(fmt.Sprintf("target: %s\n", target))
		err = os.WriteFile(WATCH_FILE, watchData, 0644)
		if err != nil {
			return err
		}

		l.Debugw("[qbtarget.perform]", "status", fmt.Sprintf("url target set to `%s`", target), qutebrowser.URL_TARGET_KEYNAME, target)
		ui.NotifyNormal("[qbtarget]", fmt.Sprintf("url target set to `%s`", target))
	}

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
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "notify-status",
			Aliases:  []string{"s"},
			Usage:    "Show currently set target notification",
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "colorize",
			Aliases:  []string{"c"},
			Usage:    "Whether to colorize text foreground using Pango <span> markup",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "foreground",
			Aliases:  []string{"fg"},
			Usage:    "Text foreground color to use. No values validation provided",
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
