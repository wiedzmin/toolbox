package impl

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	EnvPrefix            = "TB"
	DEBUG_FLAG_NAME      = "DEBUG_MODE" // NOTE: not only for `toolbox`
	SelectorFontFlagName = "selector-font"
	SelectorToolFlagName = "selector-tool"
)

var (
	logger *zap.Logger
)

func init() {
	logger = NewLogger()
}

type ErrInvalidUrl struct {
	Content string
}

func (e ErrInvalidUrl) Error() string {
	return "invalid url found"
}

type FileErrNotExist struct {
	Path string
}

func (e FileErrNotExist) Error() string {
	return fmt.Sprintf("file/dir '%s' does not exist", e.Path)
}

type FileFormatError struct {
	Content string
}

func (e FileFormatError) Error() string {
	return fmt.Sprintf("file format error: %s", e.Content)
}

type ErrNotImplemented struct {
	Token string
}

func (e ErrNotImplemented) Error() string {
	return fmt.Sprintf("'%s' not implemented", e.Token)
}

func HasSpaces(s string) bool {
	hasSpaces := false
	for c := range s {
		if c == ' ' {
			hasSpaces = true
			break
		}
	}
	return hasSpaces
}

func Quote(s string) string {
	return fmt.Sprintf("\"%s\"", s)
}

func FetchUserinfo() (*user.User, error) {
	l := logger.Sugar()
	userInfo, err := user.Current()
	if err != nil {
		l.Warnw("[fetchUserinfo]", "err", err)
		return nil, err
	}
	if userInfo.HomeDir == "" || userInfo.Uid == "" {
		err := errors.New("insufficient userinfo")
		l.Warnw("[fetchUserinfo]", "err", err, "userInfo", userInfo)
		return nil, err
	}
	l.Debugw("[fetchUserinfo]", "userInfo", userInfo)
	return userInfo, nil
}

func CommonNowTimestamp() string {
	now := time.Now()
	return fmt.Sprintf("%02d-%02d-%02d-%02d-%02d-%02d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
}

func SendToUnixSocket(socket string, data []byte) error {
	if _, err := os.Stat(socket); os.IsNotExist(err) {
		return FileErrNotExist{}
	}
	c, err := net.Dial("unix", socket)
	if err != nil {
		return err
	}
	defer c.Close()
	data = append(data, '\n')
	_, err = c.Write(data)
	return err
}

func NewLogger() *zap.Logger {
	config := zap.NewDevelopmentConfig()
	_, exists := os.LookupEnv(DEBUG_FLAG_NAME)
	if exists {
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	}
	logger, _ := config.Build()
	return logger
}

func EnsureBinary(name string, logger zap.Logger) {
	l := logger.Sugar()
	path, err := exec.LookPath(name)
	if err != nil {
		l.Warnw(fmt.Sprintf("[EnsureBinary] %s not found", name))
		os.Exit(1)
	}
	l.Debugw("[EnsureBinary]", name, path)
}

func GetSHA1(text string) string {
	h := sha1.New()
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))
}

func MapToText(m map[string]string, delimiter string) string {
	var lines []string
	for key, value := range m {
		lines = append(lines, fmt.Sprintf("%s%s%s", key, delimiter, value))
	}
	return strings.Join(lines, "\n")
}

func ShorterString(s string, l int) string {
	if len(s) > l {
		return fmt.Sprintf("%s...", strings.TrimRight(s[:l], " "))
	} else {
		return strings.TrimRight(s, " ")
	}
}

func MatchAnyRegexp(s string, regexps []regexp.Regexp) bool {
	l := logger.Sugar()
	match := false
	l.Debugw("[MatchAnyRegexp]", "string", s)
	for _, rc := range regexps {
		l.Debugw("[MatchAnyRegexp]", "trying regexp", rc)
		if rc.MatchString(s) {
			l.Debugw("[MatchAnyRegexp]", "matched", s, "regexp", rc)
			match = true
			break
		}
	}
	return match
}

// callerName is a helper function for automatically getting function name for the current call frame
// "skip" parameter denotes, how many frames up the call stack should be skipped
func callerName(skip int) string {
	const unknown = "unknown"
	pcs := make([]uintptr, 1)
	n := runtime.Callers(skip+2, pcs)
	if n < 1 {
		return unknown
	}
	frame, _ := runtime.CallersFrames(pcs).Next()
	if frame.Function == "" {
		return unknown
	}
	return frame.Function
}

// FuncDuration measures whole function execution time for debugging purposes
// `defer impl.FuncDuration("<some function ID>")()` should be the first statement in function being measured
// if aforementioned function ID is "", actual function name will be guessed from runtime metadata
func FuncDuration(id string) func() {
	l := logger.Sugar()
	functionId := id
	if functionId == "" {
		functionId = callerName(1)
	}
	start := time.Now()
	return func() {
		l.Debugw("[FuncDuration]", "function", functionId, "duration", time.Since(start))
	}
}
