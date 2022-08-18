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

func AtHomedir(suffix string) (*string, error) {
	l := logger.Sugar()
	userInfo, err := user.Current()
	l.Debugw("[AtHomedir]", "userInfo", userInfo)
	if err != nil {
		l.Warnw("[AtHomedir]", "err", err)
		return nil, err
	}
	if userInfo.HomeDir == "" {
		err := errors.New("current user has no home directory")
		l.Warnw("[AtHomedir]", "err", err)
		return nil, err
	}
	result := fmt.Sprintf("%s/%s", userInfo.HomeDir, strings.TrimPrefix(suffix, "/"))
	l.Debugw("[AtHomedir]", "result", result)
	return &result, nil
}

func AtRunUser(suffix string) (*string, error) {
	l := logger.Sugar()
	userInfo, err := user.Current()
	l.Debugw("[AtRunUser]", "userInfo", userInfo)
	if err != nil {
		l.Warnw("[AtRunUser]", "err", err)
		return nil, err
	}
	if userInfo.Uid == "" {
		err := errors.New("current user has empty UID")
		l.Warnw("[AtRunUser]", "err", err)
		return nil, err
	}
	result := fmt.Sprintf("/run/user/%s/%s", userInfo.Uid, strings.TrimPrefix(suffix, "/"))
	l.Debugw("[AtRunUser]", "result", result)
	return &result, nil
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
	defer c.Close()
	if err != nil {
		return err
	}
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
