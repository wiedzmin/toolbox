package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl/env"
	"github.com/wiedzmin/toolbox/impl/ui"
	"github.com/wiedzmin/toolbox/impl/vpn"
)

func perform(ctx *cli.Context) error {
	var result []string
	if ctx.Bool("stop-all") {
		vpnMeta, err := vpn.GetMetadata(nil)
		if err != nil {
			return err
		}
		err = vpn.StopRunning(nil, vpnMeta, true)
		if err != nil {
			return err
		}
		return nil
	}
	statuses, _, err := env.GetRedisValuesFuzzy("vpn/*/is_up", nil)
	if err == nil {
		for key, value := range statuses {
			result = append(result, fmt.Sprintf("%s: %s", strings.Split(key, "/")[1], string(value)))
		}
		ui.NotifyNormal("[VPN] statuses", strings.Join(result, "\n"))
	} else {
		ui.NotifyCritical("[VPN]", "Failed to get vpn statuses")
	}

	return nil
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "Vpnstatus"
	app.Usage = "Displays statuses of and stops registered VPN services"
	app.Description = "Vpnstatus"
	app.Version = "0.0.1#master"

	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:     "stop-all",
			Usage:    "Stop all currently running VPN services",
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
