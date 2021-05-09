package ui

import (
	"fmt"
	"strings"

	"github.com/0xAX/notificator"
	"github.com/wiedzmin/toolbox/impl/shell"
)

const (
	rofiOptionsSeparator = "@"
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
	result, err := shell.ShellCmd(fmt.Sprintf("rofi -dmenu -sep %s -p '%s'", rofiOptionsSeparator, prompt), &seqStr, nil, true)
	return *result, err
}

func NotifyNormal(title, text string) {
	notify.Push(title, text, "", notificator.UR_NORMAL)
}

func NotifyCritical(title, text string) {
	notify.Push(title, text, "", notificator.UR_CRITICAL)
}
