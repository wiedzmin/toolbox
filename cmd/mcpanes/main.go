package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/bookmarks"
	"github.com/wiedzmin/toolbox/impl/shell"
	"github.com/wiedzmin/toolbox/impl/ui"
	"go.uber.org/zap"
)

var logger *zap.Logger

func formatBookmark(title string, bm bookmarks.Bookmark) string {
	return fmt.Sprintf("%s - %s", title, bm.Path)
}

func perform(ctx *cli.Context) error {
	bms, err := bookmarks.BookmarksFromRedis("nav/bookmarks")
	if err != nil {
		return err
	}

	bmDirs, err := bms.FilteredPathsMap(false, true, true)
	if err != nil {
		return err
	}

	bookmarksForSelection := bookmarks.CustomKeyedMap(bmDirs, formatBookmark)

	selectionLeft, err := ui.GetSelection(bookmarks.GetBookmarksMapKeys(bookmarksForSelection), "left", ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), true, false)
	if err != nil {
		return err
	}
	selectionRight, err := ui.GetSelection(bookmarks.GetBookmarksMapKeys(bookmarksForSelection), "right", ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), true, false)
	if err != nil {
		return err
	}

	return shell.RunInTerminal(
		fmt.Sprintf("mc %s %s",
			strings.TrimSpace(bookmarksForSelection[selectionLeft].Path),
			strings.TrimSpace(bookmarksForSelection[selectionRight].Path)),
		"mcpanes", shell.TermTraitsFromContext(ctx))
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "Mcpanes"
	app.Usage = "Open filesystem bookmarks at Midnight Commander panels"
	app.Description = "Mcpanes"
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
		&cli.StringFlag{
			Name:     shell.TerminalCommandFlagName,
			EnvVars:  []string{impl.EnvPrefix + "_TERMINAL_CMD"},
			Usage:    "Terminal command to use",
			Required: false,
		},
		&cli.StringFlag{
			Name:     shell.TerminalBackendFlagName,
			Aliases:  []string{"t"},
			EnvVars:  []string{impl.EnvPrefix + "_TERMINAL_BACKEND"},
			Value:    shell.TerminalBackendDefault,
			Usage:    "Terminal backend to use",
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
