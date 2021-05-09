package shell

import (
	"os"
	"os/exec"
	"strings"
)

// ShellCmd executes shell commands
// environment variables are provided in form of "<name>=<value>"
func ShellCmd(cmd string, input *string, env []string, needOutput bool) (*string, error) {
	c := exec.Command("sh", "-c", cmd)
	c.Env = append(os.Environ(), env...)
	if input != nil {
		reader := strings.NewReader(*input)
		c.Stdin = reader
	}
	if needOutput {
		out, err := c.CombinedOutput()
		result := strings.TrimRight(string(out), "\n")
		return &result, err
	} else {
		err := c.Run()
		return nil, err
	}
}
