package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/emacs"
	"github.com/wiedzmin/toolbox/impl/env"
	"github.com/wiedzmin/toolbox/impl/json"
	"github.com/wiedzmin/toolbox/impl/shell"
	"github.com/wiedzmin/toolbox/impl/systemd"
	"github.com/wiedzmin/toolbox/impl/ui"
	"go.uber.org/zap"
)

var logger *zap.Logger

func open(ctx *cli.Context) error {
	bookmarksData, _, err := env.GetRedisValue("nav/bookmarks", nil)
	if err != nil {
		return err
	}
	bookmarksMeta, err := json.GetMapByPath(bookmarksData, "")
	var keys []string
	for key, _ := range bookmarksMeta {
		keys = append(keys, key)
	}
	key, err := ui.GetSelectionRofi(keys, "open")
	if err != nil {
		return err
	}
	if bookmarkMeta, ok := bookmarksMeta[key]; !ok {
		fmt.Printf("failed to get bookmark metadata for '%s'", key)
	} else {
		if path, ok := bookmarkMeta.Path("path").Data().(string); !ok {
			fmt.Printf("missing bookmark path '%s'", key)
		} else {
			if _, ok := bookmarkMeta.Path("shell").Data().(string); ok {
				fmt.Printf("open shell: not implemented")
			}
			emacsService := systemd.Unit{Name: "emacs.service", User: true}
			isActive, err := emacsService.IsActive()
			if err != nil {
				return err
			}
			if !isActive {
				ui.NotifyCritical("[bookmarks]", "Emacs service not running")
				os.Exit(1)
			}
			fi, err := os.Stat(path)
			if err != nil {
				return err
			}
			elispCmd := fmt.Sprintf("(dired \"%s\")", path)
			if !fi.IsDir() {
				elispCmd = fmt.Sprintf("(find-file \"%s\")", path)
			}
			return emacs.SendToServer(elispCmd)
		}
	}
	return nil
}

func search(ctx *cli.Context) error {
	searchTerm, err := ui.GetSelectionDmenu([]string{}, "token", 1, ctx.String("selector-font"))
	if err != nil {
		ui.NotifyCritical("[search repos]", "no keyword provided")
		return err
	}
	matchingRepos, err := shell.ShellCmd(fmt.Sprintf("fd -t d -d %d %s %s",
		ctx.Int("depth"), searchTerm, ctx.String("root")), nil, nil, true, false)
	if err != nil {
		return err
	}
	matchingReposSlice := strings.Split(*matchingRepos, "\n")
	path, err := ui.GetSelectionRofi(matchingReposSlice, "explore")
	if err != nil {
		ui.NotifyNormal("[search repos]", "no repository selected")
		return err
	}

	emacsService := systemd.Unit{Name: "emacs.service", User: true}
	isActive, err := emacsService.IsActive()
	if err != nil {
		return err
	}
	if !isActive {
		ui.NotifyCritical("[search repos]", "Emacs service not running")
		os.Exit(1)
	}
	elispCmd := fmt.Sprintf("(dired \"%s\")", path)
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
			Name:     "selector-font",
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
	impl.EnsureBinary("fd", *logger)
	app := createCLI()
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
