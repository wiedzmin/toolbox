package systemd

import (
	"fmt"
	"os/exec"
	"strings"
)

// Service struct provides SystemD unit metadata
type Service struct {
	Name string // service unit name should be sufficient for now
}

// Restart restarts service unit
func (s *Service) Restart() error {
	if err := exec.Command("systemctl", "restart", s.Name).Run(); err != nil {
		return err
	}
	return nil
}

// Start starts service unit
func (s *Service) Start() error {
	if err := exec.Command("systemctl", "start", s.Name).Run(); err != nil {
		return err
	}
	return nil
}

// Stop stops service unit
func (s *Service) Stop() error {
	if err := exec.Command("systemctl", "stop", s.Name).Run(); err != nil {
		return err
	}
	return nil
}

// Enable enables service
func (s *Service) Enable() error {
	if err := exec.Command("systemctl", "enable", s.Name).Run(); err != nil {
		return err
	}
	return nil
}

// Disable disables service
func (s *Service) Disable() error {
	if err := exec.Command("systemctl", "disable", s.Name).Run(); err != nil {
		return err
	}
	return nil
}

// IsActive checks if the service unit is active
func (s *Service) IsActive() (bool, error) {
	out, err := exec.Command("systemctl", "is-active", s.Name).Output()
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
