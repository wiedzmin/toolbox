package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/env"
	"github.com/wiedzmin/toolbox/impl/json"
	"github.com/wiedzmin/toolbox/impl/shell"
	"github.com/wiedzmin/toolbox/impl/ui"
	"github.com/wiedzmin/toolbox/impl/vpn"
)

func perform(ctx *cli.Context) error {
	searchenginesData, client, err := env.GetRedisValue("nav/searchengines", nil)
	if err != nil {
		return err
	}
	searchenginesMeta, err := json.GetMapByPath(searchenginesData, "")
	var keys []string
	for key, _ := range searchenginesMeta {
		keys = append(keys, key)
	}
	key, err := ui.GetSelectionRofi(keys, "search with")
	if err != nil {
		return err
	}
	if searchengineMeta, ok := searchenginesMeta[key]; !ok {
		fmt.Printf("failed to get searchengine metadata for '%s'", key)
	} else {
		if vpnName, ok := searchengineMeta.Path("vpn").Data().(string); ok {
			vpnMeta, err := vpn.GetMetadata(client)
			if err != nil {
				return err
			}
			if vpnStartMeta, ok := vpnMeta[vpnName]; !ok {
				ui.NotifyCritical("[VPN]", fmt.Sprintf("Cannot find '%s' service", vpnName))
				return err
			} else {
				vpn.StopRunning([]string{vpnName}, vpnMeta, true)
				vpn.StartService(vpnName, vpnStartMeta, true)
			}
		}
		if url, ok := searchengineMeta.Path("url").Data().(string); ok {
			browserCmd, ok := searchengineMeta.Path("browser").Data().(string)
			if !ok {
				browserCmd = ctx.String("browser")
				if ctx.Bool("use-fallback") {
					browserCmd = ctx.String("fallback-browser")
				}
			}
			var searchTerm string
			if ctx.Bool("prompt") {
				searchTerm, err = ui.GetSelectionDmenu([]string{}, fmt.Sprintf("%s | term", key), 1, ctx.String("selector-font"))
				if err != nil {
					return err
				}
			} else {
				result, err := shell.ShellCmd("xsel -o", nil, nil, true, false)
				if err != nil {
					return err
				}
				searchTerm = *result
			}
			if searchTerm != "" {
				_, err := shell.ShellCmd(fmt.Sprintf("%s '%s%s'", browserCmd, url,
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
	app := createCLI()
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
