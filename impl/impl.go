package impl

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"syscall"
	"time"
)

const (
	EnvPrefix = "TB"
)

type ErrInvalidUrl struct {
	Content string
}

func (e ErrInvalidUrl) Error() string {
	return "invalid url found"
}

type FileErrNotExist struct{}

func (e FileErrNotExist) Error() string {
	return "file/dir does not exist"
}

type ErrNotImplemented struct {
	Token string
}

func (e ErrNotImplemented) Error() string {
	return fmt.Sprintf("'%s' not implemented", e.Token)
}

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

func CollectFiles(path string, fullPath bool, regexpsWhitelist []string) ([]string, error) {
	files, err := ioutil.ReadDir(fmt.Sprintf("%s/.", path))
	if err != nil {
		return nil, err
	}
	var result []string
	var regexpsWhitelistRe []regexp.Regexp
	for _, re := range regexpsWhitelist {
		rc := regexp.MustCompile(re)
		regexpsWhitelistRe = append(regexpsWhitelistRe, *rc)
	}
	for _, fi := range files {
		if !fi.IsDir() {
			match := false
			for _, rc := range regexpsWhitelistRe {
				if rc.MatchString(fi.Name()) {
					match = true
					break
				}
			}
			if match {
				if fullPath {
					result = append(result, fmt.Sprintf("%s/%s", path, fi.Name()))
				} else {
					result = append(result, fi.Name())
				}
			}
		}
	}
	return result, nil
}

func CollectFilesRecursive(path string, regexpsWhitelist []string) ([]string, error) {
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
		return FileErrNotExist{}
	}
	c, err := net.Dial("unix", socket)
	defer c.Close()
	if err != nil {
		return err
	}
	_, err = c.Write([]byte(fmt.Sprintf("%s\n", data)))
	return err
}

func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

func CopyFile(src, dst string) error {
	sfi, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !sfi.Mode().IsRegular() {
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return err
		}
	}
	if err = os.Link(src, dst); err == nil {
		return err
	}
	return copyFileContents(src, dst)
}
