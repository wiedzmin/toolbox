package impl

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
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

func FileStats(name string) (atime, mtime, ctime time.Time, isDir bool, err error) {
	fi, err := os.Stat(name)
	if err != nil {
		return
	}
	mtime = fi.ModTime()
	stat := fi.Sys().(*syscall.Stat_t)
	atime = time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
	ctime = time.Unix(int64(stat.Ctim.Sec), int64(stat.Ctim.Nsec))
	isDir = fi.IsDir()
	return
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
	for _, f := range files {
		_, _, ctime, isDir, err := FileStats(fmt.Sprintf("%s/%s", path, f.Name()))
		if err != nil {
			return nil, err
		}
		if ctime.Before(pastTime) && !isDir {
			if regexWhitelist != nil && !regexWhitelistRe.MatchString(f.Name()) {
				continue
			}
			if fullPath {
				result = append(result, fmt.Sprintf("%s/%s", path, f.Name()))
			} else {
				result = append(result, f.Name())
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
	for _, f := range files {
		_, _, _, isDir, err := FileStats(fmt.Sprintf("%s/%s", path, f.Name()))
		if err != nil {
			return nil, err
		}
		if !isDir {
			if fullPath {
				result = append(result, fmt.Sprintf("%s/%s", path, f.Name()))
			} else {
				result = append(result, f.Name())
			}
		}
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
