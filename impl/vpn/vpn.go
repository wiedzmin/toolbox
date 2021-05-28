package vpn

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mediocregopher/radix/v3"
	"github.com/wiedzmin/toolbox/impl/env"
	"github.com/wiedzmin/toolbox/impl/json"
	"github.com/wiedzmin/toolbox/impl/shell"
	"github.com/wiedzmin/toolbox/impl/systemd"
	"github.com/wiedzmin/toolbox/impl/ui"
)

const (
	ipV4StatusPath       = "/proc/sys/net/ipv4/conf/"
	ovpnAttemptsMax      = 15
	ovpnAttemptsInfinite = -1
)

var nmVpnActiveStatusCodes = []string{"3", "5"}

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

func parseVpnMeta(data []byte) (map[string]map[string]string, error) {
	result := make(map[string]map[string]string)
	vpnMap, err := json.GetMapByPath(data, "")
	if err != nil {
		return nil, err
	}
	for key, value := range vpnMap {
		valueStrMap, err := json.StringifyFlatMap(value)
		if err != nil {
			return nil, err
		}
		result[key] = valueStrMap
	}
	return result, nil
}

func GetMetadata(client *radix.Pool) (map[string]map[string]string, error) {
	vpnData, _, err := env.GetRedisValue("net/vpn_meta", client)
	if err != nil {
		return nil, err
	}
	vpnMeta, err := parseVpnMeta(vpnData)
	if err != nil {
		return nil, err
	}
	return vpnMeta, nil
}

func StopRunning(omit []string, vpnMeta map[string]map[string]string, notify bool) error {
	devdns := systemd.Service{Name: "docker-devdns.service"}
	err := devdns.Stop()
	if err != nil {
		return err
	}
	omitStr := strings.Join(omit, ",")
	for name, meta := range vpnMeta {
		if omitStr != "" && strings.Contains(name, omitStr) {
			continue
		}
		ui.NotifyNormal("[VPN]", fmt.Sprintf("Stopping `%s`...", name))
		StopService(name, meta, notify)
	}
	return nil
}

func StartOVPN(name, device, cmd string, attempts int, notify bool) error {
	tun_path := fmt.Sprintf("%s%s", ipV4StatusPath, device)
	if _, err := os.Stat(tun_path); !os.IsNotExist(err) {
		env.SetRedisValue(fmt.Sprintf("vpn/%s/is_up", name), "yes", nil)
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
			env.SetRedisValue(fmt.Sprintf("vpn/%s/is_up", name), "yes", nil)
			if notify {
				ui.NotifyNormal("[VPN]", fmt.Sprintf("Started `%s` service", name))
			}
			return nil
		} else {
			env.SetRedisValue(fmt.Sprintf("vpn/%s/is_up", name), "unk", nil)
			if notify {
				ui.NotifyCritical("[VPN]", fmt.Sprintf("Error starting `%s` service:\n\n%s", name, err.Error()))
			}
			return err
		}
	}
}

func StartIPSec(name, cmd string, notify bool) error {
	up, err := nmIpsecVpnUp(name)
	if err != nil {
		return err
	}
	if up {
		env.SetRedisValue(fmt.Sprintf("vpn/%s/is_up", name), "yes", nil)
		if notify {
			ui.NotifyNormal("[VPN]", fmt.Sprintf("`%s` is up", name))
		}
		return nil
	} else {
		result, err := shell.ShellCmd(cmd, nil, []string{"LANGUAGE=en_US.en"}, true, true)
		if err != nil {
			if strings.Contains(*result, "is already active") {
				env.SetRedisValue(fmt.Sprintf("vpn/%s/is_up", name), "yes", nil)
				if notify {
					ui.NotifyNormal("[VPN]", fmt.Sprintf("`%s` is up", name))
				}
				return nil
			} else {
				env.SetRedisValue(fmt.Sprintf("vpn/%s/is_up", name), "unk", nil)
				if notify {
					ui.NotifyCritical("[VPN]", fmt.Sprintf("Error starting `%s` service:\n\n%s", name, err.Error()))
				}
				return err
			}
		} else {
			env.SetRedisValue(fmt.Sprintf("vpn/%s/is_up", name), "yes", nil)
			if notify {
				ui.NotifyNormal("[VPN]", fmt.Sprintf("`%s` is up", name))
			}
			return nil
		}
	}
}

