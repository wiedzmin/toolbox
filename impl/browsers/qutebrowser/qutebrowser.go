package qutebrowser

import (
	"bufio"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strings"

	"github.com/wiedzmin/toolbox/impl"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type SessionFormat int8

type Request struct {
	Commands        []string `json:"args"`
	targetArg       string   `json:"target_arg"`
	protocolVersion int      `json:"protocol_version"`
}

const (
	SESSION_FORMAT_YAML     SessionFormat = 0
	SESSION_FORMAT_ORG      SessionFormat = 1
	SESSION_FORMAT_ORG_FLAT SessionFormat = 2
)

var (
	RegexTimedSessionName = `session-(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})-[0-9]{2}-[0-9]{2}-[0-9]{2}`
	logger                *zap.Logger
)

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

func RawSessionsPath() *string {
	path, err := impl.AtHomedir(".local/share/qutebrowser/sessions")
	if err != nil {
		return nil
	}
	return path
}

func (r *Request) Marshal() ([]byte, error) {
	r.protocolVersion = 1
	r.targetArg = ""
	bytes, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func LoadSession(path string) (*SessionLayout, error) {
	l := logger.Sugar()
	l.Debugw("[LoadSession]", "path", path)
	var session SessionLayout
	data, err := ioutil.ReadFile(path)
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

func SaveSession(path string, data *SessionLayout, format SessionFormat) error {
	l := logger.Sugar()
	l.Debugw("[SaveSession]", "path", path, "data", data, "format", format)
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
						l.Debugw("[SaveSession]", "url", p.URL)
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
						l.Debugw("[SaveSession]", "url", p.URL)
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
