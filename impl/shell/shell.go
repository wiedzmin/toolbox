package shell

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ShellCmd executes shell commands
// environment variables are provided as string slice of "<name>=<value>" entries
func ShellCmd(cmd string, input *string, env []string, needOutput, combineOutput bool) (*string, error) {
	c := exec.Command("sh", "-c", cmd)
	c.Env = append(os.Environ(), env...)
	if input != nil {
		reader := strings.NewReader(*input)
		c.Stdin = reader
	}
	if needOutput {
		var out []byte
		var err error
		if combineOutput {
			out, err = c.CombinedOutput()
		} else {
			out, err = c.Output()
		}
		result := strings.TrimRight(string(out), "\n")
		return &result, err
	} else {
		err := c.Run()
		return nil, err
	}
}

func RunInTerminal(cmd, vtermCmd string) error {
	_, err := ShellCmd(fmt.Sprintf("%s '%s'", vtermCmd, cmd), nil, nil, false, false)
	return err
}
