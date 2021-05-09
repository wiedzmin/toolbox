package main

import (
	"fmt"
	"strings"

	"github.com/wiedzmin/toolbox/impl/env"
	"github.com/wiedzmin/toolbox/impl/ui"
)

func main() {
	var result []string
	statuses, _, err := env.GetRedisValuesFuzzy("vpn/*/is_up", nil)
	if err == nil {
		for key, value := range statuses {
			result = append(result, fmt.Sprintf("%s: %s", strings.Split(key, "/")[1], string(value)))
		}
		ui.NotifyNormal("[VPN] statuses", strings.Join(result, "\n"))
	} else {
		ui.NotifyCritical("[VPN]", "Failed to get vpn statuses")
	}
}
