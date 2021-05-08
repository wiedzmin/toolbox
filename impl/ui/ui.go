package ui

import (
	"fmt"
	"strings"

	"github.com/wiedzmin/toolbox/impl/shell"
)

const (
	rofiOptionsSeparator = "@"
)

// GetSelectionRofi returns users choice from list of options, using Rofi selector tool
func GetSelectionRofi(seq []string, prompt string) (string, error) {
	seqStr := strings.Join(seq, rofiOptionsSeparator)
	result, err := shell.ShellCmd(fmt.Sprintf("rofi -dmenu -sep %s -p '%s'", rofiOptionsSeparator, prompt), &seqStr, true)
	return *result, err
}
