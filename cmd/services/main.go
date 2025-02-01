package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/redis"
	"github.com/wiedzmin/toolbox/impl/shell"
	"github.com/wiedzmin/toolbox/impl/shell/tmux"
	"github.com/wiedzmin/toolbox/impl/systemd"
	"github.com/wiedzmin/toolbox/impl/ui"
	"github.com/wiedzmin/toolbox/impl/xserver/xkb"
	"go.uber.org/zap"
)

const (
	redisKeyName     = "system/services"
	redisKeyNameFlat = "system/services/flat"
)

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

func ensureUnitsCache(ctx *cli.Context) error {
	var units []systemd.Unit
	var err error
	l := logger.Sugar()
	flat := ctx.Bool("flat")
	l.Debugw("[ensureUnitsCache]", "flat", flat)
	if !r.KeyExists(redisKeyName) || !r.KeyExists(redisKeyNameFlat) && flat {
		l.Debugw("[ensureUnitsCache]", "units cache", "does not exist, populating")
		units, err = systemd.CollectUnits(true, true)
		l.Debugw("[ensureUnitsCache]", "found units", len(units))
		if err != nil {
			return err
		}
	} else {
		l.Debugw("[ensureUnitsCache]", "units cache", "exists, moving on")
		return nil
	}
	ui.NotifyNormal("[services]", "populating cache, please wait...")
	for _, u := range units {
		err = r.AppendToList(redisKeyName, u.String())
		if err != nil {
			return err
		}
		for _, op := range OPERATIONS {
			_ = r.AppendToList(redisKeyNameFlat, fmt.Sprintf("%s / %s", u.String(), op))
		}
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
		err = r.DeleteValue(redisKeyNameFlat)
		if err != nil {
			return err
		}
		return nil
	}

	err = ensureUnitsCache(ctx)
	if err != nil {
		return err
	}

	var entries []string
	var redisKey string
	if ctx.Bool("flat") {
		redisKey = redisKeyNameFlat
	} else {
		redisKey = redisKeyName
	}
	entries, _ = r.GetList(redisKey, 0, -1)
	xkb.EnsureEnglishKeyboardLayout()
	entry, err := ui.GetSelection(entries, "select", ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), true, false)
	if err != nil {
		return err
	}
	xkb.EnsureEnglishKeyboardLayout()
	var operation string
	var unit systemd.Unit

	if ctx.Bool("flat") {
		entryChunks := strings.Split(entry, "/")
		// FIXME: ensure sort order
		unit = systemd.UnitFromString(strings.Trim(entryChunks[0], " "))
		operation = strings.Trim(entryChunks[1], " ")
	} else {
		unit = systemd.UnitFromString(entry)
		// FIXME: ensure sort order
		operation, err = ui.GetSelection(OPERATIONS, "perform", ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), true, false)
		if err != nil {
			return err
		}
	}
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
		err = unit.ShowJournal(shell.TermTraitsFromContext(ctx), true, ctx.Bool(systemd.DumpCmdFlagName))
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
		err = unit.ShowJournal(shell.TermTraitsFromContext(ctx), true, ctx.Bool(systemd.DumpCmdFlagName))
		if err != nil {
			l.Errorw("[perform]", "err", err)
			ui.NotifyCritical("[services]", fmt.Sprintf("Error following journal for `%s`:\n\n%s", unit.Name, err.Error()))
			return err
		}
	case "show":
		err = unit.Show(shell.TermTraitsFromContext(ctx), ctx.Bool(systemd.DumpCmdFlagName))
		if err != nil {
			l.Errorw("[perform]", "err", err)
			ui.NotifyCritical("[services]", fmt.Sprintf("Error showing `%s`:\n\n%s", unit.Name, err.Error()))
			return err
		}
	case "journal":
		err = unit.ShowJournal(shell.TermTraitsFromContext(ctx), false, ctx.Bool(systemd.DumpCmdFlagName))
		if err != nil {
			l.Errorw("[perform]", "err", err)
			ui.NotifyCritical("[services]", fmt.Sprintf("Error showing journal for `%s`:\n\n%s", unit.Name, err.Error()))
			return err
		}
	case "journal/follow":
		err = unit.ShowJournal(shell.TermTraitsFromContext(ctx), true, ctx.Bool(systemd.DumpCmdFlagName))
		if err != nil {
			l.Errorw("[perform]", "err", err)
			ui.NotifyCritical("[services]", fmt.Sprintf("Error following journal for `%s`:\n\n%s", unit.Name, err.Error()))
			return err
		}
	case "status":
		err = unit.ShowStatus(shell.TermTraitsFromContext(ctx), ctx.Bool(systemd.DumpCmdFlagName))
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
		&cli.BoolFlag{
			Name:     "flat",
			Aliases:  []string{"F"},
			Usage:    "Whether to display cartesian product of units and ops for faster fuzzy selection",
			Required: false,
		},
		&cli.BoolFlag{
			Name:     systemd.DumpCmdFlagName,
			Aliases:  []string{"d"},
			Usage:    "Do nothing, copy systemd CLI command to clipboard instead",
			Required: false,
		},
		&cli.StringFlag{
			Name:     tmux.SessionFlagName,
			Aliases:  []string{"t"},
			EnvVars:  []string{impl.EnvPrefix + "_TMUX_SESSION"},
			Usage:    "Default TMUX session to use",
			Required: false,
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
			Value:    ui.SelectorToolDefault,
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
