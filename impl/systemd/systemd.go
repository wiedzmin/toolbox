package systemd

import (
	"fmt"
	"strings"

	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/shell"
	"github.com/wiedzmin/toolbox/impl/tmux"
	"go.uber.org/zap"
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

var logger *zap.Logger

func init() {
	logger = impl.NewLogger()
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
	r := strings.TrimSpace(result.String())

	return r
}

func jctlCmd(user, follow bool, name string) string {
	tokens := []string{"journalctl"}
	if user {
		tokens = append(tokens, "--user")
	}
	if follow {
		tokens = append(tokens, "--follow")
	}
	tokens = append(tokens, fmt.Sprintf("-u %s", name))
	var result strings.Builder
	for _, t := range tokens {
		result.WriteString(fmt.Sprintf("%s ", t))
	}
	r := strings.TrimSpace(result.String())

	return r
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
	l := logger.Sugar()
	out, err := shell.ShellCmd(sysctlCmd(s.User, "is-active", s.Name), nil, nil, false, false)
	if err != nil {
		return false, err
	}
	result := strings.TrimSpace(*out)
	l.Debugw(fmt.Sprintf("[%s.IsActive]", s.Name), "result", result)

	switch result {
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
	l := logger.Sugar()
	l.Debugw("[DaemonReload]")
	_, err := shell.ShellCmd("pkexec systemctl daemon-reload", nil, nil, false, false)
	return err
}

// CollectUnits returns slice of active services and timers
func CollectUnits(system, user bool) ([]Service, error) {
	l := logger.Sugar()
	var units []Service
	var cases = []struct {
		isUser bool
		cmd    string
	}{
		{false, "systemctl list-unit-files"},
		{true, "systemctl --user list-unit-files"},
	}
	l.Debugw("[CollectUnits]", "system", system, "user", user)
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
					l.Debugw("[CollectUnits]", "unit", unit)
					units = append(units, Service{Name: unit, User: c.isUser})
				}
			}
		}
	}
	return units, nil
}

func doShow(cmd, title, tmuxSession, vtermCmd string) error {
	l := logger.Sugar()
	l.Debugw("[doShow]", "cmd", cmd, "title", title, "tmuxSession", tmuxSession, "vtermCmd", vtermCmd)
	if len(vtermCmd) > 0 {
		if len(tmuxSession) > 0 {
			session, err := tmux.GetSession(tmuxSession, false, true)
			switch err.(type) {
			case tmux.ErrSessionNotFound:
				return shell.RunInTerminal(cmd, vtermCmd)
			default:
				return err
			}
			return session.NewWindow(cmd, title, "", true)
		} else {
			return shell.RunInTerminal(cmd, vtermCmd)
		}
	} else {
		return impl.ErrNotImplemented{Token: "ShowTextDialog"}
	}
	return nil
}

// ShowStatus shows unit's status in form of `systemctl status` output
func (s *Service) ShowStatus(tmuxSession, vtermCmd string) error {
	cmd := fmt.Sprintf("sh -c '%s'; read", sysctlCmd(s.User, "status", s.Name))
	title := fmt.Sprintf("status :: %s", s.Name)

	return doShow(cmd, title, tmuxSession, vtermCmd)
}

// ShowStatus shows unit's journal in form of `journalctl` output
func (s *Service) ShowJournal(follow bool, tmuxSession, vtermCmd string) error {
	cmd := fmt.Sprintf("sh -c '%s'", jctlCmd(s.User, follow, s.Name))
	title := fmt.Sprintf("journal :: %s", s.Name)
	if follow {
		cmd = fmt.Sprintf("sh -c '%s'; read", jctlCmd(s.User, follow, s.Name))
		title = fmt.Sprintf("journal/follow :: %s", s.Name)
	}

	return doShow(cmd, title, tmuxSession, vtermCmd)
}

// TryRestart tries to restart unit
func (s *Service) TryRestart() error {
	_, err := shell.ShellCmd(sysctlCmd(s.User, "try-restart", s.Name), nil, nil, false, false)
	return err
}
