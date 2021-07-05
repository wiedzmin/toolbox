package vpn

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/redis"
	"github.com/wiedzmin/toolbox/impl/shell"
	"github.com/wiedzmin/toolbox/impl/systemd"
	"github.com/wiedzmin/toolbox/impl/ui"
	"go.uber.org/zap"
)

const (
	ipV4StatusPath       = "/proc/sys/net/ipv4/conf/"
	ovpnAttemptsMax      = 15
	ovpnAttemptsInfinite = -1
)

var nmVpnActiveStatusCodes = []string{"3", "5"}

var (
	logger *zap.Logger
	r      *redis.Client
)

type ServiceNotFound struct {
	Name string
}

func (e ServiceNotFound) Error() string {
	return fmt.Sprintf("service `%s` not found", e.Name)
}

type Service struct {
	Name        string
	Type        string `json:"type"`
	Device      string `json:"dev"`
	UpCommand   string `json:"up"`
	DownCommand string `json:"down"`
}

type Services struct {
	data   []byte
	parsed map[string]Service
}

func init() {
	logger = impl.NewLogger()
	impl.EnsureBinary("nmcli", *logger)
	var err error
	r, err = redis.NewRedisLocal()
	if err != nil {
		l := logger.Sugar()
		l.Fatalw("[init]", "failed connecting to Redis", err)
	}
}

func NewServices(data []byte) (*Services, error) {
	var result Services
	result.data = data
	err := json.Unmarshal(data, &result.parsed)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func ServicesFromRedis(key string) (*Services, error) {
	r, err := redis.NewRedisLocal()
	if err != nil {
		return nil, err
	}
	servicesData, err := r.GetValue(key)
	if err != nil {
		return nil, err
	}
	return NewServices(servicesData)
}

func (vm *Services) Names() []string {
	var result []string
	for key := range vm.parsed {
		result = append(result, key)
	}
	return result
}

func (vm *Services) Get(key string) *Service {
	meta, ok := vm.parsed[key]
	if !ok {
		return nil
	}
	meta.Name = key
	return &meta
}

func nmIpsecVpnUp(name string) (bool, error) {
	result, err := shell.ShellCmd(fmt.Sprintf("nmcli con show id %s", name), nil, []string{"LANGUAGE=en_US.en"}, true, false)
	if err != nil {
		return false, err
	}
	scanner := bufio.NewScanner(strings.NewReader(*result))
	active := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "VPN.VPN-STATE") {
			for _, code := range nmVpnActiveStatusCodes {
				if strings.HasSuffix(line, code) {
					active = true
					break
				}
			}
		}
		if active {
			break
		}
	}
	return active, nil
}

func (vm *Services) StopRunning(omit []string, notify bool) error {
	l := logger.Sugar()
	devdns := systemd.Unit{Name: "docker-devdns.service"}
	err := devdns.Stop()
	if err != nil {
		return err
	}
	l.Debugw("[StopRunning]", "omit", omit)

	omitStr := strings.Join(omit, ",")
	for _, name := range vm.Names() {
		if omitStr != "" && strings.Contains(name, omitStr) {
			continue
		}
		l.Debugw("[StopRunning]", "name", name)
		ui.NotifyNormal("[VPN]", fmt.Sprintf("Stopping `%s`...", name))
		vm.Get(name).Stop(notify) // FIXME: check for nonexistent service
	}
	return nil
}

