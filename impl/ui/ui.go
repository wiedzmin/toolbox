package ui

import (
	"fmt"
	"strings"

	"github.com/0xAX/notificator"
	"github.com/wiedzmin/toolbox/impl/shell"
)

const (
	rofiOptionsSeparator  = "@"
	dmenuOptionsSeparator = "\n"
)

var notify *notificator.Notificator

func init() {
	notify = notificator.New(notificator.Options{
		// DefaultIcon: "icon/default.png",
		AppName: "webjumps",
	})
}

// GetSelectionRofi returns users choice from list of options, using Rofi selector tool
func GetSelectionRofi(seq []string, prompt string) (string, error) {
	seqStr := strings.Join(seq, rofiOptionsSeparator)
	result, err := shell.ShellCmd(fmt.Sprintf("rofi -dmenu -i -sep %s -p '%s'", rofiOptionsSeparator, prompt), &seqStr, nil, true, false)
	return *result, err
}

func GetSelectionDmenu(seq []string, prompt string, lines int, font string) (string, error) {
	seqStr := strings.Join(seq, dmenuOptionsSeparator)
	result, err := shell.ShellCmd(fmt.Sprintf("dmenu -i -p '%s' -l %d -fn '%s'", prompt, lines, font),
		&seqStr, nil, true, false)
	return *result, err
}

func GetSelectionDmenuWithCase(seq []string, prompt string, lines int, font string) (string, error) {
	seqStr := strings.Join(seq, dmenuOptionsSeparator)
	result, err := shell.ShellCmd(fmt.Sprintf("dmenu -p %s -l %d -fn %s", prompt, lines, font), &seqStr, nil, true, false)
	return *result, err
}
func NotifyNormal(title, text string) {
	notify.Push(title, text, "", notificator.UR_NORMAL)
}

func NotifyCritical(title, text string) {
	notify.Push(title, text, "", notificator.UR_CRITICAL)
}
