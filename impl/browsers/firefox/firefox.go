package firefox

import (
	"errors"
	"fmt"
	"os/user"

	"github.com/wiedzmin/toolbox/impl"
	"go.uber.org/zap"
)

type SessionFormat int8

const (
	SESSION_FORMAT_JSON SessionFormat = 0
	SESSION_FORMAT_ORG  SessionFormat = 1
)

var logger *zap.Logger

func init() {
	logger = impl.NewLogger()
}

func RawSessionsPath() (*string, error) {
	l := logger.Sugar()
	userInfo, err := user.Current()
	l.Debugw("[RawSessionsPath]", "userInfo", userInfo)
	if err != nil {
		return nil, err
	}
	if userInfo.HomeDir == "" {
		return nil, errors.New("current user has no home directory")
	}
	result := fmt.Sprintf("%s/.mozilla/firefox/profile.default/sessionstore-backups", userInfo.HomeDir)
	l.Debugw("[RawSessionsPath]", "result", result)
	return &result, nil
}
