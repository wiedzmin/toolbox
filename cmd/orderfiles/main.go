package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/file"
	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/fs"
	"github.com/wiedzmin/toolbox/impl/ui"
	"go.uber.org/zap"
)

var (
	logger                 *zap.Logger
	conf                   = koanf.New(".")
	unmatchedSubdirDefault = "named"
)

func perform(ctx *cli.Context) error {
	l := logger.Sugar()
	if err := conf.Load(file.Provider(ctx.String("config")), json.Parser()); err != nil {
		return err
	}

	notificationTitle := conf.String("title")
	if notificationTitle == "" {
		notificationTitle = "orderfiles"
	}
	notificationTitle = fmt.Sprintf("[%s]", notificationTitle)

	if conf.String("from") == "" {
		ui.NotifyCritical(notificationTitle, "Source path is not set")
		os.Exit(0)
	}
	l.Debugw("[perform]", "from", conf.String("from"))
	if len(conf.StringMap("rules")) == 0 {
		ui.NotifyCritical(notificationTitle, "No rules provided")
		os.Exit(0)
	}
	l.Debugw("[perform]", "rules", conf.StringMap("rules"))

	files := fs.NewFSCollection(conf.String("from"), nil, nil, false).Emit(false)
	filesCount := len(files)
	if filesCount == 0 {
		ui.NotifyCritical(notificationTitle, "No source files found")
		os.Exit(0)
	} else {
		var regexpsC []regexp.Regexp
		for re := range conf.StringMap("rules") {
			regexpsC = append(regexpsC, *regexp.MustCompile(re))
		}
		fromTrimmed := strings.TrimSuffix(conf.String("from"), "/")
		toTrimmed := strings.TrimSuffix(conf.String("to"), "/")
		if toTrimmed == "" {
			toTrimmed = fromTrimmed
		}
		l.Debugw("[perform]", "fromTrimmed", fromTrimmed, "toTrimmed", toTrimmed)
		for _, f := range files {
			failCount := 0
			errored := false
			var destDir, fallbackDir, srcPath, destPath string
			for _, rc := range regexpsC {
				if rc.MatchString(f) {
					l.Debugw("[perform]", "matched", f, "rc", rc.String())
					result, err := impl.RegexpToTemplate(f, rc, conf.StringMap("rules"))
					if err != nil {
						errored = true
						l.Debugw("[perform]", "error getting result path", err)
						continue
					}
					l.Debugw("[perform]", "result path", result)

					destDir = fmt.Sprintf("%s/%s", toTrimmed, *result)
					l.Debugw("[perform]", "destDir", destDir)
					err = os.MkdirAll(destDir, 0777)
					if err != nil && !os.IsExist(err) {
						errorText := fmt.Sprintf("failed to create path %s\n\nCause: %#v", destDir, err)
						ui.NotifyCritical(notificationTitle, errorText)
						l.Debugw("[perform]", "error", errorText)
						continue
					}

					srcPath = fmt.Sprintf("%s/%s", fromTrimmed, f)
					destPath = fmt.Sprintf("%s/%s", destDir, f)
					l.Debugw("[perform]", "srcPath", srcPath, "destPath", destPath)

					err = os.Rename(srcPath, destPath)
					if err != nil {
						ui.NotifyCritical(notificationTitle, fmt.Sprintf("%s --> %s FAILED\n\nCause: %#v", f, destDir, err))
					} else {
						ui.NotifyNormal(notificationTitle, fmt.Sprintf("%s --> %s", f, destDir))
					}
					break
				} else {
					failCount++
				}
			}
			if errored {
				break
			}
			if failCount == len(regexpsC) {
				l.Debugw("[perform]", "unmatched.skip", conf.Bool("unmatched.skip"))
				l.Debugw("[perform]", "unmatched.subdir", conf.Bool("unmatched.subdir"))
				if !conf.Bool("unmatched.skip") {
					unmatchedSubdir := conf.String("unmatched.subdir")
					if unmatchedSubdir == "" {
						unmatchedSubdir = unmatchedSubdirDefault
					}
					fallbackDir = fmt.Sprintf("%s/%s", fromTrimmed, unmatchedSubdir)
					err := os.MkdirAll(fallbackDir, 0777)
					if err != nil && !os.IsExist(err) {
						ui.NotifyCritical(notificationTitle, fmt.Sprintf("failed to create path %s\n\nCause: %#v", fallbackDir, err))
					}
					srcPath = fmt.Sprintf("%s/%s", fromTrimmed, f)
					destPath = fmt.Sprintf("%s/%s", fallbackDir, f)
					ui.NotifyCritical(notificationTitle, fmt.Sprintf("%s did not matched any regexps, custom name encountered\n\nMoving under '%s' subdirectory", f, fallbackDir))
					l.Debugw("[perform]", "fallbackDir", fallbackDir, "srcPath", srcPath, "destPath", destPath)
					err = os.Rename(srcPath, destPath)
					if err != nil {
						ui.NotifyCritical(notificationTitle, fmt.Sprintf("%s --> %s FAILED\n\nCause: %#v", f, fallbackDir, err))
					} else {
						ui.NotifyNormal(notificationTitle, fmt.Sprintf("%s --> %s", f, fallbackDir))
					}
				} else if conf.Bool("unmatched.skip") {
					ui.NotifyNormal(notificationTitle, fmt.Sprintf("skipping '%s' due to `skip-unmatched` flag", f))
					l.Debugw("[perform]", "note", fmt.Sprintf("skipping '%s' due to `skip-unmatched` flag", f))
				}
			}
		}
	}

	return nil
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "Orderfiles"
	app.Usage = "Arbitrary files ordering"
	app.Description = "Orderfiles"
	app.Version = "0.0.1#master"

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "config",
			Aliases:  []string{"c"},
			Usage:    "Configuration file to use",
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
