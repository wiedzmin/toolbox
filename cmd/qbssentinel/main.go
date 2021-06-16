package main

import "fmt"

// links
// https://github.com/shirou/gopsutil
// https://github.com/mackerelio/go-osstat
// https://github.com/c9s/goprocinfo
// https://bitbucket.org/bertimus9/systemstat/src/master/

// idea
// systemd service (oneshot) that periodically checks or being pushed with system
// load information. If it goes above some threshold, we start to dump sessions
// with some another period. when load goes normal, dumping should stop.

func main() {
	fmt.Println("not implemented")
}
