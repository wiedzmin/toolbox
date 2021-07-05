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
	"github.com/wiedzmin/toolbox/impl/vpn"
	"go.uber.org/zap"
)

var logger *zap.Logger

func perform(ctx *cli.Context) error {
	l := logger.Sugar()
	searchengines, err := bookmarks.WebjumpsFromRedis("nav/searchengines")
	if err != nil {
		return err
	}
	key, err := ui.GetSelectionRofi(searchengines.Keys(), "search with")
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
				return vpn.ServiceNotFound{searchengine.VPN}
			}
			l.Debugw("[perform]", "searchengine.VPN", searchengine.VPN, "vpnMeta", service, "services", *services)
		}
		if searchengine.URL != "" {
			l.Debugw("[perform]", "url", searchengine.URL)
			var browserCmd string
			if searchengine.Browser != "" {
				browserCmd = searchengine.Browser
			} else {
				browserCmd = ctx.String("browser")
				if ctx.Bool("use-fallback") {
					browserCmd = ctx.String("fallback-browser")
				}
			}
			var searchTerm string
			if ctx.Bool("prompt") {
				searchTerm, err = ui.GetSelectionDmenu([]string{}, fmt.Sprintf("%s | term", key), 1, ctx.String("selector-font"))
				l.Debugw("[perform]", "searchTerm", searchTerm, "err", err)
				if err != nil {
					return err
				}
			} else {
				result, err := shell.ShellCmd("xsel -o", nil, nil, true, false)
				l.Debugw("[perform]", "clipboard/searchTerm", *result, "err", err)
				if err != nil {
					return err
				}
				searchTerm = *result
			}
			if searchTerm != "" {
				l.Debugw("[perform]", "browserCmd", browserCmd, "searchengine.URL", searchengine.URL)
				_, err := shell.ShellCmd(fmt.Sprintf("%s '%s%s'", browserCmd, searchengine.URL,
					strings.ReplaceAll(searchTerm, " ", "+")), nil, nil, false, false)
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
			Name:     "browser",
			Aliases:  []string{"b"},
			EnvVars:  []string{impl.EnvPrefix + "_DEFAULT_BROWSER"},
			Usage:    "Default browser for opening selected links",
			Required: true,
		},
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
	logger = impl.NewLogger()
	defer logger.Sync()
	l := logger.Sugar()
	impl.EnsureBinary("xsel", *logger)
	app := createCLI()
	err := app.Run(os.Args)
	if err != nil {
		l.Errorw("[main]", "err", err)
	}
}
