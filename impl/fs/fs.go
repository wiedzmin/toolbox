package fs

import (
	"fmt"
	"io"
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
	l.Debugw("[FilesOlderThan]", "path", path, "olderThan", olderThan, "fullPath", fullPath, "regexWhitelist", regexWhitelist)
	files, err := os.ReadDir(fmt.Sprintf("%s/.", path))
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
		fiInfo, err := fi.Info()
		if err != nil {
			return nil, err
		}
		stat := fiInfo.Sys().(*syscall.Stat_t)
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

func CollectFiles(path string, fullPath bool, allowDirs bool, regexpsWhitelist, regexpsBlacklist []string) ([]string, error) {
	l := logger.Sugar()
	l.Debugw("[CollectFiles]", "path", path, "fullPath", fullPath, "regexpsWhitelist", regexpsWhitelist, "regexpsBlacklist", regexpsBlacklist)
	files, err := os.ReadDir(fmt.Sprintf("%s/.", path))
	l.Debugw("[CollectFiles]", "files", files)
	if err != nil {
		return nil, err
	}
	var result []string
	var regexpsWhitelistRe, regexpsBlacklistRe []regexp.Regexp
	acceptAll := false

	if regexpsWhitelist != nil && regexpsBlacklist != nil {
		return nil, fmt.Errorf("it makes no sense to provide both blacklist and whitelist")
	}
	if regexpsWhitelist != nil {
		for _, re := range regexpsWhitelist {
			rc := regexp.MustCompile(re)
			regexpsWhitelistRe = append(regexpsWhitelistRe, *rc)
		}
	}
	if regexpsBlacklist != nil {
		for _, re := range regexpsBlacklist {
			rc := regexp.MustCompile(re)
			regexpsBlacklistRe = append(regexpsBlacklistRe, *rc)
		}
	}
	if regexpsWhitelist == nil && regexpsBlacklist == nil {
		acceptAll = true
	}

	if acceptAll {
		for _, fi := range files {
			if !fi.IsDir() || allowDirs {
				if fullPath {
					result = append(result, fmt.Sprintf("%s/%s", path, fi.Name()))
				} else {
					result = append(result, fi.Name())
				}
			}
		}
	} else if regexpsWhitelist != nil {
		for _, fi := range files {
			if !fi.IsDir() || allowDirs {
				if impl.MatchAnyRegexp(fi.Name(), regexpsWhitelistRe) {
					if fullPath {
						result = append(result, fmt.Sprintf("%s/%s", path, fi.Name()))
					} else {
						result = append(result, fi.Name())
					}
				}
			}
		}
	} else if regexpsBlacklist != nil {
		for _, fi := range files {
			if !fi.IsDir() || allowDirs {
				if !impl.MatchAnyRegexp(fi.Name(), regexpsBlacklistRe) {
					if fullPath {
						result = append(result, fmt.Sprintf("%s/%s", path, fi.Name()))
					} else {
						result = append(result, fi.Name())
					}
				}
			}
		}
	}

	return result, nil
}

func CollectFilesRecursive(path string, allowDirs bool, regexpsWhitelist, regexpsBlacklist []string, trimPrefix bool) ([]string, error) {
	l := logger.Sugar()
	l.Debugw("[CollectFilesRecursive]", "path", path, "regexpsWhitelist", regexpsWhitelist, "regexpsBlacklist", regexpsBlacklist)

	var result []string
	var regexpsWhitelistRe, regexpsBlacklistRe []regexp.Regexp
	acceptAll := false
	if regexpsWhitelist != nil && regexpsBlacklist != nil {
		return nil, fmt.Errorf("it makes no sense to provide both blacklist and whitelist")
	}
	if regexpsWhitelist != nil {
		for _, re := range regexpsWhitelist {
			rc := regexp.MustCompile(re)
			regexpsWhitelistRe = append(regexpsWhitelistRe, *rc)
		}
	}
	if regexpsBlacklist != nil {
		for _, re := range regexpsBlacklist {
			rc := regexp.MustCompile(re)
			regexpsBlacklistRe = append(regexpsBlacklistRe, *rc)
		}
	}
	if regexpsWhitelist == nil && regexpsBlacklist == nil {
		acceptAll = true
	}

	err := filepath.Walk(path,
		func(pathentry string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !fi.IsDir() || allowDirs {
				if acceptAll {
					if trimPrefix {
						pathentry = strings.TrimPrefix(pathentry, path+"/")
					}
					result = append(result, pathentry)
				} else if regexpsWhitelist != nil && impl.MatchAnyRegexp(fi.Name(), regexpsWhitelistRe) {
					if trimPrefix {
						pathentry = strings.TrimPrefix(pathentry, path+"/")
					}
					result = append(result, pathentry)
				} else if regexpsBlacklist != nil && !impl.MatchAnyRegexp(fi.Name(), regexpsBlacklistRe) {
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

// FileExists checks if path exists and is really a file
func FileExists(path string) bool {
	l := logger.Sugar()
	fi, err := os.Stat(path)
	if err != nil {
		l.Debugw("[FileExists]", "desc", "error occured, assuming file not exist", "err", err)
		return false
	}
	if fi.IsDir() {
		return false
	}
	return true
}

func AtHomedir(suffix string) (*string, error) {
	userInfo, err := impl.FetchUserinfo()
	if err != nil {
		return nil, err
	}
	result := fmt.Sprintf("%s/%s", userInfo.HomeDir, strings.TrimPrefix(suffix, "/"))
	return &result, nil
}

func AtRunUser(suffix string) (*string, error) {
	userInfo, err := impl.FetchUserinfo()
	if err != nil {
		return nil, err
	}
	result := fmt.Sprintf("/run/user/%s/%s", userInfo.Uid, strings.TrimPrefix(suffix, "/"))
	return &result, nil
}

func AtDotConfig(suffix string) (*string, error) {
	userInfo, err := impl.FetchUserinfo()
	if err != nil {
		return nil, err
	}
	result := fmt.Sprintf("%s/.config/%s", userInfo.HomeDir, strings.TrimPrefix(suffix, "/"))
	return &result, nil
}
