package shell

import (
	"os"
	"os/exec"
	"strings"
)

func ShellCmd(cmd string, input *string, needOutput bool) (*string, error) {
	c := exec.Command("sh", "-c", cmd)
	c.Stderr = os.Stderr
	if input != nil {
		reader := strings.NewReader(*input)
		c.Stdin = reader
	}
	if needOutput {
		out, err := c.Output()
		result := strings.TrimRight(string(out), "\n")
		return &result, err
	} else {
		err := c.Run()
		return nil, err
	}
}
