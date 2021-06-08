package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/env"
	"github.com/wiedzmin/toolbox/impl/shell"
	"github.com/wiedzmin/toolbox/impl/ui"
)

func perform(ctx *cli.Context) error {
	var result []string
	ebooks, _, err := env.GetRedisValuesFuzzy("content/*/ebooks", nil)
	if err != nil {
		ui.NotifyCritical("[bookshelf]", "Failed to fetch ebooks data")
		os.Exit(1)
	}
	for _, data := range ebooks {
		var ebooks []string
		err := json.Unmarshal(data, &ebooks)
		if err != nil {
			return err
		}
		for _, book := range ebooks {
			result = append(result, book)
		}
	}

	book, err := ui.GetSelectionRofi(result, "open")
	if err != nil {
		ui.NotifyNormal("[bookshelf]", "no book selected")
		return err
	}
	fmt.Printf("book: %s\n", book)
	_, err = shell.ShellCmd(fmt.Sprintf("%s \"%s\"", ctx.String("reader-command"), book),
		nil, nil, false, false)
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
	app := createCLI()
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
