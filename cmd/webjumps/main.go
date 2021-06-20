package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/env"
	"github.com/wiedzmin/toolbox/impl/json"
	"github.com/wiedzmin/toolbox/impl/shell"
	"github.com/wiedzmin/toolbox/impl/ui"
	"github.com/wiedzmin/toolbox/impl/vpn"
	"go.uber.org/zap"
)

var logger *zap.Logger

func perform(ctx *cli.Context) error {
	l := logger.Sugar()
	webjumpsData, client, err := env.GetRedisValue("nav/webjumps", nil)
	if err != nil {
		return err
	}
	webjumpsMeta, err := json.GetMapByPath(webjumpsData, "")
	var keys []string
	for key, _ := range webjumpsMeta {
		keys = append(keys, key)
	}
	l.Debugw("[perform]", "keys", keys)

	key, err := ui.GetSelectionRofi(keys, "jump to")
	l.Debugw("[perform]", "key", key, "err", err)
	if err != nil {
		return err
	}
	if webjumpMeta, ok := webjumpsMeta[key]; !ok {
		l.Errorw("[main]", "failed to get webjump metadata for", key)
	} else {
		if vpnName, ok := webjumpMeta.Path("vpn").Data().(string); ok {
			vpnsMeta, err := vpn.GetMetadata(client)
			if err != nil {
				return err
			}
			var vpnStartMeta map[string]string
			if vpnStartMeta, ok = vpnsMeta[vpnName]; !ok {
				ui.NotifyCritical("[VPN]", fmt.Sprintf("Cannot find '%s' service", vpnName))
				return err
			} else {
				vpn.StopRunning([]string{vpnName}, vpnsMeta, true)
				vpn.StartService(vpnName, vpnStartMeta, true)
			}
			l.Debugw("[perform]", "vpnName", vpnName, "vpnStartMeta", vpnStartMeta, "vpnsMeta", vpnsMeta)
		}
		if url, ok := webjumpMeta.Path("url").Data().(string); ok {
			l.Debugw("[perform]", "url", url)
			copyURL := ctx.Bool("copy")
			if copyURL {
				_, err := shell.ShellCmd("xsel -ib", &url, nil, false, false)
				if err != nil {
					return err
				}
			} else {
				browserCmd, ok := webjumpMeta.Path("browser").Data().(string)
				if !ok {
					browserCmd = ctx.String("browser")
					if ctx.Bool("use-fallback") {
						browserCmd = ctx.String("fallback-browser")
					}
				}
				l.Debugw("[perform]", "browserCmd", browserCmd)
				_, err := shell.ShellCmd(fmt.Sprintf("%s %s", browserCmd, url), nil, nil, false, false)
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
