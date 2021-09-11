package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/0xAX/notificator"
	"github.com/wiedzmin/toolbox/impl"
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
		AppName: "toolbox",
	})
	logger = impl.NewLogger()
}

// GetSelectionRofi returns users choice from list of options, using Rofi selector tool
func GetSelectionRofi(seq []string, prompt string, normalWindow bool) (string, error) {
	impl.EnsureBinary("rofi", *logger)
	l := logger.Sugar()
	sort.Strings(seq)
	seqStr := strings.Join(seq, rofiOptionsSeparator)
	l.Debugw("[GetSelectionRofi]", "seq", seq, "seqStr", seqStr, "normalWindow", normalWindow)
	normalWindowStr := ""
	if normalWindow {
		normalWindowStr = " -normal-window"
	}
	result, err := shell.ShellCmd(fmt.Sprintf("rofi%s -dmenu -i -sep '%s' -p '%s'",
		normalWindowStr, rofiOptionsSeparator, prompt), &seqStr, nil, true, false)
	return *result, err
}

func GetSelectionDmenu(seq []string, prompt string, lines int, withCase bool, font string) (string, error) {
	impl.EnsureBinary("dmenu", *logger)
	l := logger.Sugar()
	sort.Strings(seq)
	seqStr := strings.Join(seq, dmenuOptionsSeparator)
	l.Debugw("[GetSelectionDmenu]", "seq", seq, "seqStr", seqStr, "case-sensitive", withCase)
	caseFlagStr := ""
	if withCase {
		caseFlagStr = " -i"
	}
	result, err := shell.ShellCmd(fmt.Sprintf("dmenu%s -p '%s' -l %d -fn '%s'", caseFlagStr, prompt, lines, font),
		&seqStr, nil, true, false)
	return *result, err
}

// TODO: make selector function for fzf, see example(s): https://junegunn.kr/2016/02/using-fzf-in-your-program

func NotifyNormal(title, text string) {
	notify.Push(title, text, "", notificator.UR_NORMAL)
}

func NotifyCritical(title, text string) {
	notify.Push(title, text, "", notificator.UR_CRITICAL)
}
