package main

import (
	"fmt"
	"os"
	"regexp"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/fs"
	"github.com/wiedzmin/toolbox/impl/ui"
	"go.uber.org/zap"
)

var (
	logger      *zap.Logger
	dateRegexps = []string{
		"screenshot-(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})_[0-9]{2}:[0-9]{2}:[0-9]{2}",
		"screenshot-(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})_[0-9]{2}-[0-9]{2}-[0-9]{2}",
		"(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})_[0-9]{2}:[0-9]{2}:[0-9]{2}_[0-9]+x[0-9]+_scrot",
		"(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})-[0-9]{6}_[0-9]+x[0-9]+_scrot",
		"(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})_[0-9]{2}-[0-9]{2}",
		"screenshot-(?P<day>[0-9]{2})-(?P<month>[0-9]{2})-(?P<year>[0-9]{4})-[0-9]{2}:[0-9]{2}:[0-9]{2}",
		"screenshot-[0-9]{2}:[0-9]{2}:[0-9]{2} (?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})",
		"screenshot-[0-9]{2}:[0-9]{2}:[0-9]{2}_(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})",
	}
)

func perform(ctx *cli.Context) error {
	l := logger.Sugar()
	files, err := fs.CollectFiles(ctx.String("root"), false, false, nil, nil)
	if err != nil {
		return err
	}
	filesCount := len(files)
	if filesCount == 0 {
		ui.NotifyCritical("[screenshots]", "No screenshots found")
		os.Exit(0)
	} else {
		var regexpsC []regexp.Regexp
		for _, re := range dateRegexps { // FIXME: could we precompile regexps earlier?
			rc := regexp.MustCompile(re)
			regexpsC = append(regexpsC, *rc)
		}
		for _, f := range files {
			failCount := 0
			var destDir, fallbackDir, srcPath, destPath string
			for _, rc := range regexpsC {
				if rc.MatchString(f) {
					l.Debugw("[perform]", "matched", f, "rc", rc)
					matches := rc.FindStringSubmatch(f)
					yearIndex := rc.SubexpIndex("year")
					monthIndex := rc.SubexpIndex("month")
					dayIndex := rc.SubexpIndex("day")
					destDir = fmt.Sprintf("%s/%s/%s/%s", ctx.String("root"), matches[yearIndex], matches[monthIndex], matches[dayIndex])
					srcPath = fmt.Sprintf("%s/%s", ctx.String("root"), f)
					destPath = fmt.Sprintf("%s/%s", destDir, f)
					l.Debugw("[perform]", "destDir", destDir, "srcPath", srcPath, "destPath", destPath)
					err := os.MkdirAll(destDir, 0777)
					if err != nil && !os.IsExist(err) {
						ui.NotifyCritical("[screenshots]", fmt.Sprintf("failed to create path %s\n\nCause: %#v", destDir, err))
					}
					err = os.Rename(srcPath, destPath)
					if err != nil {
						ui.NotifyCritical("[screenshots]", fmt.Sprintf("%s --> %s FAILED\n\nCause: %#v", f, destDir, err))
					} else {
						ui.NotifyNormal("[screenshots]", fmt.Sprintf("%s --> %s", f, destDir))
					}
					break
				} else {
					failCount++
				}
			}
			if failCount == len(regexpsC) {
				fallbackDir = fmt.Sprintf("%s/%s", ctx.String("root"), ctx.String("non-matched"))
				err := os.MkdirAll(fallbackDir, 0777)
				if err != nil && !os.IsExist(err) {
					ui.NotifyCritical("[screenshots]", fmt.Sprintf("failed to create path %s\n\nCause: %#v", fallbackDir, err))
				}
				srcPath = fmt.Sprintf("%s/%s", ctx.String("root"), f)
				destPath = fmt.Sprintf("%s/%s", fallbackDir, f)
				ui.NotifyCritical("[screenshots]", fmt.Sprintf("%s did not matched any regexps, custom name encountered\n\nMoving under '%s' subdirectory", f, fallbackDir))
				l.Debugw("[perform]", "fallbackDir", fallbackDir, "srcPath", srcPath, "destPath", destPath)
				err = os.Rename(srcPath, destPath)
				if err != nil {
					ui.NotifyCritical("[screenshots]", fmt.Sprintf("%s --> %s FAILED\n\nCause: %#v", f, fallbackDir, err))
				} else {
					ui.NotifyNormal("[screenshots]", fmt.Sprintf("%s --> %s", f, fallbackDir))
				}
			}
		}
	}

	return nil
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "Screenshots"
	app.Usage = "Screenshots ordering"
	app.Description = "Screenshots"
	app.Version = "0.0.1#master"

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "root",
			Usage:    "Base directory to perform sorting under",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "non-matched",
			Value:    "named",
			Usage:    "Directory under base to place named/non-matched screenshots to",
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
