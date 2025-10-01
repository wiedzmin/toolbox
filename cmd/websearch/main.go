package main

// FIXME: issue error when not predefined engine is selected (e.g. especially when trying to print search terms on this step)

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/bookmarks"
	"github.com/wiedzmin/toolbox/impl/shell"
	"github.com/wiedzmin/toolbox/impl/ui"
	"github.com/wiedzmin/toolbox/impl/vpn"
	"github.com/wiedzmin/toolbox/impl/xserver"
	"github.com/wiedzmin/toolbox/impl/xserver/xkb"
	"go.uber.org/zap"
)

var logger *zap.Logger

func perform(ctx *cli.Context) error {
	l := logger.Sugar()
	searchengines, err := bookmarks.WebjumpsFromRedis("nav/searchengines")
	if err != nil {
		return err
	}
	xkb.EnsureEnglishKeyboardLayout()
	key, err := ui.GetSelection(searchengines.Keys(), "search with", ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), true, false)
	l.Debugw("[perform]", "key", key, "err", err)
	if err != nil {
		return err
	}
	if searchengine := searchengines.Get(key); searchengine == nil {
		l.Errorw("[perform]", "failed to get searchengine metadata for", key)
	} else {
		if searchengine.VPN != "" {
			services, err := vpn.ServicesFromRedis("net/vpn_meta")
			if err != nil {
				return err
			}
			service := services.Get(searchengine.VPN)
			if service != nil {
				services.StopRunning([]string{searchengine.VPN}, true)
				service.Start(true)
			} else {
				ui.NotifyCritical("[VPN]", fmt.Sprintf("Cannot find '%s' service", searchengine.VPN))
				return vpn.ServiceNotFound{Name: searchengine.VPN}
			}
			l.Debugw("[perform]", "searchengine.VPN", searchengine.VPN, "vpnMeta", service, "services", *services)
		}
		if searchengine.URL != "" {
			l.Debugw("[perform]", "url", searchengine.URL)
			browserCmd := searchengine.BrowseWith
			if ctx.Bool("use-fallback") {
				browserCmd = ctx.String("fallback-browser")
			}
			var searchTerm string
			if ctx.Bool("prompt") {
				searchTerm, err = ui.GetSelection([]string{}, fmt.Sprintf("%s | term", key), ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), true, true)
				l.Debugw("[perform]", "searchTerm", searchTerm, "err", err)
				if err != nil {
					return err
				}
			} else if ctx.String("term") != "" {
				searchTerm = ctx.String("term")
			} else {
				impl.EnsureBinary("xsel", *logger)
				result, err := xserver.ReadClipboard(true)
				l.Debugw("[perform]", "clipboard/searchTerm", *result, "err", err)
				if err != nil {
					return err
				}
				searchTerm = *result
			}
			if searchTerm != "" {
				searchTermPrepared := strings.ReplaceAll(searchTerm, " ", "+")
				searchTermPrepared = strings.ReplaceAll(searchTermPrepared, ",", "")
				searchTermPrepared = strings.ReplaceAll(searchTermPrepared, "'s", "")
				l.Debugw("[perform]", "browserCmd", browserCmd, "searchengine.URL", searchengine.URL)
				_, err := shell.ShellCmd(fmt.Sprintf("%s '%s%s'", browserCmd, searchengine.URL,
					searchTermPrepared), nil, nil, nil, false, false)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("no URL to open")
		}
	}

	return nil
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "Websearch"
	app.Usage = "Searches for selected data tokens on various web resources from predefined list"
	app.Description = "Websearch"
	app.Version = "0.0.1#master"

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "fallback-browser",
			Aliases:  []string{"B"},
			EnvVars:  []string{impl.EnvPrefix + "_FALLBACK_BROWSER"},
			Usage:    "Fallback browser for opening selected links",
			Required: true,
		},
		&cli.BoolFlag{
			Name:     "use-fallback",
			Usage:    "Use fallback browser",
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "prompt",
			Usage:    "Prompt for tokens to search",
			Required: false,
		},
		&cli.StringFlag{
			Name:    "term",
			Aliases: []string{"t"},
			Usage:   "Explicitly search `term` with selected search engine",
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
