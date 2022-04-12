package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/bookmarks"
	"github.com/wiedzmin/toolbox/impl/emacs"
	"github.com/wiedzmin/toolbox/impl/shell"
	"github.com/wiedzmin/toolbox/impl/systemd"
	"github.com/wiedzmin/toolbox/impl/ui"
	"github.com/wiedzmin/toolbox/impl/xserver/xkb"
	"go.uber.org/zap"
)

var logger *zap.Logger

func open(ctx *cli.Context) error {
	l := logger.Sugar()
	bookmarks, err := bookmarks.BookmarksFromRedis("nav/bookmarks")
	if err != nil {
		return err
	}
	xkb.EnsureEnglishKeyboardLayout()
	key, err := ui.GetSelection(bookmarks.Keys(), "open", true, false, ctx.String(impl.SelectorFontFlagName))
	l.Debugw("[open]", "key", key, "err", err)
	if err != nil {
		return err
	}
	if bookmark := bookmarks.Get(key); bookmark == nil {
		l.Errorw("[open]", "failed to get bookmark metadata for", key)
	} else {
		if bookmark.Path == "" {
			l.Errorw("[open]", "missing bookmark path", key)
		} else {
			if bookmark.Shell {
				l.Errorw("[open]", "open shell", "not implemented")
			}
			emacsService := systemd.Unit{Name: "emacs.service", User: true}
			isActive, err := emacsService.IsActive()
			if err != nil {
				return err
			}
			if !isActive {
				l.Errorw("[open]", "`emacs` service", "not running")
				ui.NotifyCritical("[bookmarks]", "Emacs service not running")
				os.Exit(1)
			}
			fi, err := os.Stat(bookmark.Path)
			if err != nil {
				return err
			}
			elispCmd := fmt.Sprintf("(open-project \"%s\")", bookmark.Path)
			if !fi.IsDir() {
				elispCmd = fmt.Sprintf("(find-file \"%s\")", bookmark.Path)
			}
			l.Debugw("[open]", "elispCmd", elispCmd)
			return emacs.SendToServer(elispCmd)
		}
	}
	return nil
}

func search(ctx *cli.Context) error {
	l := logger.Sugar()
	xkb.EnsureEnglishKeyboardLayout()
	searchTerm, err := ui.GetSelection([]string{}, "token", true, false, ctx.String(impl.SelectorFontFlagName))
	if err != nil {
		l.Warnw("[search]", "no keyword provided")
		ui.NotifyCritical("[search repos]", "no keyword provided")
		return err
	}
	impl.EnsureBinary("fd", *logger)
	matchingRepos, err := shell.ShellCmd(fmt.Sprintf("fd -t d -d %d %s %s",
		ctx.Int("depth"), searchTerm, ctx.String("root")), nil, nil, nil, true, false)
	if err != nil {
		return err
	}
	matchingReposSlice := strings.Split(*matchingRepos, "\n")
	xkb.EnsureEnglishKeyboardLayout()
	path, err := ui.GetSelection(matchingReposSlice, "explore", true, false, ctx.String(impl.SelectorFontFlagName))
	if err != nil {
		l.Warnw("[search]", "no repository provided")
		ui.NotifyNormal("[search repos]", "no repository selected")
		return err
	}

	emacsService := systemd.Unit{Name: "emacs.service", User: true}
	l.Debugw("[search]", "emacsService", emacsService)
	isActive, err := emacsService.IsActive()
	if err != nil {
		return err
	}
	if !isActive {
		l.Errorw("[search]", "`emacs` service", "not running")
		ui.NotifyCritical("[search repos]", "Emacs service not running")
		os.Exit(1)
	}
	elispCmd := fmt.Sprintf("(open-project \"%s\")", path)
	l.Debugw("[search]", "elispCmd", elispCmd)
	return emacs.SendToServer(elispCmd)
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "Projects"
	app.Usage = "Open or fuzzy search project"
	app.Description = "Projects"
	app.Version = "0.0.1#master"

	app.Commands = cli.Commands{
		{
			Name:   "open",
			Usage:  "Open bookmarked project",
			Action: open,
		},
		{
			Name:   "search",
			Usage:  "Fuzzy search project repo",
			Action: search,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "root",
					Aliases:  []string{"r"},
					Usage:    "Project repos root directory",
					Required: true,
				},
				&cli.IntFlag{
					Name:     "depth",
					Aliases:  []string{"d"},
					Value:    4,
					Usage:    "Search depth",
					Required: true,
				},
				&cli.BoolFlag{
					Name:     "shell",
					Usage:    "spawn shell at selected path",
					Required: false,
				},
			},
		},
	}
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "tmux-session",
			Aliases:  []string{"t"},
			EnvVars:  []string{impl.EnvPrefix + "_TMUX_SESSION"},
			Usage:    "Default TMUX session to use",
			Required: false,
		},
		&cli.StringFlag{
			Name:     impl.SelectorFontFlagName,
			Aliases:  []string{"f"},
			EnvVars:  []string{impl.EnvPrefix + "_SELECTOR_FONT"},
			Usage:    "Font to use for selector application, e.g. dmenu, rofi, etc.",
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
