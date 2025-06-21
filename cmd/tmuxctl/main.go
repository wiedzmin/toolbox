package main

import (
	"os"
	"sort"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/shell/tmux/tmuxp"
	"github.com/wiedzmin/toolbox/impl/ui"
	"github.com/wiedzmin/toolbox/impl/xserver/xkb"
	"go.uber.org/zap"
)

var logger *zap.Logger

func perform(ctx *cli.Context) error {
	sessions, err := tmuxp.CollectSessions(tmuxp.SessionsRootDefault())
	if err != nil {
		return err
	}
	sessionsByName := make(map[string]tmuxp.Session)
	var names []string
	for _, s := range sessions {
		names = append(names, s.Name)
		sessionsByName[s.Name] = s
	}
	sort.Strings(names)
	xkb.EnsureEnglishKeyboardLayout()
	sessionName, err := ui.GetSelection(names, "load", ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), true, false)
	if err != nil {
		return err
	}
	session, ok := sessionsByName[sessionName]
	if !ok {
		return tmuxp.ErrSessionNotFound{Name: sessionName}
	}
	return session.Load(false)
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "Tmuxctl"
	app.Usage = "Lists and loads Tmuxp sessions"
	app.Description = "Tmuxctl"
	app.Version = "0.0.1#master"

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
