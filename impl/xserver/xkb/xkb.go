package xkb

import (
	"fmt"
	"os"
	"strings"

	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/shell"
	"github.com/wiedzmin/toolbox/impl/ui"
	"go.uber.org/zap"
)

var (
	logger *zap.Logger
)

func init() {
	logger = impl.NewLogger()
	impl.EnsureBinary("xkb-switch", *logger)
}

func GetKeyboardLayouts() ([]string, error) {
	result, err := shell.ShellCmd("xkb-switch -l", nil, nil, true, false)
	if err != nil {
		return nil, err
	}
	layouts := strings.Split(*result, "\n")
	return layouts, nil
}

func GetCurrentKeyboardLayout() (*string, error) {
	result, err := shell.ShellCmd("xkb-switch -p", nil, nil, true, false)
	return result, err
}

func SetNextKeyboardLayout() error {
	_, err := shell.ShellCmd("xkb-switch -n", nil, nil, false, false)
	return err
}

func SetKeyboardLayoutByName(layout string) error {
	_, err := shell.ShellCmd(fmt.Sprintf("xkb-switch -s %s", layout), nil, nil, false, false)
	return err
}

func EnsureEnglishKeyboardLayout() {
	var err error
	layout, err := GetCurrentKeyboardLayout()
	if *layout != "us" {
		err = SetKeyboardLayoutByName("us")
	}
	if err != nil {
		ui.NotifyCritical("[xkb]", "Error setting keyboard layout")
		os.Exit(1)
	}
}
