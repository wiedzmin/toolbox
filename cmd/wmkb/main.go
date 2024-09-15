package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/redis"
	"github.com/wiedzmin/toolbox/impl/ui"
	"github.com/wiedzmin/toolbox/impl/xserver/wm"
	"go.uber.org/zap"
)

var (
	logger *zap.Logger
	r      *redis.Client
)

func modes(ctx *cli.Context) error {
	modebindings, err := wm.ModebindingsFromRedis("wm/modebindings")
	if err != nil {
		return err
	}

	prompt := "Modes bindings"
	if ctx.Bool("fuzzy") {
		_, err := ui.GetSelection(modebindings.Fuzzy(), prompt, ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), true, false)
		if err != nil {
			return err
		}
	} else {
		ui.ShowTextDialog(modebindings.AsText(), prompt)
	}

	return nil
}

func keys(ctx *cli.Context) error {
	keybindings, err := wm.KeybindingsFromRedis("wm/keybindings")
	if err != nil {
		return err
	}
	modebindings, err := wm.ModebindingsFromRedis("wm/modebindings")
	if err != nil {
		return err
	}

	wm.LinkBindings(keybindings, modebindings)

	prompt := "Keybindings"
	if ctx.Bool("fuzzy") {
		kbFuzzy, err := keybindings.Fuzzy(wm.FormatFuzzyCommon)
		if err != nil {
			return err
		}
		selection, err := ui.GetSelection(kbFuzzy, prompt, ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), true, false)
		if err != nil {
			return err
		}
		parts, err := keybindings.GetPartsForSelection(selection)
		if err != nil {
			return err
		}
		formattedStr := fmt.Sprintf("command: %s\nkeys: %s\nmode: %s\nleave fullscreen: %s\nraw: %s\ndangling: %s\n",
			parts.Cmd,
			parts.Key,
			parts.Mode,
			parts.LeaveFullscreen,
			parts.Raw,
			parts.Dangling,
		)
		ui.NotifyNormal(prompt, formattedStr)
		if ctx.Bool("dump") {
			ui.ShowTextDialog(formattedStr, prompt)
		}
	} else {
		if ctx.Bool("tree") {
			text, err := keybindings.AsTextTree(wm.FormatTextIndented)
			if err != nil {
				return err
			}

			ui.ShowTextDialog(*text, prompt)
		} else {
			text, err := keybindings.AsText(wm.FormatTextFlat)
			if err != nil {
				return err
			}
			ui.ShowTextDialog(*text, prompt)
		}
	}

	return nil
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "wmkb"
	app.Usage = "Handy dumper of WM keybindings"
	app.Description = "wmkb"
	app.Version = "0.0.1#master"

	app.Commands = cli.Commands{
		{
			Name:   "modes",
			Usage:  "Whether to show modes keybindings",
			Action: modes,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:     "fuzzy",
					Usage:    "Whether to use fuzzy matching",
					Required: false,
				},
			},
		},
		{
			Name:   "keys",
			Usage:  "Whether to show keybindings",
			Action: keys,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:     "fuzzy",
					Usage:    "Whether to use fuzzy matching",
					Required: false,
				},
				&cli.BoolFlag{
					Name:     "dump",
					Usage:    "Whether to dump keybindind metadata to text dialog",
					Required: false,
				},
				&cli.BoolFlag{
					Name:     "tree",
					Usage:    "Whether to show tree-like, representation, conflicts with fuzzy matching",
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
			Value:    ui.SelectorTool,
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
