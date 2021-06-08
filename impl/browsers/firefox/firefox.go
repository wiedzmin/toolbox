package firefox

import (
	"errors"
	"fmt"
	"os/user"
)

type SessionFormat int8

const (
	SESSION_FORMAT_JSON SessionFormat = 0
	SESSION_FORMAT_ORG  SessionFormat = 1
)

func RawSessionsPath() (*string, error) {
	userInfo, err := user.Current()
	if err != nil {
		return nil, err
	}
	if userInfo.HomeDir == "" {
		return nil, errors.New("current user has no home directory")
	}
	result := fmt.Sprintf("%s/.mozilla/firefox/profile.default/sessionstore-backups", userInfo.HomeDir)
	return &result, nil
}
