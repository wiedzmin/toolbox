package qutebrowser

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"os"
	"os/user"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/fs"
	"github.com/wiedzmin/toolbox/impl/ui"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type SessionFormat int8

type Request struct {
	Commands        []string `json:"args"`
	TargetArg       string   `json:"target_arg"`
	ProtocolVersion int      `json:"protocol_version"`
}

const (
	SESSION_FORMAT_YAML SessionFormat = iota
	SESSION_FORMAT_JSON
	SESSION_FORMAT_ORG
	SESSION_FORMAT_ORG_FLAT
	SessionstoreSubpath = ".local/share/qutebrowser/sessions"
	URL_TARGET_SETTING  = "new_instance_open_target"
	URL_TARGET_KEYNAME  = "qb_current_url_target"
)

var logger *zap.Logger

func init() {
	logger = impl.NewLogger()
}

type LastVisitedTS string

func (f LastVisitedTS) MarshalYAML() (interface{}, error) {
	if f != "" {
		return fmt.Sprintf("'%s'", f), nil
	}
	return nil, nil
}

type Pos struct {
	X int `yaml:"x"`
	Y int `yaml:"y"`
}
type Page struct {
	Active      bool          `yaml:"active"`
	LastVisited LastVisitedTS `yaml:"last_visited"`
	Pinned      bool          `yaml:"pinned"`
	ScrollPos   Pos           `yaml:"scroll_pos"`
	Title       string        `yaml:"title"`
	URL         string        `yaml:"url"`
	Zoom        float64       `yaml:"zoom"`
}

type Tab struct {
	Active  bool   `yaml:"active"`
	History []Page `yaml:"history"`
}

type Window struct {
	Geometry string `yaml:"geometry"`
	Tabs     []Tab  `yaml:"tabs"`
}

type SessionLayout struct {
	Windows []Window `yaml:"windows"`
}

func SocketPath() (*string, error) {
	l := logger.Sugar()
	userInfo, err := user.Current()
	l.Debugw("[SocketPath]", "userInfo", userInfo)
	if err != nil {
		return nil, err
	}
	result := fmt.Sprintf("/run/user/%s/qutebrowser/ipc-%x",
		userInfo.Uid, md5.Sum([]byte(userInfo.Username)))
	l.Debugw("[SocketPath]", "socket path", result)
	return &result, nil
}

func Execute(commands []string) error {
	l := logger.Sugar()
	req := Request{Commands: commands}
	l.Debugw("[qutebrowser.Execute]", "request", req)
	rb, err := req.Marshal()
	if err != nil {
		return err
	}
	socketPath, err := SocketPath()
	if err != nil {
		return err
	}

	err = impl.SendToUnixSocket(*socketPath, rb)
	if _, ok := err.(impl.FileErrNotExist); ok {
		msg := fmt.Sprintf("cannot access socket at `%s`\nIs qutebrowser running?", *socketPath)
		etraits := impl.GetEnvTraits()
		if etraits.InX {
			ui.NotifyCritical("[qutebrowser]", msg)
		}
		l.Debugw("[qutebrowser.Execute]", "err", msg)
		os.Exit(0)
	} else {
		return err
	}
	return nil
}

func RawSessionsPath() *string {
	path, err := fs.AtHomedir(SessionstoreSubpath)
	if err != nil {
		return nil
	}
	return path
}

func (r *Request) Marshal() ([]byte, error) {
	r.ProtocolVersion = 1
	r.TargetArg = ""
	bytes, err := jsoniter.Marshal(r)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func SaveSessionInternal(name string) error {
	sessionName := name
	if name == "" {
		sessionName = fmt.Sprintf("session-%s", impl.CommonNowTimestamp(false))
	}

	return Execute([]string{
		fmt.Sprintf(":session-save --quiet %s", sessionName),
		":session-save --quiet",
	})
}

func LoadSession(path string) (*SessionLayout, error) {
	l := logger.Sugar()
	l.Debugw("[LoadSession]", "path", path)
	var session SessionLayout
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func FixSession(data *SessionLayout) *SessionLayout {
	l := logger.Sugar()
	var result SessionLayout
	for _, w := range data.Windows {
		l.Debugw("[FixSession]", "window/before", w)
		var win Window
		win.Geometry = w.Geometry
		for _, t := range w.Tabs {
			l.Debugw("[FixSession]", "tab/before", t)
			var tab Tab
			tab.Active = t.Active
			for _, p := range t.History {
				l.Debugw("[FixSession]", "page", p)
				if strings.HasPrefix(p.Title, "Error loading") {
					continue
				}
				if strings.HasPrefix(p.URL, "data:text/html") {
					continue
				}
				tab.History = append(tab.History, p)
			}
			if len(tab.History) > 0 {
				tab.History[len(tab.History)-1].ScrollPos = Pos{0, 0}
				tab.History[len(tab.History)-1].Active = true
				tab.History[len(tab.History)-1].Zoom = 1.0
			}
			l.Debugw("[FixSession]", "window/after", tab)
			win.Tabs = append(win.Tabs, tab)
		}
		l.Debugw("[FixSession]", "window/after", win)
		result.Windows = append(result.Windows, win)
	}
	return &result
}

func DumpSession(path string, data *SessionLayout, format SessionFormat) error {
	l := logger.Sugar()
	l.Debugw("[DumpSession]", "path", path, "data", data, "format", format)
	if data == nil {
		return fmt.Errorf("empty session")
	}
	file, err := os.OpenFile(path, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	switch format {
	case SESSION_FORMAT_YAML:
		b, err := yaml.Marshal(data)
		if err != nil {
			return err
		}
		_, err = writer.Write(b)
		if err != nil {
			return err
		}
	case SESSION_FORMAT_ORG:
		index := 1
		var result []string
		for _, w := range data.Windows {
			result = append(result, (fmt.Sprintf("* window %d", index)))
			for _, t := range w.Tabs {
				for _, p := range t.History {
					if p.Active {
						l.Debugw("[DumpSession]", "url", p.URL)
						result = append(result, (fmt.Sprintf("** %s", p.URL)))
						break
					}
				}
			}
			index = index + 1
		}
		for _, line := range result {
			_, _ = writer.WriteString(fmt.Sprintf("%s\n", line))
		}
	case SESSION_FORMAT_ORG_FLAT:
		var result []string
		for _, w := range data.Windows {
			for _, t := range w.Tabs {
				for _, p := range t.History {
					if p.Active {
						l.Debugw("[DumpSession]", "url", p.URL)
						result = append(result, (fmt.Sprintf("* %s", p.URL)))
						break
					}
				}
			}
		}
		for _, line := range result {
			_, _ = writer.WriteString(fmt.Sprintf("%s\n", line))
		}
	}
	return nil
}
