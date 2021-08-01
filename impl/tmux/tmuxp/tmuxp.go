package tmuxp

import (
	"fmt"
	"strings"

	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/fs"
	"github.com/wiedzmin/toolbox/impl/shell"
	"go.uber.org/zap"
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

var logger *zap.Logger

func init() {
	logger = impl.NewLogger()
}

func SessionsRootDefault() *string {
	path, err := impl.AtHomedir(".tmuxp")
	if err != nil {
		return nil
	}
	return path
}

func CollectSessions(root string) ([]Session, error) {
	l := logger.Sugar()
	sessionFiles, err := fs.CollectFiles(root, false, []string{SESSION_FILE_SUFFIX})
	if err != nil {
		return nil, err
	}
	var result []Session
	for _, f := range sessionFiles {
		l.Debugw("[CollectSessions]", "session", f)
		result = append(result, Session{
			Name: strings.Split(f, ".")[0],
			Path: fmt.Sprintf("%s/%s", root, f),
		})
	}
	return result, nil
}

func GetSession(root, name string) (*Session, error) {
	l := logger.Sugar()
	regexp := name
	if !strings.HasSuffix(name, SESSION_FILE_SUFFIX) {
		regexp = fmt.Sprintf("%s/%s", name, SESSION_FILE_SUFFIX)
	}
	l.Debugw("[GetSession]", "root", root, "name", name, "regexp", regexp)
	sessionFiles, err := fs.CollectFiles(root, false, []string{regexp})
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
	l := logger.Sugar()
	cmd := fmt.Sprintf("tmuxp load -y -d %s", s.Path)
	if attach {
		cmd = fmt.Sprintf("tmuxp load -y %s", s.Path)
	}
	l.Debugw(fmt.Sprintf("[%s.GetSession]", s.Name), "cmd", cmd)
	_, err := shell.ShellCmd(cmd, nil, nil, false, false)
	return err
}
