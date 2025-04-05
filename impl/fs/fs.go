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

type FSCollection struct {
	path                                   string
	regexpsWhitelistRe, regexpsBlacklistRe []regexp.Regexp
	acceptAll                              bool
	allowDirs                              bool
}

// NewFSCollection creates FS path representation object that could be further queried in various applicable ways
// Note that currently there is no black/white-listing cross logic implemented
func NewFSCollection(path string, regexpsWhitelist, regexpsBlacklist []string, allowDirs bool) *FSCollection {
	l := logger.Sugar()
	l.Debugw("[NewFSCollection]", "path", path, "regexpsWhitelist", regexpsWhitelist, "regexpsBlacklist", regexpsBlacklist, "allowDirs", allowDirs)
	if regexpsWhitelist != nil && regexpsBlacklist != nil {
		l.Debugw("[FSCollection.Emit]", "error", "Note that currently there is no black/white-listing cross logic implemented")
		return nil
	}

	var result FSCollection
	result.path = path
	result.allowDirs = allowDirs
	for _, re := range regexpsWhitelist {
		rc := regexp.MustCompile(re)
		result.regexpsWhitelistRe = append(result.regexpsWhitelistRe, *rc)
	}
	for _, re := range regexpsBlacklist {
		rc := regexp.MustCompile(re)
		result.regexpsBlacklistRe = append(result.regexpsBlacklistRe, *rc)
	}
	if regexpsWhitelist == nil && regexpsBlacklist == nil {
		result.acceptAll = true
	}
	l.Debugw("[NewFSCollection]", "result.path", result.path, "result.regexpsWhitelistRe", result.regexpsWhitelistRe, "result.regexpsBlacklistRe", result.regexpsBlacklistRe, "result.acceptAll", result.acceptAll)
	return &result
}

// Emit generates list of files according to previously initialized FSCollection,
// either with absolute path oor basename only
func (fsc *FSCollection) Emit(absolutePath bool) []string {
	l := logger.Sugar()
	files, err := os.ReadDir(fmt.Sprintf("%s/.", fsc.path))
	l.Debugw("[FSCollection.Emit]", "files", files)
	if err != nil {
		l.Debugw("[FSCollection.Emit]", "err", err)
		return nil
	}
	var result []string

	for _, fi := range files {
		l.Debugw("[FSCollection.Emit]", "fi.IsDir", fi.IsDir(), "fi.Name", fi.Name())
		if !fi.IsDir() || fsc.allowDirs {
			if fsc.acceptAll ||
				fsc.regexpsWhitelistRe != nil && impl.MatchAnyRegexp(fi.Name(), fsc.regexpsWhitelistRe) ||
				fsc.regexpsBlacklistRe != nil && !impl.MatchAnyRegexp(fi.Name(), fsc.regexpsBlacklistRe) {
				if absolutePath {
					result = append(result, fmt.Sprintf("%s/%s", fsc.path, fi.Name()))
				} else {
					result = append(result, fi.Name())
				}
			}
		}
	}
	return result
}

// EmitRecursive recursively generates list of files according to previously initialized FSCollection,
// either with absolute path oor basename only
func (fsc *FSCollection) EmitRecursive(absolutePath bool) []string {
	l := logger.Sugar()

	var result []string

	err := filepath.Walk(fsc.path,
		func(pathentry string, fi os.FileInfo, err error) error {
			if err != nil {
				l.Debugw("[FSCollection.EmitRecursive/walker]", "err", err, "pathentry", pathentry)
				return err
			}
			if !fi.IsDir() || fsc.allowDirs {
				if fsc.acceptAll ||
					fsc.regexpsWhitelistRe != nil && impl.MatchAnyRegexp(fi.Name(), fsc.regexpsWhitelistRe) ||
					fsc.regexpsBlacklistRe != nil && !impl.MatchAnyRegexp(fi.Name(), fsc.regexpsBlacklistRe) {
					if !absolutePath {
						pathentry = strings.TrimPrefix(pathentry, fsc.path+"/")
					}
					result = append(result, pathentry)
				}
			}
			return nil
		})
	if err != nil {
		return nil
	}

	l.Debugw("[FSCollection.EmitRecursive]", "result", result)
	return result
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
		l.Debugw("[FileExists]", "desc", "error occurred, assuming file not exist", "err", err)
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
