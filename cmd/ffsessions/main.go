package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/browsers"
	"github.com/wiedzmin/toolbox/impl/browsers/firefox"
	"github.com/wiedzmin/toolbox/impl/emacs"
	"github.com/wiedzmin/toolbox/impl/fs"
	"github.com/wiedzmin/toolbox/impl/ui"
	"go.uber.org/zap"
)

var logger *zap.Logger

func dump(ctx *cli.Context) error {
	sessionsPath := firefox.RawSessionsPath()

	// TODO: check/investigate cases, where we really need "previous.jsonlz4" here
	sourceSessionPreviousFile := fmt.Sprintf("%s/previous.jsonlz4", sessionsPath)
	sourceSessionRecoveryFile := fmt.Sprintf("%s/recovery.jsonlz4", sessionsPath)
	sourceSessionFile := sourceSessionPreviousFile
	if fs.FileExists(sourceSessionRecoveryFile) {
		sourceSessionFile = sourceSessionRecoveryFile
	}

	session, err := firefox.LoadSession(sourceSessionFile)
	if err != nil {
		return err
	}

	var sessionFormat firefox.SessionFormat
	var sessionExtension string
	switch {
	case ctx.Bool("json"):
		sessionFormat = firefox.SESSION_FORMAT_JSON
		sessionExtension = "json"
	case ctx.Bool("flat"):
		sessionFormat = firefox.SESSION_FORMAT_ORG_FLAT
		sessionExtension = "org"
	default:
		sessionFormat = firefox.SESSION_FORMAT_ORG
		sessionExtension = "org"
	}

	var sessionName string
	if ctx.String("out") != "" {
		sessionName = ctx.String("out")
	} else {
		sessionName = fmt.Sprintf("%s-%s.%s", ctx.String("dump-basename"), impl.CommonNowTimestamp(false), sessionExtension)
	}

	return firefox.DumpSession(
		fmt.Sprintf("%s/%s", ctx.String("dumps-path"), sessionName),
		session,
		sessionFormat,
		ctx.Bool("raw"),
		ctx.Bool("keep-tabs-history"),
	)
}

func edit(ctx *cli.Context) error {
	l := logger.Sugar()
	sessionName, err := browsers.SelectSession(ctx.String("dumps-path"), "edit", ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), nil, nil)
	l.Debugw("[edit]", "sessionName", sessionName, "err", err)
	if err != nil {
		return err
	}
	return emacs.SendToServer(fmt.Sprintf("(find-file \"%s/%s\")", ctx.String("dumps-path"), *sessionName), true)
}

func remove(ctx *cli.Context) error {
	l := logger.Sugar()
	sessionName, err := browsers.SelectSession(ctx.String("dumps-path"), "remove", ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), nil, nil)
	l.Debugw("[remove]", "sessionName", sessionName, "err", err)
	if err != nil {
		return err
	}
	sessionPath := fmt.Sprintf("%s/%s", ctx.String("dumps-path"), *sessionName)
	err = os.Remove(sessionPath)
	if err != nil {
		return err
	}
	l.Debugw("[remove]", "removed", sessionPath)
	ui.NotifyNormal("[ffsessions]", fmt.Sprintf("Removed %s", sessionPath))
	return nil
}

func rotate(ctx *cli.Context) error {
	return fs.RotateOlderThan(ctx.String("dumps-path"), fmt.Sprintf("%dm", ctx.Int("keep-minutes")), nil)
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "Ffsessions"
	app.Usage = "Firefox sessions management tool"
	app.Description = "Ffsessions"
	app.Version = "0.0.1#master"

	app.Commands = cli.Commands{
		{
			Name:   "dump",
			Usage:  "Dump session",
			Action: dump,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:     "raw",
					Usage:    "Dump raw URLs without descriptions/titles",
					Required: false,
				},
				&cli.BoolFlag{
					Name:     "json",
					Aliases:  []string{"j"},
					Usage:    "Dump session in JSON format",
					Required: false,
				},
				&cli.BoolFlag{
					Name:     "flat",
					Usage:    "Dump flat Org session layout, without windows breakdown",
					Required: false,
				},
				&cli.StringFlag{
					Name:     "out",
					Aliases:  []string{"o"},
					Usage:    "Dump filename",
					Required: false,
				},
				&cli.StringFlag{
					Name:  "dump-basename",
					Usage: "Dump basename",
					Value: "firefox-session-auto",
				},
				&cli.BoolFlag{
					Name:     "keep-tabs-history",
					Aliases:  []string{"k"},
					Value:    false,
					Usage:    "Also dump links from tab history",
					Required: false,
				},
			},
		},
		{
			Name:   "edit",
			Usage:  "select and edit one of saved sessions",
			Action: edit,
		},
		{
			Name:   "remove",
			Usage:  "select and remove one of saved sessions",
			Action: remove,
		},
		{
			Name:   "rotate",
			Usage:  "rotate saved sessions",
			Action: rotate,
			Flags: []cli.Flag{
				&cli.IntFlag{ // FIXME: implement
					Name:     "keep-count",
					Usage:    "Number of last sessions to keep",
					Required: false,
				},
				&cli.IntFlag{
					Name:     "keep-minutes",
					Aliases:  []string{"k"},
					EnvVars:  []string{impl.EnvPrefix + "_FIREFOX_SESSIONS_KEEP_MINUTES"},
					Usage:    "Rotate sessions older than it",
					Required: true,
				},
			},
		},
	}
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "dumps-path",
			Usage:    "Path to store dumps under",
			Required: true,
		},
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
	app := createCLI()
	err := app.Run(os.Args)
	if err != nil {
		l.Errorw("[main]", "err", err)
	}
}
