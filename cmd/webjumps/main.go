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
	"github.com/wiedzmin/toolbox/impl/xserver"
	"github.com/wiedzmin/toolbox/impl/xserver/xkb"
	"go.uber.org/zap"
)

var logger *zap.Logger

func perform(ctx *cli.Context) error {
	l := logger.Sugar()
	webjumps, err := bookmarks.WebjumpsFromRedis("nav/webjumps")
	if err != nil {
		return err
	}
	var keys []string
	if ctx.Bool("filter-workspace") || ctx.String("workspace") != "" {
		workspaceTag := ctx.String("workspace")
		if workspaceTag == "" {
			title, err := xserver.CurrentWorkspaceTitle()
			if err != nil {
				return err
			}
			titleFields := strings.Fields(title)
			workspaceTag = titleFields[len(titleFields)-1]
		}
		keys = webjumps.KeysByTag(workspaceTag)
	}
	if len(keys) == 0 {
		keys = webjumps.Keys()
	}

	xkb.EnsureEnglishKeyboardLayout()
	key, err := ui.GetSelection(keys, "jump to", true, false, ctx.String("selector-font"))
	l.Debugw("[perform]", "key", key, "err", err)
	if err != nil {
		return err
	}
	if webjump := webjumps.Get(key); webjump == nil {
		l.Errorw("[main]", "failed to get webjump metadata for", key)
	} else {
		if webjump.VPN != "" {
			services, err := vpn.ServicesFromRedis("net/vpn_meta")
			if err != nil {
				return err
			}
			service := services.Get(webjump.VPN)
			if service != nil {
				services.StopRunning([]string{webjump.VPN}, true)
				service.Start(true)
			} else {
				ui.NotifyCritical("[VPN]", fmt.Sprintf("Cannot find '%s' service", webjump.VPN))
				return vpn.ServiceNotFound{webjump.VPN}
			}
			l.Debugw("[perform]", "webjump.VPN", webjump.VPN, "vpnMeta", service)
		}
		if webjump.URL != "" {
			l.Debugw("[perform]", "url", webjump.URL)
			copyURL := ctx.Bool("copy")
			if copyURL {
				_, err := shell.ShellCmd("xsel -ib", &webjump.URL, nil, nil, false, false)
				if err != nil {
					return err
				}
			} else {
				var browserCmd string
				if webjump.Browser != "" {
					browserCmd = webjump.Browser
				} else {
					browserCmd = ctx.String("browser")
					if ctx.Bool("use-fallback") {
						browserCmd = ctx.String("fallback-browser")
					}
				}
				l.Debugw("[perform]", "browserCmd", browserCmd)
				_, err := shell.ShellCmd(fmt.Sprintf("%s %s", browserCmd, webjump.URL), nil, nil, nil, false, false)
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
	app.Name = "Webjumps"
	app.Usage = "Opens various web resources from predefined list"
	app.Description = "Webjumps"
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
			Name:     "copy",
			Usage:    "Copy url to clipboard",
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "filter-workspace",
			Usage:    "filter tagged jumps by active workspace",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "workspace",
			Aliases:  []string{"w"},
			Usage:    "Force workspace name for filtering tagged jumps",
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
	app := createCLI()
	err := app.Run(os.Args)
	if err != nil {
		l.Errorw("[main]", "err", err)
	}
}
