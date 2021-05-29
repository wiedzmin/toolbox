package emacs

import (
	"fmt"
	"os/user"

	"github.com/wiedzmin/toolbox/impl/shell"
)

func SocketPath() (*string, error) {
	userInfo, err := user.Current()
	if err != nil {
		return nil, err
	}
	result := fmt.Sprintf("/run/user/%s/emacs/server", userInfo.Uid)
	return &result, nil
}

func SendToServer(elisp string) error {
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
