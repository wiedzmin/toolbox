package systemd

import (
	"fmt"
	"strings"

	"github.com/wiedzmin/toolbox/impl/shell"
)

// Service struct provides SystemD unit metadata
type Service struct {
	Name string // service unit name should be sufficient for now
	User bool
}

func sysctlCmd(user bool, cmd, name string) string {
	var tokens []string
	if user {
		tokens = []string{"systemctl", "--user", cmd, name}
	} else {
		tokens = []string{"systemctl", cmd, name}
	}
	var result strings.Builder
	for _, t := range tokens {
		result.WriteString(fmt.Sprintf("%s ", t))
	}
	return strings.TrimSpace(result.String())
}

// Restart restarts service unit
func (s *Service) Restart() error {
	_, err := shell.ShellCmd(sysctlCmd(s.User, "restart", s.Name), nil, nil, false, false)
	return err
}

// Start starts service unit
func (s *Service) Start() error {
	_, err := shell.ShellCmd(sysctlCmd(s.User, "start", s.Name), nil, nil, false, false)
	return err
}

// Stop stops service unit
func (s *Service) Stop() error {
	_, err := shell.ShellCmd(sysctlCmd(s.User, "stop", s.Name), nil, nil, false, false)
	return err
}

// Enable enables service
func (s *Service) Enable() error {
	_, err := shell.ShellCmd(sysctlCmd(s.User, "enable", s.Name), nil, nil, false, false)
	return err
}

// Disable disables service
func (s *Service) Disable() error {
	_, err := shell.ShellCmd(sysctlCmd(s.User, "disable", s.Name), nil, nil, false, false)
	return err
}

// IsActive checks if the service unit is active
func (s *Service) IsActive() (bool, error) {
	out, err := shell.ShellCmd(sysctlCmd(s.User, "disable", s.Name), nil, nil, false, false)
	if err != nil {
		return false, err
	}

	switch strings.TrimSpace(*out) {
	case "active":
		return true, nil
	case "inactive":
		return false, nil
	default:
		return false, fmt.Errorf("unknown status '%s'", *out)
	}
}

// TODO: implement "try-restart" command
