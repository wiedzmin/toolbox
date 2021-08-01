package emacs

import (
	"fmt"

	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/shell"
	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	logger = impl.NewLogger()
}

func SocketPath() (*string, error) {
	return impl.AtRunUser("emacs/server")
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
