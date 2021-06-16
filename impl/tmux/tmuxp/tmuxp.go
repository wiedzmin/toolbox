package tmuxp

import (
	"errors"
	"fmt"
	"os/user"
	"strings"

	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/shell"
)

const SESSION_FILE_SUFFIX = "yml"

type Session struct {
	Name string
	Path string
}

type ErrSessionNotFound struct {
	Name string
}

func (e ErrSessionNotFound) Error() string {
	return fmt.Sprintf("tmuxp: session '%s' not exist", e.Name)
}

func SessionsRootDefault() (*string, error) {
	userInfo, err := user.Current()
	if err != nil {
		return nil, err
	}
	if userInfo.HomeDir == "" {
		return nil, errors.New("current user has no home directory")
	}
	result := fmt.Sprintf("%s/.tmuxp", userInfo.HomeDir)
	return &result, nil
}

func CollectSessions(root string) ([]Session, error) {
	sessionFiles, err := impl.CollectFiles(root, false, []string{SESSION_FILE_SUFFIX})
	if err != nil {
		return nil, err
	}
	var result []Session
	for _, f := range sessionFiles {
		result = append(result, Session{
			Name: strings.Split(f, ".")[0],
			Path: fmt.Sprintf("%s/%s", root, f),
		})
	}
	return result, nil
}

func GetSession(root, name string) (*Session, error) {
	regexp := name
	if !strings.HasSuffix(name, SESSION_FILE_SUFFIX) {
		regexp = fmt.Sprintf("%s/%s", name, SESSION_FILE_SUFFIX)
	}
	sessionFiles, err := impl.CollectFiles(root, false, []string{regexp})
	if err != nil {
		return nil, err
	}
	if len(sessionFiles) == 0 {
		return nil, ErrSessionNotFound{Name: name}
	}
	return &Session{
		Name: name,
		Path: fmt.Sprintf("%s/%s", root, sessionFiles[0]),
	}, nil
}

func (s *Session) Load(attach bool) error {
	cmd := fmt.Sprintf("tmuxp load -y -d %s", s.Path)
	if attach {
		cmd = fmt.Sprintf("tmuxp load -y %s", s.Path)
	}
	_, err := shell.ShellCmd(cmd, nil, nil, false, false)
	return err
}
