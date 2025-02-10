package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/bookmarks"
	"github.com/wiedzmin/toolbox/impl/emacs"
	"github.com/wiedzmin/toolbox/impl/shell"
	"github.com/wiedzmin/toolbox/impl/shell/tmux"
	"github.com/wiedzmin/toolbox/impl/ui"
	"github.com/wiedzmin/toolbox/impl/xserver"
	"github.com/wiedzmin/toolbox/impl/xserver/xkb"
	"go.uber.org/zap"
)

var logger *zap.Logger

func open(ctx *cli.Context) error {
	l := logger.Sugar()
	if !(ctx.Bool("copy-local") || ctx.Bool("shell")) {
		emacs.ServiceState("open", true)
	}

	var pathStr string
	if ctx.String("path") != "" {
		pathStr = ctx.String("path")
	} else {
		bookmarks, err := bookmarks.BookmarksFromRedis("nav/bookmarks")
		if err != nil {
			return err
		}
		var keyStr string
		if ctx.String("key") != "" {
			keyStr = ctx.String("key")
		} else {
			xkb.EnsureEnglishKeyboardLayout()
			keyStr, err = ui.GetSelection(bookmarks.Keys(), "open", ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), true, false)
			l.Debugw("[open]", "key", keyStr, "err", err)
			if err != nil {
				return err
			}
		}
		if bookmark := bookmarks.Get(keyStr); bookmark == nil {
			l.Errorw("[open]", "failed to get bookmark metadata for", keyStr)
		} else {
			if bookmark.Path == "" {
				l.Errorw("[open]", "missing bookmark path", keyStr)
			} else {
				if bookmark.Shell {
					// FIXME: consider not erroring out, think of more useful reaction
					l.Errorw("[open]", "open shell", "not implemented")
				}
				pathStr = bookmark.Path
			}
		}
	}

	if ctx.Bool("copy-local") {
		return xserver.WriteClipboard(&pathStr, false)
	} else if !ctx.Bool("shell") {
		fi, err := os.Stat(pathStr)
		if err != nil {
			return err
		}
		elispCmd := fmt.Sprintf("(open-project \"%s\")", pathStr)
		if !fi.IsDir() {
			elispCmd = fmt.Sprintf("(find-file \"%s\")", pathStr)
		}
		l.Debugw("[open]", "elispCmd", elispCmd)
		return emacs.SendToServer(elispCmd)
	} else {
		ui.NotifyNormal("[open]", fmt.Sprintf("opening terminal at %s", pathStr))
		return shell.OpenTerminal(pathStr, shell.TermTraitsFromContext(ctx))
	}
}

func search(ctx *cli.Context) error {
	l := logger.Sugar()
	if !(ctx.Bool("copy-local") || ctx.Bool("shell")) {
		emacs.ServiceState("search", true)
	}

	xkb.EnsureEnglishKeyboardLayout()
	searchTerm, err := ui.GetSelection([]string{}, "token", ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), true, false)
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
	var path string
	if len(*matchingRepos) > 0 {
		matchingReposSlice := strings.Split(*matchingRepos, "\n")
		xkb.EnsureEnglishKeyboardLayout()
		path, err = ui.GetSelection(matchingReposSlice, "explore", ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), true, false) // FIXME: handle "no search results" case, do not show empty `dmenu`
		if err != nil {
			l.Warnw("[search]", "no repository provided")
			ui.NotifyNormal("[search repos]", "no repository selected")
			return err
		}
	} else {
		l.Debugw("[search]", "error", "no matching repos found")
		ui.NotifyNormal("[search repos]", "no matching repos found")
		return errors.New("no matching repos found")
	}
	if ctx.Bool("copy-local") {
		return xserver.WriteClipboard(&path, false)
	} else if !ctx.Bool("shell") {
		elispCmd := fmt.Sprintf("(open-project \"%s\")", path)
		l.Debugw("[search]", "elispCmd", elispCmd)
		return emacs.SendToServer(elispCmd)
	} else {
		ui.NotifyNormal("[search repos]", fmt.Sprintf("opening terminal at %s", path))
		return shell.OpenTerminal(path, shell.TermTraitsFromContext(ctx))
	}
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
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "path",
					Usage:    "Use predefined path",
					Required: false,
				},
				&cli.StringFlag{
					Name:     "key",
					Usage:    "Search bookmarks by preselected key",
					Required: false,
				},
				&cli.BoolFlag{
					Name:     "shell",
					Usage:    "spawn shell at selected path",
					Required: false,
				},
				&cli.BoolFlag{
					Name:     "copy-local",
					Usage:    "Copy project's local path to clipboard",
					Required: false,
				},
			},
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
				&cli.BoolFlag{
					Name:     "copy-local",
					Usage:    "Copy project's local path to clipboard",
					Required: false,
				},
			},
		},
	}
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     tmux.SessionFlagName,
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
		&cli.StringFlag{
			Name:     ui.SelectorToolFlagName,
			Aliases:  []string{"T"},
			EnvVars:  []string{impl.EnvPrefix + "_SELECTOR_TOOL"},
			Value:    ui.SelectorToolDefault,
			Usage:    "Selector tool to use, e.g. dmenu, rofi, etc.",
			Required: false,
		},
		&cli.StringFlag{
			Name:     shell.TerminalCommandFlagName,
			EnvVars:  []string{impl.EnvPrefix + "_TERMINAL_CMD"},
			Usage:    "Terminal command to use",
			Required: false,
		},
		&cli.StringFlag{
			Name:     shell.TerminalBackendFlagName,
			EnvVars:  []string{impl.EnvPrefix + "_TERMINAL_BACKEND"},
			Value:    shell.TerminalBackendDefault,
			Usage:    "Terminal backend to use",
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
