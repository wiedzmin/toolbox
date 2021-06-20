package ui

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/0xAX/notificator"
	"github.com/wiedzmin/toolbox/impl/shell"
	"go.uber.org/zap"
)

const (
	rofiOptionsSeparator  = "\n"
	dmenuOptionsSeparator = "\n"
)

var notify *notificator.Notificator
var logger *zap.Logger

func init() {
	notify = notificator.New(notificator.Options{
		// DefaultIcon: "icon/default.png",
		AppName: "webjumps",
	})
	logger, _ = zap.NewProduction()
	l := logger.Sugar()
	rofiPath, err := exec.LookPath("rofi")
	if err != nil {
		l.Warnw("[ui::init] rofi not found")
		os.Exit(1)
	}
	l.Debugw("[ui::init]", "rofi", rofiPath)
	dmenuPath, err := exec.LookPath("dmenu")
	if err != nil {
		l.Warnw("[ui::init] dmenu not found")
		os.Exit(1)
	}
	l.Debugw("[ui::init]", "dmenu", dmenuPath)
}

// GetSelectionRofi returns users choice from list of options, using Rofi selector tool
func GetSelectionRofi(seq []string, prompt string) (string, error) {
	l := logger.Sugar()
	sort.Strings(seq)
	seqStr := strings.Join(seq, rofiOptionsSeparator)
	l.Debugw("[ui::GetSelectionRofi]", "seq", seq, "seqStr", seqStr)
	result, err := shell.ShellCmd(fmt.Sprintf("rofi -dmenu -i -sep '%s' -p '%s'", rofiOptionsSeparator, prompt), &seqStr, nil, true, false)
	return *result, err
}

func GetSelectionDmenu(seq []string, prompt string, lines int, font string) (string, error) {
	l := logger.Sugar()
	sort.Strings(seq)
	seqStr := strings.Join(seq, dmenuOptionsSeparator)
	l.Debugw("[ui::GetSelectionDmenu]", "seq", seq, "seqStr", seqStr)
	result, err := shell.ShellCmd(fmt.Sprintf("dmenu -i -p '%s' -l %d -fn '%s'", prompt, lines, font),
		&seqStr, nil, true, false)
	return *result, err
}

func GetSelectionDmenuWithCase(seq []string, prompt string, lines int, font string) (string, error) {
	l := logger.Sugar()
	sort.Strings(seq)
	seqStr := strings.Join(seq, dmenuOptionsSeparator)
	l.Debugw("[ui::GetSelectionDmenuWithCase]", "seq", seq, "seqStr", seqStr)
	result, err := shell.ShellCmd(fmt.Sprintf("dmenu -p %s -l %d -fn %s", prompt, lines, font), &seqStr, nil, true, false)
	return *result, err
}

// TODO: make selector function for fzf, see example(s): https://junegunn.kr/2016/02/using-fzf-in-your-program

func NotifyNormal(title, text string) {
	notify.Push(title, text, "", notificator.UR_NORMAL)
}

func NotifyCritical(title, text string) {
	notify.Push(title, text, "", notificator.UR_CRITICAL)
}
