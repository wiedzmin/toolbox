package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/fs"
	"github.com/wiedzmin/toolbox/impl/ui"
	"go.uber.org/zap"
)

var (
	logger                  *zap.Logger
	srcRegexpToDestTemplate = map[string]string{ // TODO: extract to external configuration
		"screenshot-(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})_[0-9]{2}:[0-9]{2}:[0-9]{2}":          "{{.year}}/{{.month}}/{{.day}}",
		"screenshot-(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})_[0-9]{2}-[0-9]{2}-[0-9]{2}":          "{{.year}}/{{.month}}/{{.day}}",
		"(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})_[0-9]{2}:[0-9]{2}:[0-9]{2}_[0-9]+x[0-9]+_scrot": "{{.year}}/{{.month}}/{{.day}}",
		"(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})-[0-9]{6}_[0-9]+x[0-9]+_scrot":                   "{{.year}}/{{.month}}/{{.day}}",
		"(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})_[0-9]{2}-[0-9]{2}":                              "{{.year}}/{{.month}}/{{.day}}",
		"screenshot-(?P<day>[0-9]{2})-(?P<month>[0-9]{2})-(?P<year>[0-9]{4})-[0-9]{2}:[0-9]{2}:[0-9]{2}":          "{{.year}}/{{.month}}/{{.day}}",
		"screenshot-[0-9]{2}:[0-9]{2}:[0-9]{2} (?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})":          "{{.year}}/{{.month}}/{{.day}}",
		"screenshot-[0-9]{2}:[0-9]{2}:[0-9]{2}_(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})":          "{{.year}}/{{.month}}/{{.day}}",
	}
)

func perform(ctx *cli.Context) error {
	l := logger.Sugar()
	files := fs.NewFSCollection(ctx.String("root"), nil, nil, false).Emit(false)
	filesCount := len(files)
	if filesCount == 0 {
		ui.NotifyCritical("[orderfiles]", "No source files found")
		os.Exit(0)
	} else {
		var regexpsC []regexp.Regexp
		for re := range srcRegexpToDestTemplate {
			regexpsC = append(regexpsC, *regexp.MustCompile(re))
		}
		rootTrimmed := strings.TrimSuffix(ctx.String("root"), "/")
		destRootTrimmed := strings.TrimSuffix(ctx.String("dest-root"), "/")
		if destRootTrimmed == "" {
			destRootTrimmed = rootTrimmed
		}
		l.Debugw("[perform]", "rootTrimmed", rootTrimmed, "destRootTrimmed", destRootTrimmed)
		for _, f := range files {
			failCount := 0
			errored := false
			var destDir, fallbackDir, srcPath, destPath string
			for _, rc := range regexpsC {
				if rc.MatchString(f) {
					l.Debugw("[perform]", "matched", f, "rc", rc)
					result, err := impl.RegexpToTemplate(f, rc, srcRegexpToDestTemplate)
					if err != nil {
						errored = true
						l.Debugw("[perform]", "error getting result path", err)
						continue
					}
					l.Debugw("[perform]", "result path", result)

					destDir = fmt.Sprintf("%s/%s", destRootTrimmed, *result)
					l.Debugw("[perform]", "destDir", destDir)
					err = os.MkdirAll(destDir, 0777)
					if err != nil && !os.IsExist(err) {
						errorText := fmt.Sprintf("failed to create path %s\n\nCause: %#v", destDir, err)
						ui.NotifyCritical("[orderfiles]", errorText)
						l.Debugw("[perform]", "error", errorText)
						continue
					}

					srcPath = fmt.Sprintf("%s/%s", rootTrimmed, f)
					destPath = fmt.Sprintf("%s/%s", destDir, f)
					l.Debugw("[perform]", "srcPath", srcPath, "destPath", destPath)

					err = os.Rename(srcPath, destPath)
					if err != nil {
						ui.NotifyCritical("[orderfiles]", fmt.Sprintf("%s --> %s FAILED\n\nCause: %#v", f, destDir, err))
					} else {
						ui.NotifyNormal("[orderfiles]", fmt.Sprintf("%s --> %s", f, destDir))
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
				if !ctx.Bool("skip-unmatched") {
					fallbackDir = fmt.Sprintf("%s/%s", rootTrimmed, ctx.String("unmatched"))
					err := os.MkdirAll(fallbackDir, 0777)
					if err != nil && !os.IsExist(err) {
						ui.NotifyCritical("[orderfiles]", fmt.Sprintf("failed to create path %s\n\nCause: %#v", fallbackDir, err))
					}
					srcPath = fmt.Sprintf("%s/%s", rootTrimmed, f)
					destPath = fmt.Sprintf("%s/%s", fallbackDir, f)
					ui.NotifyCritical("[orderfiles]", fmt.Sprintf("%s did not matched any regexps, custom name encountered\n\nMoving under '%s' subdirectory", f, fallbackDir))
					l.Debugw("[perform]", "fallbackDir", fallbackDir, "srcPath", srcPath, "destPath", destPath)
					err = os.Rename(srcPath, destPath)
					if err != nil {
						ui.NotifyCritical("[orderfiles]", fmt.Sprintf("%s --> %s FAILED\n\nCause: %#v", f, fallbackDir, err))
					} else {
						ui.NotifyNormal("[orderfiles]", fmt.Sprintf("%s --> %s", f, fallbackDir))
					}
				} else if ctx.Bool("skip-unmatched") {
					ui.NotifyNormal("[orderfiles]", fmt.Sprintf("skipping '%s' due to `skip-unmatched` flag", f))
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
			Name:     "root",
			Usage:    "Base directory to perform sorting under",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "dest-root",
			Usage:    "Base directory to move sorted files under",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "unmatched",
			Value:    "named",
			Usage:    "Directory under base to place custom-named/unmatched files to",
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "skip-unmatched",
			Usage:    "Whether to skip files that did not matched any regexps",
			Value:    false,
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
