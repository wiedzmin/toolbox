package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/browsers"
	"github.com/wiedzmin/toolbox/impl/browsers/qutebrowser"
	"github.com/wiedzmin/toolbox/impl/fs"
	"github.com/wiedzmin/toolbox/impl/ui"
	"go.uber.org/zap"
)

var logger *zap.Logger

func exportSession(sessionsPath, sessionName, exportPath string, format qutebrowser.SessionFormat) error {
	session, err := qutebrowser.LoadSession(fmt.Sprintf("%s/%s", sessionsPath, sessionName))
	if err != nil {
		return err
	}
	return qutebrowser.DumpSession(fmt.Sprintf("%s/%s.org",
		exportPath, strings.Split(sessionName, ".")[0]), session, format)
}

func perform(ctx *cli.Context) error {
	sessionsPath := qutebrowser.RawSessionsPath()
	if ctx.Bool("rotate") {
		return fs.RotateOlderThan(sessionsPath, fmt.Sprintf("%dm", ctx.Int("keep-minutes")), &browsers.RegexTimedSessionName)
	}
	exportFormat := qutebrowser.SESSION_FORMAT_ORG
	if ctx.Bool("flat") {
		exportFormat = qutebrowser.SESSION_FORMAT_ORG_FLAT
	}
	if ctx.Bool("export") {
		sessionName, err := browsers.SelectSession(sessionsPath, "export", ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), nil, nil)
		if err != nil {
			return err
		}
		return exportSession(sessionsPath, *sessionName, ctx.String("export-path"), exportFormat)
	}
	if ctx.Bool("export-all") {
		for _, f := range fs.NewFSCollection(sessionsPath, nil, nil, false).Emit(false) {
			err := exportSession(sessionsPath, f, ctx.String("export-path"), exportFormat)
			if err != nil {
				return err
			}
		}
	}
	// TODO: add "fix" implementation

	return nil
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "Qbsessions"
	app.Usage = "Qutebrowser sessions management tool"
	app.Description = "Qbsessions"
	app.Version = "0.0.1#master"

	// TODO: stabilize rethought [[file:~/workspace/repos/github.com/wiedzmin/toolbox/cmd/ffsessions/main.go::func createCLI() *cli.App {][ffsessions CLI]]  and rework qbsessions' one after it
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:     "save",
			Usage:    "Save current session",
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "save-named",
			Usage:    "Save current session under particular name",
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "export",
			Usage:    "Select session and export it to Org format",
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "export-all",
			Usage:    "Export all sessions to Org format",
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "flat",
			Usage:    "Export session in flat layout, instead of default per-window layout",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "export-path",
			Aliases:  []string{"p"},
			EnvVars:  []string{impl.EnvPrefix + "_DEFAULT_BROWSER_SESSIONS_STORE"},
			Usage:    "Path to export under",
			Required: true,
		},
		&cli.BoolFlag{
			Name:     "rotate",
			Usage:    "Rotate saved sessions",
			Required: false,
		},
		&cli.IntFlag{
			Name:     "keep-minutes",
			Aliases:  []string{"k"},
			EnvVars:  []string{impl.EnvPrefix + "_QUTEBROWSER_SESSIONS_KEEP_MINUTES"},
			Usage:    "Rotate sessions older than it",
			Required: true,
		},
		&cli.BoolFlag{
			Name:     "fix",
			Usage:    "Select session and fix it",
			Required: false,
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
