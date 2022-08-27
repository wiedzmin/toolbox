package shell

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/wiedzmin/toolbox/impl"
	"go.uber.org/zap"
)

const (
	TerminalCommandFlagName = "term-command"
)

var logger *zap.Logger
type ErrInvalidCmd struct {
	Cmd string
}

func (e ErrInvalidCmd) Error() string {
	return fmt.Sprintf("invalid shell command: '%s'", e.Cmd)
}

type ErrNoEnvVar struct {
	Name string
}

func (e ErrNoEnvVar) Error() string {
	return fmt.Sprintf("no value found for: %s", e.Name)
}

func init() {
	logger = impl.NewLogger()
}

// ShellCmd executes shell commands
// environment variables are provided as string slice of "<name>=<value>" entries
func ShellCmd(cmd string, input *string, cwd *string, env []string, needOutput, combineOutput bool) (*string, error) {
	l := logger.Sugar()
	c := exec.Command("sh", "-c", cmd)
	c.Env = append(os.Environ(), env...)
	if input != nil {
		reader := strings.NewReader(*input)
		c.Stdin = reader
	}
	if cwd != nil {
		c.Dir = *cwd
	}

	l.Debugw("[ShellCmd]", "cmd", cmd, "env", env, "input", input)
	if needOutput {
		var out []byte
		var err error
		if combineOutput {
			out, err = c.CombinedOutput()
		} else {
			out, err = c.Output()
		}
		result := strings.TrimRight(string(out), "\n")
		l.Debugw("[ShellCmd]", "cmd", cmd, "result", result, "err", err)
		return &result, err
	} else {
		err := c.Run()
		return nil, err
	}
}

func RunInBareTerminal(cmd, vtermCmd string) error {
	// TODO: elaborate/ensure `transient` commands proper handling, i.e. those who need "; read" thereafter
	_, err := ShellCmd(fmt.Sprintf("%s \"sh -c %s\"", vtermCmd, cmd), nil, nil, nil, false, false)
	return err
}
