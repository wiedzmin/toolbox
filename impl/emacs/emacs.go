package emacs

import (
	"fmt"
	"os/user"

	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/shell"
	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	logger = impl.NewLogger()
}

func SocketPath() (*string, error) {
	l := logger.Sugar()
	userInfo, err := user.Current()
	l.Debugw("[SocketPath]", "userInfo", userInfo, "err", err)
	if err != nil {
		return nil, err
	}
	result := fmt.Sprintf("/run/user/%s/emacs/server", userInfo.Uid)
	l.Debugw("[SocketPath]", "socket path", result)
	return &result, nil
}

func SendToServer(elisp string) error {
	l := logger.Sugar()
	l.Debugw("[SendToServer]", "elisp", elisp)
	socketPath, err := SocketPath()
	if err != nil {
		return err
	}
	_, err = shell.ShellCmd(fmt.Sprintf("emacsclient -c -s %s -e '%s'", *socketPath, elisp), nil, nil, false, false)
	if err != nil {
		return err
	}
	return nil
}