func startOVPN(name, device, cmd string, attempts int, notify bool) error {
	l := logger.Sugar()
	tun_path := fmt.Sprintf("%s%s", ipV4StatusPath, device)
	l.Debugw("[startOVPN]", "name", name, "device", device, "cmd", cmd, "attempts", attempts, "notify", notify)
	l.Debugw("[startOVPN]", "tun_path", tun_path)
	if _, err := os.Stat(tun_path); !os.IsNotExist(err) {
		r.SetValue(fmt.Sprintf("vpn/%s/is_up", name), "yes")
		if notify {
			ui.NotifyNormal("[VPN]", fmt.Sprintf("`%s` is up", name))
		}
		return nil
	} else {
		success := false
		_, err := shell.ShellCmd(cmd, nil, nil, false, false)
		if err == nil {
			attempt := 0
			for {
				if _, err := os.Stat(tun_path); !os.IsNotExist(err) {
					success = true
					break
				}
				time.Sleep(1 * time.Second)
				if attempts != ovpnAttemptsInfinite {
					attempt++
					if attempt >= attempts {
						err = fmt.Errorf("failed to start service for %d attempts", attempts)
						break
					}
				}
			}
		}
		if success {
			r.SetValue(fmt.Sprintf("vpn/%s/is_up", name), "yes")
			l.Debugw("[startOVPN]", fmt.Sprintf("vpn/%s/is_up", name), "yes")
			if notify {
				ui.NotifyNormal("[VPN]", fmt.Sprintf("Started `%s` service", name))
			}
			return nil
		} else {
			r.SetValue(fmt.Sprintf("vpn/%s/is_up", name), "unk")
			l.Debugw("[startOVPN]", fmt.Sprintf("vpn/%s/is_up", name), "unk")
			if notify {
				ui.NotifyCritical("[VPN]", fmt.Sprintf("Error starting `%s` service:\n\n%s", name, err.Error()))
			}
			return err
		}
	}
}

func startIPSec(name, cmd string, notify bool) error {
	l := logger.Sugar()
	up, err := nmIpsecVpnUp(name)
	if err != nil {
		return err
	}
	if up {
		r.SetValue(fmt.Sprintf("vpn/%s/is_up", name), "yes")
		l.Debugw("[startIPSec]", fmt.Sprintf("vpn/%s/is_up", name), "yes")
		if notify {
			ui.NotifyNormal("[VPN]", fmt.Sprintf("`%s` is up", name))
		}
		return nil
	} else {
		result, err := shell.ShellCmd(cmd, nil, []string{"LANGUAGE=en_US.en"}, true, true)
		if err != nil {
			if strings.Contains(*result, "is already active") {
				r.SetValue(fmt.Sprintf("vpn/%s/is_up", name), "yes")
				l.Debugw("[startIPSec]", fmt.Sprintf("vpn/%s/is_up", name), "yes")
				if notify {
					ui.NotifyNormal("[VPN]", fmt.Sprintf("`%s` is up", name))
				}
				return nil
			} else {
				r.SetValue(fmt.Sprintf("vpn/%s/is_up", name), "unk")
				l.Debugw("[startIPSec]", fmt.Sprintf("vpn/%s/is_up", name), "unk")
				if notify {
					ui.NotifyCritical("[VPN]", fmt.Sprintf("Error starting `%s` service:\n\n%s", name, err.Error()))
				}
				return err
			}
		} else {
			r.SetValue(fmt.Sprintf("vpn/%s/is_up", name), "yes")
			l.Debugw("[startIPSec]", fmt.Sprintf("vpn/%s/is_up", name), "yes")
			if notify {
				ui.NotifyNormal("[VPN]", fmt.Sprintf("`%s` is up", name))
			}
			return nil
		}
	}
}

