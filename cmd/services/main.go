package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/redis"
	"github.com/wiedzmin/toolbox/impl/systemd"
	"github.com/wiedzmin/toolbox/impl/ui"
	"github.com/wiedzmin/toolbox/impl/xserver/xkb"
	"go.uber.org/zap"
)

const redisKeyName = "system/services"

var (
	OPERATIONS = []string{
		"stop",
		"stop/follow",
		"kill",
		"restart",
		"restart/follow",
		"show",
		"journal",
		"journal/follow",
		"status",
	}
	logger *zap.Logger
	r      *redis.Client
)

func ensureUnitsCache() error {
	l := logger.Sugar()
	if !r.KeyExists(redisKeyName) {
		l.Debugw("[ensureUnitsCache]", "units cache", "does not exist, populating")
		units, err := systemd.CollectUnits(true, true)
		l.Debugw("[ensureUnitsCache]", "found units", len(units))
		if err != nil {
			return err
		}
		for _, u := range units {
			err = r.AppendToList(redisKeyName, u.String())
			if err != nil {
				return err
			}
		}
	} else {
		l.Debugw("[ensureUnitsCache]", "units cache", "exists, moving on")
	}

	return nil
}

func perform(ctx *cli.Context) error {
	var err error
	l := logger.Sugar()
	if ctx.Bool("invalidate-cache") {
		err := systemd.DaemonReload()
		if err != nil {
			return err
		}
		err = r.DeleteValue(redisKeyName)
		if err != nil {
			return err
		}
		return nil
	}

	err = ensureUnitsCache()
	if err != nil {
		return err
	}

	units, err := r.GetList(redisKeyName, 0, -1)
	xkb.EnsureEnglishKeyboardLayout()
	unitStr, err := ui.GetSelection(ctx, units, "select", true, false)
	if err != nil {
		return err
	}
	xkb.EnsureEnglishKeyboardLayout()
	// FIXME: ensure sort order
	operation, err := ui.GetSelection(ctx, OPERATIONS, "perform", true, false)
	if err != nil {
		return err
	}
	unit := systemd.UnitFromString(unitStr)
	switch operation {
	case "stop":
		err = unit.Stop()
		if err != nil {
			l.Errorw("[perform]", "err", err)
			ui.NotifyCritical("[services]", fmt.Sprintf("Error stopping `%s`:\n\n%s", unit.Name, err.Error()))
			return err
		}
	case "kill":
		err = unit.Kill()
		if err != nil {
			l.Errorw("[perform]", "err", err)
			ui.NotifyCritical("[services]", fmt.Sprintf("Error killing `%s`:\n\n%s", unit.Name, err.Error()))
			return err
		}
	case "stop/follow":
		err = unit.Stop()
		if err != nil {
			l.Errorw("[perform]", "err", err)
			ui.NotifyCritical("[services]", fmt.Sprintf("Error stopping `%s`:\n\n%s", unit.Name, err.Error()))
			return err
		}
		err = unit.ShowJournal(true, ctx.String("tmux-session"), ctx.String("term-command"))
		if err != nil {
			l.Errorw("[perform]", "err", err)
			ui.NotifyCritical("[services]", fmt.Sprintf("Error following journal for `%s`:\n\n%s", unit.Name, err.Error()))
			return err
		}
	case "restart":
		err = unit.Restart()
		if err != nil {
			l.Errorw("[perform]", "err", err)
			ui.NotifyCritical("[services]", fmt.Sprintf("Error restarting `%s`:\n\n%s", unit.Name, err.Error()))
			return err
		}
	case "restart/follow":
		err = unit.Restart()
		if err != nil {
			l.Errorw("[perform]", "err", err)
			ui.NotifyCritical("[services]", fmt.Sprintf("Error restarting `%s`:\n\n%s", unit.Name, err.Error()))
			return err
		}
		err = unit.ShowJournal(true, ctx.String("tmux-session"), ctx.String("term-command"))
		if err != nil {
			l.Errorw("[perform]", "err", err)
			ui.NotifyCritical("[services]", fmt.Sprintf("Error following journal for `%s`:\n\n%s", unit.Name, err.Error()))
			return err
		}
	case "show":
		err = unit.Show(ctx.String("tmux-session"), ctx.String("term-command"))
		if err != nil {
			l.Errorw("[perform]", "err", err)
			ui.NotifyCritical("[services]", fmt.Sprintf("Error showing `%s`:\n\n%s", unit.Name, err.Error()))
			return err
		}
	case "journal":
		err = unit.ShowJournal(false, ctx.String("tmux-session"), ctx.String("term-command"))
		if err != nil {
			l.Errorw("[perform]", "err", err)
			ui.NotifyCritical("[services]", fmt.Sprintf("Error showing journal for `%s`:\n\n%s", unit.Name, err.Error()))
			return err
		}
	case "journal/follow":
		err = unit.ShowJournal(true, ctx.String("tmux-session"), ctx.String("term-command"))
		if err != nil {
			l.Errorw("[perform]", "err", err)
			ui.NotifyCritical("[services]", fmt.Sprintf("Error following journal for `%s`:\n\n%s", unit.Name, err.Error()))
			return err
		}
	case "status":
		err = unit.ShowStatus(ctx.String("tmux-session"), ctx.String("term-command"))
		if err != nil {
			l.Errorw("[perform]", "err", err)
			ui.NotifyCritical("[services]", fmt.Sprintf("Error showing status for `%s`:\n\n%s", unit.Name, err.Error()))
			return err
		}
	}
	ui.NotifyNormal(fmt.Sprintf("[services :: %s]", operation), unit.Name)

	return nil
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "Services"
	app.Usage = "Manages Systemd services (and timers)"
	app.Description = "Services"
	app.Version = "0.0.1#master"

	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:     "invalidate-cache",
			Aliases:  []string{"i"},
			Usage:    "Whether to invalidate services metadata cache",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "tmux-session",
			Aliases:  []string{"t"},
			EnvVars:  []string{impl.EnvPrefix + "_TMUX_SESSION"},
			Usage:    "Default TMUX session to use",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "term-command",
			Aliases:  []string{"c"},
			EnvVars:  []string{impl.EnvPrefix + "_TERMINAL_CMD"},
			Usage:    "Terminal command to use as a Tmux fallback option",
			Required: true,
		},
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
	app.Action = perform
	return app
}

func main() {
	var err error
	logger = impl.NewLogger()
	defer logger.Sync()
	l := logger.Sugar()
	r, err = redis.NewRedisLocal()
	if err != nil {
		l.Errorw("[main]", "err", err)
	}
	app := createCLI()
	err = app.Run(os.Args)
	if err != nil {
		l.Errorw("[main]", "err", err)
	}
}