func StartService(name string, meta map[string]string, notify bool) error {
	t, ok := meta["type"]
	if !ok {
		return fmt.Errorf("failed to get vpn type")
	}
	cmd, ok := meta["up"]
	if !ok {
		return fmt.Errorf("failed to get `up` command for `%s`", name)
	}
	switch t {
	case "ovpn":
		device, ok := meta["dev"]
		if !ok {
			return fmt.Errorf("failed to get OpenVPN device")
		}
		return StartOVPN(name, device, cmd, ovpnAttemptsMax, notify)
	case "ipsec":
		return StartIPSec(name, cmd, notify)
	}
	return nil
}

func StopOVPN(name, device, cmd string, attempts int, notify bool) error {
	tun_path := fmt.Sprintf("%s%s", ipV4StatusPath, device)
	if _, err := os.Stat(tun_path); !os.IsNotExist(err) {
		env.SetRedisValue(fmt.Sprintf("vpn/%s/is_up", name), "no", nil)
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
			env.SetRedisValue(fmt.Sprintf("vpn/%s/is_up", name), "no", nil)
			if notify {
				ui.NotifyNormal("[VPN]", fmt.Sprintf("Stopped `%s` service", name))
			}
			return nil
		} else {
			env.SetRedisValue(fmt.Sprintf("vpn/%s/is_up", name), "unk", nil)
			if notify {
				ui.NotifyCritical("[VPN]", fmt.Sprintf("Error stopping `%s` service:\n\n%s", name, err.Error()))
			}
			return err
		}
	}
}

func StopIPSec(name, cmd string, notify bool) error {
	up, err := nmIpsecVpnUp(name)
	if err != nil {
		return err
	}
	if !up {
		env.SetRedisValue(fmt.Sprintf("vpn/%s/is_up", name), "no", nil)
		if notify {
			ui.NotifyNormal("[VPN]", fmt.Sprintf("`%s` is down", name))
		}
		return nil
	} else {
		result, err := shell.ShellCmd(cmd, nil, []string{"LANGUAGE=en_US.en"}, true, true)
		if err != nil {
			if strings.Contains(*result, "not an active") {
				env.SetRedisValue(fmt.Sprintf("vpn/%s/is_up", name), "no", nil)
				if notify {
					ui.NotifyNormal("[VPN]", fmt.Sprintf("`%s` is down", name))
				}
				return nil
			} else {
				env.SetRedisValue(fmt.Sprintf("vpn/%s/is_up", name), "unk", nil)
				if notify {
					ui.NotifyCritical("[VPN]", fmt.Sprintf("Error stopping `%s` service:\n\n%s", name, err.Error()))
				}
				return err
			}
		} else {
			env.SetRedisValue(fmt.Sprintf("vpn/%s/is_up", name), "no", nil)
			if notify {
				ui.NotifyNormal("[VPN]", fmt.Sprintf("`%s` is down", name))
			}
			return nil
		}
	}
}

func StopService(name string, meta map[string]string, notify bool) error {
	ui.NotifyNormal("[VPN]", fmt.Sprintf("Stopping `%s`...", name))
	t, ok := meta["type"]
	if !ok {
		return fmt.Errorf("failed to get vpn type")
	}
	cmd, ok := meta["down"]
	if !ok {
		return fmt.Errorf("failed to get `down` command for `%s`", name)
	}
	switch t {
	case "ovpn":
		device, ok := meta["dev"]
		if !ok {
			return fmt.Errorf("failed to get OpenVPN device")
		}
		return StopOVPN(name, device, cmd, ovpnAttemptsMax, notify)
	case "ipsec":
		return StopIPSec(name, cmd, notify)
	}
	return nil
}