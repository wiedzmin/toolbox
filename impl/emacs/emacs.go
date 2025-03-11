package emacs

import (
	"fmt"
	"os"

	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/fs"
	"github.com/wiedzmin/toolbox/impl/shell"
	"github.com/wiedzmin/toolbox/impl/systemd"
	"github.com/wiedzmin/toolbox/impl/ui"
	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	logger = impl.NewLogger()
}

func SocketPath() (*string, error) {
	return fs.AtRunUser("emacs/server")
}

func ServiceState(tag string, restart bool) {
	l := logger.Sugar()
	service := systemd.Unit{Name: "emacs.service", User: true}
	l.Debugw("[emacs.ServiceState]", "service", service)
	isActive, err := service.IsActive()
	if err != nil {
		l.Errorw("[emacs.ServiceState]", "err", err)
	}
	if !isActive {
		l.Errorw("[emacs.ServiceState]", "state", "not running")
		if restart {
			l.Errorw("[emacs.ServiceState]", "NOTE", "restarting is not properly implemented yet")
			// l.Errorw("[emacs.ServiceState]", "state", "restarting")
			// ui.NotifyCritical(tag, "Emacs service not running, trying to restart...")
			// err = service.Restart() // FIXME: elaborate repeating logic and state synchronization, consider merging logic with implementation from `vpn` module
			// time.Sleep(3 * time.Second)
			// if err != nil {
			// 	l.Errorw("[emacs.ServiceState]", "err", err)
			// 	ui.NotifyCritical(tag, "Emacs service failed to start")
			// 	os.Exit(1)
			// }
			ui.NotifyCritical(tag, "Emacs service not running")
			os.Exit(1)
		} else {
			ui.NotifyCritical(tag, "Emacs service not running")
			os.Exit(1)
		}
	}
}

func SendToServer(elisp string, createFrame bool) error {
	l := logger.Sugar()
	l.Debugw("[SendToServer]", "elisp", elisp)
	socketPath, err := SocketPath()
	if err != nil {
		return err
	}

	var createFrameStr string
	if createFrame {
		createFrameStr = "-c "
	} else {
		createFrameStr = ""
	}
	_, err = shell.ShellCmd(fmt.Sprintf("emacsclient %s-s %s -e '%s'", createFrameStr, *socketPath, elisp), nil, nil, nil, false, false)
	if err != nil {
		return err
	}
	return nil
}
