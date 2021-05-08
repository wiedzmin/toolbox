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
)

func perform(ctx *cli.Context) error {
	webjumpsData, _, err := env.GetRedisValue("nav/webjumps")
	webjumpsMeta, err := json.GetMapByPath(webjumpsData, "")
	if err != nil {
		return err
	}
	var keys []string
	for key, _ := range webjumpsMeta {
		keys = append(keys, key)
	}

	key, err := ui.GetSelectionRofi(keys, "jump to")
	if err != nil {
		return err
	}
	if webjumpMeta, ok := webjumpsMeta[key]; !ok {
		fmt.Printf("failed to get webjump metadata for '%s'", key)
	} else {
		if vpnName, ok := webjumpMeta.Path("vpn").Data().(string); ok {
			_, err := shell.ShellCmd(fmt.Sprintf("vpnctl --start %s", vpnName), nil, nil, false)
			if err != nil {
				return err
			}
		}
		if url, ok := webjumpMeta.Path("url").Data().(string); ok {
			copyURL := ctx.Bool("copy")
			if copyURL {
				_, err := shell.ShellCmd("xsel -ib", &url, false)
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
				_, err := shell.ShellCmd(fmt.Sprintf("%s %s", browserCmd, url), nil, false)
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
	app.Usage = "Opens various web resource from predefined list"
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
	app := createCLI()
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
