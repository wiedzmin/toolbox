package shell

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/wiedzmin/toolbox/impl"
	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	logger = impl.NewLogger()
}

// ShellCmd executes shell commands
// environment variables are provided as string slice of "<name>=<value>" entries
func ShellCmd(cmd string, input *string, path *string, env []string, needOutput, combineOutput bool) (*string, error) {
	l := logger.Sugar()
	c := exec.Command("sh", "-c", cmd)
	c.Env = append(os.Environ(), env...)
	if input != nil {
		reader := strings.NewReader(*input)
		c.Stdin = reader
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

func RunInTerminal(cmd, vtermCmd string) error {
	_, err := ShellCmd(fmt.Sprintf("%s '%s'", vtermCmd, cmd), nil, nil, nil, false, false)
	return err
}
