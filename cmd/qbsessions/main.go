package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/browsers/qutebrowser"
	"github.com/wiedzmin/toolbox/impl/fs"
	"github.com/wiedzmin/toolbox/impl/ui"
)

func saveSession(name *string) error {
	sessionName := fmt.Sprintf("session-%s", impl.CommonNowTimestamp())
	if name != nil {
		sessionName = *name
	}
	socketPath, err := qutebrowser.SocketPath()
	if err != nil {
		return err
	}
	err = impl.SendToUnixSocket(*socketPath, qutebrowser.CommandsJSON([]string{
		fmt.Sprintf(":session-save --quiet %s", sessionName),
		":session-save --quiet",
	}))
	if _, ok := err.(impl.FileErrNotExist); ok {
		ui.NotifyCritical("[qutebrowser]", fmt.Sprintf("cannot access socket at `%s`\nIs qutebrowser running?", *socketPath))
		os.Exit(0)
	}
	return err
}

func selectSession(path string) (*string, error) {
	files, err := fs.CollectFiles(path, false, nil)
	if err != nil {
		return nil, err
	}
	sessionName, err := ui.GetSelectionRofi(files, "export")
	if err != nil {
		return nil, err
	}
	return &sessionName, nil
}

func exportSession(sessionsPath, sessionName, exportPath string, format qutebrowser.SessionFormat) error {
	session, err := qutebrowser.LoadSession(fmt.Sprintf("%s/%s", sessionsPath, sessionName))
	if err != nil {
		return err
	}
	return qutebrowser.SaveSession(fmt.Sprintf("%s/%s.org",
		exportPath, strings.Split(sessionName, ".")[0]), session, format)
}

func perform(ctx *cli.Context) error {
	sessionsPath, err := qutebrowser.RawSessionsPath()
	if err != nil {
		return err
	}
	if ctx.Bool("save") {
		return saveSession(nil)
	}
	if ctx.Bool("save-named") {
		name, err := ui.GetSelectionDmenu([]string{}, "save as", 1, ctx.String("selector-font"))
		if err != nil {
			return err
		}
		return saveSession(&name)
	}
	if ctx.Bool("rotate") {
		return fs.RotateOlderThan(*sessionsPath, fmt.Sprintf("%dm", ctx.Int("keep-minutes")), &qutebrowser.RegexTimedSessionName)
	}
	exportFormat := qutebrowser.SESSION_FORMAT_ORG
	if ctx.Bool("flat") {
		exportFormat = qutebrowser.SESSION_FORMAT_ORG_FLAT
	}
	if ctx.Bool("export") {
		sessionName, err := selectSession(*sessionsPath)
		if err != nil {
			return err
		}
		return exportSession(*sessionsPath, *sessionName, ctx.String("export-path"), exportFormat)
	}
	if ctx.Bool("export-all") {
		files, err := fs.CollectFiles(*sessionsPath, false, nil)
		if err != nil {
			return err
		}
		for _, f := range files {
			err = exportSession(*sessionsPath, f, ctx.String("export-path"), exportFormat)
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
			Name:     "selector-font",
			Aliases:  []string{"f"},
			EnvVars:  []string{impl.EnvPrefix + "_SELECTOR_FONT"},
			Usage:    "Font to use for selector application, e.g. dmenu, rofi, etc.",
			Required: false,
		},
	}
	app.Action = perform
	return app
}

func main() {
	app := createCLI()
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
