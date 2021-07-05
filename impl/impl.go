package impl

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"go.uber.org/zap"
)

const (
	EnvPrefix       = "TB"
	DEBUG_FLAG_NAME = "DEBUG_MODE" // NOTE: not only for `toolbox`
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
