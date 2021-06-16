package tmux

import (
	"fmt"
	"strings"

	"github.com/wiedzmin/toolbox/impl/shell"
)

type Session struct {
	Name string
}

type ErrSessionNotFound struct {
	Name string
}

func (e ErrSessionNotFound) Error() string {
	return fmt.Sprintf("tmuxp: session '%s' not exist", e.Name)
}

func GetSession(name string, create, attach bool) (*Session, error) {
	out, err := shell.ShellCmd(fmt.Sprintf("tmux has-session -t %s", name), nil, nil, true, false)
	if err != nil {
		return nil, err
	}
	if len(*out) > 0 {
		if create {
			_, err := shell.ShellCmd(fmt.Sprintf("tmux switch-client -t %s", name), nil, nil, false, false)
			if err != nil {
				return nil, err
			}
			return &Session{Name: name}, nil
		} else {
			return nil, ErrSessionNotFound{Name: name}
		}
	}
	return &Session{Name: name}, nil
}

func (s *Session) NewWindow(cmd, title, startDirectory string, attach bool) error {
	args := []string{
		fmt.Sprintf("-t %s", s.Name),
		fmt.Sprintf("-n %s", title),
	}
	var argsStr strings.Builder
	if !attach {
		args = append(args, "-d")
	}
	if len(startDirectory) > 0 {
		args = append(args, fmt.Sprintf("-c %s", startDirectory))
	}
	args = append(args, fmt.Sprintf("'%s'", cmd))
	for _, arg := range args {
		argsStr.WriteString(fmt.Sprintf("%s ", arg))
	}

	tmuxCmd := fmt.Sprintf("tmux new-window %s", strings.TrimSpace(argsStr.String()))
	_, err := shell.ShellCmd(tmuxCmd, nil, nil, false, false)
	if err != nil {
		return err
	}
	return nil
}
