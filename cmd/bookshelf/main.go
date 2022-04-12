package main

import (
	"fmt"
	"os"

	jsoniter "github.com/json-iterator/go"
	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/redis"
	"github.com/wiedzmin/toolbox/impl/shell"
	"github.com/wiedzmin/toolbox/impl/ui"
	"go.uber.org/zap"
)

var logger *zap.Logger

func perform(ctx *cli.Context) error {
	var result []string
	r, err := redis.NewRedisLocal()
	if err != nil {
		return err
	}
	ebooks, err := r.GetValuesFuzzy("content/*/ebooks")
	if err != nil {
		ui.NotifyCritical("[bookshelf]", "Failed to fetch ebooks data")
		os.Exit(1)
	}
	for _, data := range ebooks {
		var ebooks []string
		err := jsoniter.Unmarshal(data, &ebooks)
		if err != nil {
			return err
		}
		for _, book := range ebooks {
			result = append(result, book)
		}
	}

	book, err := ui.GetSelection(result, "open", true, true, ctx.String("selector-font"))
	if err != nil {
		ui.NotifyNormal("[bookshelf]", "no book selected")
		return err
	}
	fmt.Printf("book: %s\n", book)
	_, err = shell.ShellCmd(fmt.Sprintf("%s \"%s\"", ctx.String("reader-command"), book),
		nil, nil, nil, false, false)
	if err != nil {
		return err
	}
	return nil
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "Bookshelf"
	app.Usage = "Select and open ebooks"
	app.Description = "Bookshelf"
	app.Version = "0.0.1#master"

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "reader-command",
			Aliases:  []string{"c"},
			EnvVars:  []string{impl.EnvPrefix + "_EBOOKS_READER_COMMAND"},
			Usage:    "Reader application to use for books opening",
			Required: true,
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
