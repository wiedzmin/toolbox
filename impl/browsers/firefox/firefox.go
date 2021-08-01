package firefox

import (
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

func RawSessionsPath() *string {
	path, err := impl.AtHomedir(".mozilla/firefox/profile.default/sessionstore-backups")
	if err != nil {
		return nil
	}
	return path
}
