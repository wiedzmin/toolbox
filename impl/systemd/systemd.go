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

const (
	UNIT_TYPE_SERVICE string = "service"
	UNIT_TYPE_TIMER   string = "timer"
	UNIT_TYPE_SLICE   string = "slice"
	UNIT_TYPE_SOCKET  string = "socket"
	UNIT_TYPE_TARGET  string = "target"
)

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

// DaemonReload tries to reload systemd, mostly to take care of newly added/removed units
func DaemonReload() error {
	_, err := shell.ShellCmd("pkexec systemctl daemon-reload", nil, nil, false, false)
	return err
}

// CollectUnits returns slice of active services and timers
func CollectUnits(system, user bool) ([]Service, error) {
	var units []Service
	var cases = []struct {
		isUser bool
		cmd    string
	}{
		{false, "systemctl list-unit-files"},
		{true, "systemctl --user list-unit-files"},
	}
	for _, c := range cases {
		out, err := shell.ShellCmd(c.cmd, nil, nil, true, false)
		if err != nil {
			return nil, err
		}
		unitsSlice := strings.Split(*out, "\n")
		for _, unit := range unitsSlice[1 : len(unitsSlice)-1] {
			if len(unit) > 0 {
				unit = strings.Fields(unit)[0]
				if strings.HasSuffix(unit, UNIT_TYPE_SERVICE) || strings.HasSuffix(unit, UNIT_TYPE_TIMER) {
					units = append(units, Service{Name: unit, User: c.isUser})
				}
			}
		}
	}
	return units, nil
}

// TODO: implement "try-restart" command
