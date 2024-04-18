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

func docs(ctx *cli.Context) error {
	l := logger.Sugar()
	var result []string
	r, err := redis.NewRedisLocal()
	if err != nil {
		return err
	}
	docs, err := r.GetValuesMapFuzzy("paperworks/*/docs")
	if err != nil {
		ui.NotifyCritical("[insight]", "Failed to fetch docs data")
		os.Exit(1)
	}
	l.Debugw("[insight]", "docs", docs)
	for key, data := range docs {
		l.Debugw("[insight]", "key", key, "data", data)
		var docs []string
		err := jsoniter.Unmarshal(data, &docs)
		if err != nil {
			return err
		}
		for _, doc := range docs {
			result = append(result, doc)
		}
	}

	doc, err := ui.GetSelection(result, "open", ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), true, true)
	if err != nil {
		ui.NotifyNormal("[insight]", "no document selected")
		return err
	}
	fmt.Printf("doc: %s\n", doc)
	_, err = shell.ShellCmd(fmt.Sprintf("%s \"%s\"", ctx.String("office-command"), doc),
		nil, nil, nil, false, false)
	if err != nil {
		return err
	}
	return nil
}

func ebooks(ctx *cli.Context) error {
	l := logger.Sugar()
	var result []string
	r, err := redis.NewRedisLocal()
	if err != nil {
		return err
	}
	ebooks, err := r.GetValuesMapFuzzy("content/*/ebooks")
	if err != nil {
		ui.NotifyCritical("[insight]", "Failed to fetch ebooks data")
		os.Exit(1)
	}
	l.Debugw("[insight]", "ebooks", ebooks)
	for key, data := range ebooks {
		l.Debugw("[insight]", "key", key, "data", data)
		var ebooks []string
		err := jsoniter.Unmarshal(data, &ebooks)
		if err != nil {
			return err
		}
		for _, book := range ebooks {
			result = append(result, book)
		}
	}

	book, err := ui.GetSelection(result, "open", ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), true, true)
	if err != nil {
		ui.NotifyNormal("[insight]", "no book selected")
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
	app.Name = "Insight"
	app.Usage = "Select and open documents/ebooks/<whatever the same>"
	app.Description = "Insight"
	app.Version = "0.0.1#master"

	app.Commands = cli.Commands{
		{
			Name:   "docs",
			Usage:  "Select and open documents, i.e. WYSIWYG text, spreadsheets, and other office-related stuff",
			Action: docs,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "office-command",
					Aliases:  []string{"c"},
					EnvVars:  []string{impl.EnvPrefix + "_DOCS_VIEWER_COMMAND"},
					Usage:    "Viewing/editing application to use for documents opening",
					Required: true,
				},
			},
		},
		{
			Name:   "ebooks",
			Usage:  "Select and open ebooks",
			Action: ebooks,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "reader-command",
					Aliases:  []string{"c"},
					EnvVars:  []string{impl.EnvPrefix + "_EBOOKS_READER_COMMAND"},
					Usage:    "Reader application to use for books opening",
					Required: true,
				},
			},
		},
	}
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     impl.SelectorFontFlagName,
			Aliases:  []string{"f"},
			EnvVars:  []string{impl.EnvPrefix + "_SELECTOR_FONT"},
			Usage:    "Font to use for selector application, e.g. dmenu, rofi, etc.",
			Required: false,
		},
		&cli.StringFlag{
			Name:     ui.SelectorToolFlagName,
			Aliases:  []string{"T"},
			EnvVars:  []string{impl.EnvPrefix + "_SELECTOR_TOOL"},
			Value:    ui.SelectorTool,
			Usage:    "Selector tool to use, e.g. dmenu, rofi, etc.",
			Required: false,
		},
	}
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
