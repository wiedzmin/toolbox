package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl/fs"
	"github.com/wiedzmin/toolbox/impl/redis"
)

func perform(ctx *cli.Context) error {
	result, err := fs.CollectFilesRecursive(ctx.String("root"), strings.Split(ctx.String("exts"), ","))
	if err != nil {
		return err
	}
	jsonData, err := json.Marshal(result)
	if err != nil {
		return err
	}
	r, err := redis.NewRedisLocal()
	if err != nil {
		return err
	}
	err = r.SetValue(ctx.String("key"), string(jsonData))
	return err
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
	app := createCLI()
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
