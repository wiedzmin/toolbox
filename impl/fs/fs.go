package fs

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/wiedzmin/toolbox/impl"
	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	logger = impl.NewLogger()
}

func FilesOlderThan(path, olderThan string, fullPath bool, regexWhitelist *string) ([]string, error) {
	l := logger.Sugar()
	l.Debugw("[FilesOlderThan]", "path", path, "olderThan", olderThan, "fullPath", fullPath, "regexWhitelist", *regexWhitelist)
	files, err := ioutil.ReadDir(fmt.Sprintf("%s/.", path))
	l.Debugw("[FilesOlderThan]", "files", files)
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

	l.Debugw("[FilesOlderThan]", "result", result)
	return result, nil
}

func RotateOlderThan(path, olderThan string, regexWhitelist *string) error {
	l := logger.Sugar()
	files, err := FilesOlderThan(path, olderThan, true, regexWhitelist)
	if err != nil {
		return err
	}
	for _, f := range files {
		err := os.Remove(f)
		if err != nil {
			return err
		}
		l.Debugw("[RotateOlderThan]", "removed", f)
	}
	return nil
}

func CollectFiles(path string, fullPath bool, regexpsWhitelist []string) ([]string, error) {
	l := logger.Sugar()
	l.Debugw("[CollectFiles]", "path", path, "fullPath", fullPath, "regexpsWhitelist", regexpsWhitelist)
	files, err := ioutil.ReadDir(fmt.Sprintf("%s/.", path))
	l.Debugw("[CollectFiles]", "files", files)
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
					l.Debugw("[CollectFiles]", "matched", fi.Name(), "regexp", rc)
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

func CollectFilesRecursive(path string, regexpsWhitelist []string, trimPrefix bool) ([]string, error) {
	var result []string
	var regexpsWhitelistRe []regexp.Regexp
	var acceptAll bool
	l := logger.Sugar()
	l.Debugw("[CollectFilesRecursive]", "path", path, "regexpsWhitelist", regexpsWhitelist)
	if regexpsWhitelist != nil {
		for _, re := range regexpsWhitelist {
			rc := regexp.MustCompile(re)
			regexpsWhitelistRe = append(regexpsWhitelistRe, *rc)
		}
	} else {
		acceptAll = true
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
						l.Debugw("[CollectFiles]", "matched", info.Name(), "rc", rc)
						match = true
						break
					}
				}
				if match || acceptAll {
					if trimPrefix {
						pathentry = strings.TrimPrefix(pathentry, path+"/")
					}
					result = append(result, pathentry)
				}
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	l.Debugw("[CollectFilesRecursive]", "result", result)
	return result, nil
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
