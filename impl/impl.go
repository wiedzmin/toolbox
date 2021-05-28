package impl

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"syscall"
	"time"

	"github.com/wiedzmin/toolbox/impl/tberrors"
)

const (
	EnvPrefix = "TB"
)

func CommonNowTimestamp() string {
	now := time.Now()
	return fmt.Sprintf("%02d-%02d-%d-%02d-%02d-%02d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
}

func FilesOlderThan(path, olderThan string, fullPath bool, regexWhitelist *string) ([]string, error) {
	files, err := ioutil.ReadDir(fmt.Sprintf("%s/.", path))
	if err != nil {
		return nil, err
	}
	d, err := time.ParseDuration(olderThan)
	if err != nil {
		return nil, err
	}

	var regexWhitelistRe *regexp.Regexp
	if regexWhitelist != nil {
		regexWhitelistRe = regexp.MustCompile(*regexWhitelist)
	}

	pastTime := time.Now().Add(-d)
	var result []string
	for _, fi := range files {
		stat := fi.Sys().(*syscall.Stat_t)
		ctime := time.Unix(int64(stat.Ctim.Sec), int64(stat.Ctim.Nsec))
		if ctime.Before(pastTime) && !fi.IsDir() {
			if regexWhitelist != nil && !regexWhitelistRe.MatchString(fi.Name()) {
				continue
			}
			if fullPath {
				result = append(result, fmt.Sprintf("%s/%s", path, fi.Name()))
			} else {
				result = append(result, fi.Name())
			}
		}
	}
	return result, nil
}

func RotateOlderThan(path, olderThan string, regexWhitelist *string) error {
	files, err := FilesOlderThan(path, olderThan, true, regexWhitelist)
	if err != nil {
		return err
	}
	for _, f := range files {
		err := os.Remove(f)
		if err != nil {
			return err
		}
	}
	return nil
}

func CollectFiles(path string, fullPath bool) ([]string, error) {
	files, err := ioutil.ReadDir(fmt.Sprintf("%s/.", path))
	if err != nil {
		return nil, err
	}
	var result []string
	for _, fi := range files {
		if !fi.IsDir() {
			if fullPath {
				result = append(result, fmt.Sprintf("%s/%s", path, fi.Name()))
			} else {
				result = append(result, fi.Name())
			}
		}
	}
	return result, nil
}

func CollectFilesRecursive(path string, fullPath bool, regexpsWhitelist []string) ([]string, error) {
	var result []string
	var regexpsWhitelistRe []regexp.Regexp
	for _, re := range regexpsWhitelist {
		rc := regexp.MustCompile(re)
		regexpsWhitelistRe = append(regexpsWhitelistRe, *rc)
	}
	err := filepath.Walk(path,
		func(pathentry string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				match := false
				for _, rc := range regexpsWhitelistRe {
					if rc.MatchString(info.Name()) {
						match = true
						break
					}
				}
				if match {
					result = append(result, pathentry)
				}
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func SendToUnixSocket(socket, data string) error {
	if _, err := os.Stat(socket); os.IsNotExist(err) {
		return tberrors.ErrNotExist{}
	}
	c, err := net.Dial("unix", socket)
	defer c.Close()
	if err != nil {
		return err
	}
	_, err = c.Write([]byte(fmt.Sprintf("%s\n", data)))
	return err
}
