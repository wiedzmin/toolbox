package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/fs"
	"github.com/wiedzmin/toolbox/impl/shell"
	"github.com/wiedzmin/toolbox/impl/ui"
	"github.com/wiedzmin/toolbox/impl/xserver"
	"github.com/wiedzmin/toolbox/impl/xserver/xkb"
	"go.uber.org/zap"
)

var logger *zap.Logger

func fingerprint(ctx *cli.Context) error {
	l := logger.Sugar()
	fp, heads, err := xserver.HeadsFingerprint()
	if err != nil {
		return err
	}
	l.Debugw("[fingerprint]", "fingerprint", fp)
	xkb.EnsureEnglishKeyboardLayout()
	head, err := ui.GetSelection(heads, "head", ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), true, false)
	if err != nil {
		return err
	}
	if edid, ok := fp[head]; ok {
		ui.NotifyNormal("[randrutil]", fmt.Sprintf("copying EDID for '%s' to clipboard", head))
		return xserver.WriteClipboard(&edid, false)
	} else {
		ui.NotifyCritical("[randrutil]", fmt.Sprintf("Strangely, no EDID found for '%s'", head))
	}
	return nil
}

func appTraits(ctx *cli.Context) error {
	x, err := xserver.NewX()
	if err != nil {
		return err
	}
	windows, err := x.ListWindows()
	if err != nil {
		return err
	}

	traitsMap := make(map[string]xserver.WindowTraits)

	var titles []string
	for _, win := range windows {
		traits, err := x.GetWindowTraits(&win)
		if err != nil {
			return err
		}
		traitsMap[traits.Title] = *traits
		titles = append(titles, traits.Title)
	}

	title, err := ui.GetSelection(titles, "window", ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), true, false)
	if err != nil {
		return err
	}
	if traits, ok := traitsMap[title]; ok {
		traitName, err := ui.GetSelection(traits.ListNames(), ">", ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), true, false)
		if err != nil {
			return err
		}
		if trait, ok := traits.AsMap()[traitName]; ok {
			ui.NotifyNormal("[randrutil]", fmt.Sprintf("copying trait '%s' for '%s' to clipboard", traitName, impl.ShorterString(title, 20)))
			return xserver.WriteClipboard(&trait, false)
		} else {
			ui.NotifyCritical("[randrutil]", fmt.Sprintf("Strangely, no '%s' trait found for '%s'", traitName, title))
		}
	} else {
		ui.NotifyCritical("[randrutil]", fmt.Sprintf("Found no window traits for '%s'", title))
	}
	return nil
}

func traits(ctx *cli.Context) error {
	impl.EnsureBinary("xsel", *logger)
	switch {
	case ctx.Bool("fingerprint"):
		return fingerprint(ctx)
	case ctx.Bool("apps"):
		return appTraits(ctx)
	}
	return nil
}

func activate(ctx *cli.Context) error {
	impl.EnsureBinary("autorandr", *logger)
	profilesPath := fs.AtDotConfig("autorandr")

	profile, err := ui.GetSelection(
		fs.NewFSCollection(profilesPath, nil, []string{"\\.d$"}, true).Emit(false), "profile", ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), true, false)
	if err != nil {
		return err
	}

	_, err = shell.ShellCmd(fmt.Sprintf("autorandr --load %s", profile), nil, nil, nil, false, false)
	if err != nil {
		ui.NotifyCritical("[randrutil]", fmt.Sprintf("Failed to activate '%s' profile\n\nCause: %#v", profile, err))
		return err
	}
	ui.NotifyNormal("[randrutil]", fmt.Sprintf("Activated '%s' profile", profile))
	return nil
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "Randrutil"
	app.Usage = "Manage XRandR-related activities"
	app.Description = "Randrutil"
	app.Version = "0.0.1#master"

	app.Commands = cli.Commands{
		{
			Name:   "traits",
			Usage:  "Show traits",
			Action: traits,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:     "fingerprint",
					Usage:    "Show RandR fingerprint",
					Required: false,
				},
				&cli.BoolFlag{
					Name:     "apps",
					Usage:    "Show X traits of running applications windows",
					Required: false,
				},
			},
		},
		{
			Name:   "activate",
			Usage:  "activate Autorandr profile",
			Action: activate,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "profiles-root",
					Usage:    "Path where profiles are stored",
					Required: true,
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
