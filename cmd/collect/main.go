package main

import (
	"os"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/fs"
	"github.com/wiedzmin/toolbox/impl/redis"
	"go.uber.org/zap"
)

var logger *zap.Logger

func perform(ctx *cli.Context) error {
	result := fs.NewFSCollection(ctx.String("root"), strings.Split(ctx.String("exts"), ","), nil, false).EmitRecursive(true)
	jsonData, err := jsoniter.Marshal(result)
	if err != nil {
		return err
	}
	r, err := redis.NewRedisLocal()
	if err != nil {
		return err
	}
	return r.SetValue(ctx.String("key"), string(jsonData))
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "Collect"
	app.Usage = "Collect files for given regexps recursively ans save them under given Redis key"
	app.Description = "Collect"
	app.Version = "0.0.1#master"

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "exts",
			Aliases:  []string{"e"},
			Usage:    "Comma-separated files extensions to take into account",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "root",
			Aliases:  []string{"r"},
			Usage:    "Files search root",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "key",
			Aliases:  []string{"k"},
			Usage:    "Redis key name to save collected files under",
			Required: true,
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
