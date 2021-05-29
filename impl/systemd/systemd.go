package systemd

import (
	"fmt"
	"os/exec"
	"strings"
)

// Service struct provides SystemD unit metadata
type Service struct {
	Name string // service unit name should be sufficient for now
	User bool
}

// Restart restarts service unit
func (s *Service) Restart() error {
	if s.User {
		return exec.Command("systemctl", "--user", "restart", s.Name).Run()
	} else {
		return exec.Command("systemctl", "restart", s.Name).Run()
	}
}

// Start starts service unit
func (s *Service) Start() error {
	if s.User {
		return exec.Command("systemctl", "--user", "start", s.Name).Run()
	} else {
		return exec.Command("systemctl", "start", s.Name).Run()
	}
}

// Stop stops service unit
func (s *Service) Stop() error {
	if s.User {
		return exec.Command("systemctl", "--user", "stop", s.Name).Run()
	} else {
		return exec.Command("systemctl", "stop", s.Name).Run()
	}
}

// Enable enables service
func (s *Service) Enable() error {
	if s.User {
		return exec.Command("systemctl", "--user", "enable", s.Name).Run()
	} else {
		return exec.Command("systemctl", "enable", s.Name).Run()
	}
}

// Disable disables service
func (s *Service) Disable() error {
	if s.User {
		return exec.Command("systemctl", "--user", "disable", s.Name).Run()
	} else {
		return exec.Command("systemctl", "disable", s.Name).Run()
	}
}

// IsActive checks if the service unit is active
func (s *Service) IsActive() (bool, error) {
	args := []string{"is-active", s.Name}
	if s.User {
		args = []string{"--user", "is-active", s.Name}
	}
	out, err := exec.Command("systemctl", args...).Output()
	if err != nil {
		return false, err
	}

	switch strings.TrimSpace(string(out)) {
	case "active":
		return true, nil
	case "inactive":
		return false, nil
	default:
		return false, fmt.Errorf("unknown status '%s'", string(out))
	}
}

// TODO: implement "try-restart" command