func stopOVPN(name, device, cmd string, attempts int, notify bool) error {
	l := logger.Sugar()
	tun_path := fmt.Sprintf("%s%s", ipV4StatusPath, device)
	l.Debugw("[stopOVPN]", "name", name, "device", device, "cmd", cmd, "attempts", attempts, "notify", notify)
	l.Debugw("[stopOVPN]", "tun_path", tun_path)
	if _, err := os.Stat(tun_path); !os.IsNotExist(err) {
		r.SetValue(fmt.Sprintf("vpn/%s/is_up", name), "no")
		l.Debugw("[stopOVPN]", fmt.Sprintf("vpn/%s/is_up", name), "no")
		if notify {
			ui.NotifyNormal("[VPN]", fmt.Sprintf("`%s` is down", name))
		}
		return nil
	} else {
		success := false
		_, err := shell.ShellCmd(cmd, nil, nil, false, false)
		if err == nil {
			attempt := 0
			for {
				if _, err := os.Stat(tun_path); os.IsNotExist(err) {
					success = true
					break
				}
				time.Sleep(1 * time.Second)
				if attempts != ovpnAttemptsInfinite {
					attempt++
					if attempt >= attempts {
						err = fmt.Errorf("failed to start service for %d attempts", attempts)
						break
					}
				}
			}
		}
		if success {
			r.SetValue(fmt.Sprintf("vpn/%s/is_up", name), "no")
			l.Debugw("[stopOVPN]", fmt.Sprintf("vpn/%s/is_up", name), "no")
			if notify {
				ui.NotifyNormal("[VPN]", fmt.Sprintf("Stopped `%s` service", name))
			}
			return nil
		} else {
			r.SetValue(fmt.Sprintf("vpn/%s/is_up", name), "unk")
			l.Debugw("[stopOVPN]", fmt.Sprintf("vpn/%s/is_up", name), "unk")
			if notify {
				ui.NotifyCritical("[VPN]", fmt.Sprintf("Error stopping `%s` service:\n\n%s", name, err.Error()))
			}
			return err
		}
	}
}

func stopIPSec(name, cmd string, notify bool) error {
	l := logger.Sugar()
	up, err := nmIpsecVpnUp(name)
	if err != nil {
		return err
	}
	if !up {
		r.SetValue(fmt.Sprintf("vpn/%s/is_up", name), "no")
		l.Debugw("[stopIPSec]", fmt.Sprintf("vpn/%s/is_up", name), "no")
		if notify {
			ui.NotifyNormal("[VPN]", fmt.Sprintf("`%s` is down", name))
		}
		return nil
	} else {
		result, err := shell.ShellCmd(cmd, nil, []string{"LANGUAGE=en_US.en"}, true, true)
		if err != nil {
			if strings.Contains(*result, "not an active") {
				r.SetValue(fmt.Sprintf("vpn/%s/is_up", name), "no")
				l.Debugw("[stopIPSec]", fmt.Sprintf("vpn/%s/is_up", name), "no")
				if notify {
					ui.NotifyNormal("[VPN]", fmt.Sprintf("`%s` is down", name))
				}
				return nil
			} else {
				r.SetValue(fmt.Sprintf("vpn/%s/is_up", name), "unk")
				l.Debugw("[stopIPSec]", fmt.Sprintf("vpn/%s/is_up", name), "unk")
				if notify {
					ui.NotifyCritical("[VPN]", fmt.Sprintf("Error stopping `%s` service:\n\n%s", name, err.Error()))
				}
				return err
			}
		} else {
			r.SetValue(fmt.Sprintf("vpn/%s/is_up", name), "no")
			l.Debugw("[stopIPSec]", fmt.Sprintf("vpn/%s/is_up", name), "no")
			if notify {
				ui.NotifyNormal("[VPN]", fmt.Sprintf("`%s` is down", name))
			}
			return nil
		}
	}
}

func (s *Service) Start(notify bool) error {
	l := logger.Sugar()
	l.Debugw("[%s.Start]", "meta", s, "notify", notify)
	ui.NotifyNormal("[VPN]", fmt.Sprintf("Starting `%s`...", s.Name))
	switch s.Type {
	case "ovpn":
		return startOVPN(s.Name, s.Device, s.UpCommand, ovpnAttemptsMax, notify)
	case "ipsec":
		return startIPSec(s.Name, s.UpCommand, notify)
	}
	return nil
}

func (s *Service) Stop(notify bool) error {
	l := logger.Sugar()
	l.Debugw("[%s.Stop]", "meta", s, "notify", notify)
	ui.NotifyNormal("[VPN]", fmt.Sprintf("Stopping `%s`...", s.Name))
	switch s.Type {
	case "ovpn":
		return stopOVPN(s.Name, s.Device, s.DownCommand, ovpnAttemptsMax, notify)
	case "ipsec":
		return stopIPSec(s.Name, s.DownCommand, notify)
	}
	return nil
}
