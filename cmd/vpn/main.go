package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/redis"
	"github.com/wiedzmin/toolbox/impl/ui"
	"github.com/wiedzmin/toolbox/impl/vpn"
	"go.uber.org/zap"
)

var logger *zap.Logger

func perform(ctx *cli.Context) error {
	var result []string
	r, err := redis.NewRedisLocal()
	if err != nil {
		return err
	}
	if ctx.Bool("status") {
		statuses, err := r.GetValuesMapFuzzy("vpn/*/is_up")
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
	services, err := vpn.ServicesFromRedis("net/vpn_meta")
	if err != nil {
		return err
	}
	if ctx.Bool("stop-all") {
		err = services.StopRunning(nil, true)
		if err != nil {
			return err
		}
		return nil
	}
	if ctx.String("start") != "" {
		service := services.Get(ctx.String("start"))
		if service == nil {
			return vpn.ServiceNotFound{ctx.String("start")}
		}
		return service.Start(true)
	}
	if ctx.String("stop") != "" {
		service := services.Get(ctx.String("stop"))
		if service == nil {
			return vpn.ServiceNotFound{ctx.String("stop")}
		}
		return service.Stop(true)
	}

	return nil
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "Vpn"
	app.Usage = "Manages and shows statuses of registered VPN services"
	app.Description = "Vpn"
	app.Version = "0.0.1#master"

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "start",
			Aliases:  []string{"f"},
			Usage:    "Start selected service",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "stop",
			Aliases:  []string{"S"},
			Usage:    "Stop selected service",
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "stop-all",
			Usage:    "Stop all currently running VPN services",
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "status",
			Usage:    "Show statuses of all registered VPN services",
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
