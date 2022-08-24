package tmux

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/wiedzmin/toolbox/impl"
	"go.uber.org/zap"
)

const (
	SessionFlagName = "tmux-session"
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

var logger *zap.Logger

func init() {
	logger = impl.NewLogger()
}

func GetSession(name string, create, attach bool) (*Session, error) {
	l := logger.Sugar()
	l.Debugw("[GetSession]", "name", name, "create", create, "attach", attach)
	c := exec.Command("sh", "-c", fmt.Sprintf("tmux has-session -t %s", name))
	out, err := c.Output()
	outStr := strings.TrimRight(string(out), "\n")
	l.Debugw("[GetSession]", "out", out, "err", err)
	if err != nil {
		return nil, err
	}
	if len(outStr) > 0 {
		if create {
			c = exec.Command("sh", "-c", fmt.Sprintf("tmux new-session -d -s %s", name))
			err := c.Run()
			if err != nil {
				return nil, err
			}
			c = exec.Command("sh", "-c", fmt.Sprintf("tmux switch-client -t %s", name))
			err = c.Run()
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
	l := logger.Sugar()
	args := []string{
		fmt.Sprintf("-t %s", s.Name),
		fmt.Sprintf("-n '%s'", title),
	}
	l.Debugw(fmt.Sprintf("[%s.NewWindow]", s.Name), "cmd", cmd, "title", title, "startDirectory", startDirectory, "attach", attach)
	var argsStr strings.Builder
	if !attach {
		args = append(args, "-d")
	}
	if len(startDirectory) > 0 {
		args = append(args, fmt.Sprintf("-c %s", startDirectory))
	}
	args = append(args, fmt.Sprintf("\"%s\"", cmd))
	for _, arg := range args {
		argsStr.WriteString(fmt.Sprintf("%s ", arg))
	}

	tmuxCmd := fmt.Sprintf("tmux new-window %s", strings.TrimSpace(argsStr.String()))
	l.Debugw(fmt.Sprintf("[%s.NewWindow]", s.Name), "tmuxCmd", tmuxCmd)
	c := exec.Command("sh", "-c", tmuxCmd)
	err := c.Run()
	if err != nil {
		return err
	}
	return nil
}
