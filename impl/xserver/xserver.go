package xserver

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/jezek/xgbutil"
	"github.com/jezek/xgbutil/ewmh"
	"github.com/jezek/xgbutil/icccm"
	jsoniter "github.com/json-iterator/go"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/redis"
	"github.com/wiedzmin/toolbox/impl/shell"
	"go.uber.org/zap"
)

var (
	logger *zap.Logger
	r      *redis.Client
)

func init() {
	logger = impl.NewLogger()
	var err error
	r, err = redis.NewRedisLocal()
	if err != nil {
		l := logger.Sugar()
		l.Fatalw("[init]", "failed connecting to Redis", err)
	}
	impl.EnsureBinary("xkb-switch", *logger)
}

type X struct {
	connXU  *xgbutil.XUtil
	connXGB *xgb.Conn
}

type WindowQuery struct {
	Name           string
	nameRegexp     *regexp.Regexp
	Class          string
	classRegexp    *regexp.Regexp
	Instance       string
	instanceRegexp *regexp.Regexp
	Role           string
	roleRegexp     *regexp.Regexp
	Fuzzy          bool
	prepared       bool
}

type WindowTraits struct {
	Title    string
	Class    string
	Instance string
	Role     string
}

func (t WindowTraits) AsMap() map[string]string {
	return map[string]string{
		"title":    t.Title,
		"class":    t.Class,
		"instance": t.Instance,
	}
}

func (t WindowTraits) ListNames() []string {
	return []string{"title", "class", "instance"}
}

type WindowRule struct {
	Class    string `json:"class"`
	Title    string `json:"title"`
	Instance string `json:"instance"`
	Role     string `json:"role"`
	Desktop  string `json:"desktop"`
	Activate bool   `json:"activate"`
}

type WindowRules struct {
	data   []byte
	parsed []WindowRule
}

type Workspaces struct {
	data   []byte
	parsed []string
}

type ErrWindowNotFound struct {
	Query WindowQuery
}

func (e ErrWindowNotFound) Error() string {
	return fmt.Sprintf("no windows found by query: %v", e.Query)
}

func (q WindowQuery) Empty() bool {
	return q.Name == "" && q.Class == "" && q.Instance == ""
}

func (q WindowQuery) MatchTraits(traits WindowTraits) bool {
	if q.Empty() {
		return false
	}
	if q.Fuzzy {
		if !q.prepared {
			q.prepare()
		}
		if q.Name != "" && !q.nameRegexp.MatchString(traits.Title) {
			return false
		}
		if q.Class != "" && !q.classRegexp.MatchString(traits.Class) {
			return false
		}
		if q.Instance != "" && !q.instanceRegexp.MatchString(traits.Instance) {
			return false
		}
		if q.Role != "" && !q.roleRegexp.MatchString(traits.Role) {
			return false
		}
	} else {
		if q.Name != traits.Title {
			return false
		}
		if q.Class != traits.Class {
			return false
		}
		if q.Instance != traits.Instance {
			return false
		}
		if q.Role != traits.Role {
			return false
		}
	}
	return true
}

func (q WindowQuery) prepare() {
	if q.Fuzzy {
		if q.Name != "" {
			q.nameRegexp = regexp.MustCompile(q.Name)
		}
		if q.Class != "" {
			q.classRegexp = regexp.MustCompile(q.Class)
		}
		if q.Instance != "" {
			q.instanceRegexp = regexp.MustCompile(q.Instance)
		}
		if q.Role != "" {
			q.roleRegexp = regexp.MustCompile(q.Role)
		}
	}
	q.prepared = true
}

func NewX() (*X, error) {
	l := logger.Sugar()
	connXgb, err := xgb.NewConn()
	connXu, err := xgbutil.NewConnXgb(connXgb)
	if err != nil {
		l.Warnw("[NewX]", "err", err)
		return nil, err
	}
	l.Debugw("[NewX]", "connXu", fmt.Sprintf("%v", connXu), "connXgb", fmt.Sprintf("%v", connXgb))
	return &X{connXU: connXu, connXGB: connXgb}, nil
}

func (x *X) GetWindowTraits(win *xproto.Window) (*WindowTraits, error) {
	var window xproto.Window
	var err error
	if win == nil {
		window, err = ewmh.ActiveWindowGet(x.connXU)
		if err != nil {
			return nil, err
		}
	} else {
		window = *win
	}
	title, err := ewmh.WmNameGet(x.connXU, window)
	if err != nil {
		return nil, err
	}
	wmClassData, err := icccm.WmClassGet(x.connXU, window)
	if err != nil {
		return nil, err
	}
	// TODO: try to implement WM_WINDOW_ROLE retrieval
	return &WindowTraits{
		Title:    title,
		Class:    wmClassData.Class,
		Instance: wmClassData.Instance,
	}, nil
}

func (x *X) ListWindows() ([]xproto.Window, error) {
	windows, err := ewmh.ClientListGet(x.connXU)
	if err != nil {
		return nil, err
	}
	return windows, nil
}

func (x *X) FindWindow(query WindowQuery) (*xproto.Window, error) {
	l := logger.Sugar()
	l.Debugw("[FindWindow]", "query", query)
	windows, err := x.ListWindows()
	if err != nil {
		return nil, err
	}
	for _, win := range windows {
		traits, err := x.GetWindowTraits(&win)
		if err != nil {
			return nil, err
		}
		if query.MatchTraits(*traits) {
			return &win, nil
		}
	}
	return nil, ErrWindowNotFound{query}
}

func (x *X) BringWindowAbove(query WindowQuery) error {
	l := logger.Sugar()
	win, err := x.FindWindow(query)
	l.Debugw("[BringWindowAbove]", "win", win, "err", err)
	if err != nil {
		return err
	}
	winDesktop, err := ewmh.WmDesktopGet(x.connXU, *win)
	if err != nil {
		return err
	}
	curDesktop, err := ewmh.CurrentDesktopGet(x.connXU)
	l.Debugw("[BringWindowAbove]", "winDesktop", winDesktop, "curDesktop", curDesktop)
	err = ewmh.CurrentDesktopSet(x.connXU, winDesktop)
	if err != nil {
		return err
	}
	// NOTE: Using a workaround.
	// Instead of just ewmh.ActiveWindowSet which has no effect.
	xproto.SetInputFocus(x.connXGB, xproto.InputFocusParent, *win, xproto.TimeCurrentTime)
	err = ewmh.WmStateReq(x.connXU, *win, ewmh.StateToggle, "_NET_WM_STATE_ABOVE")
	if err != nil {
		return err
	}
	return nil
}

func NewWindowRules(data []byte) (*WindowRules, error) {
	var result WindowRules
	result.data = data
	err := jsoniter.Unmarshal(data, &result.parsed)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func WindowRulesFromRedis(key string) (*WindowRules, error) {
	rulesData, err := r.GetValue(key)
	if err != nil {
		return nil, err
	}
	return NewWindowRules(rulesData)
}

func (wr *WindowRules) List() []WindowRule {
	return wr.parsed
}

func (wr *WindowRules) MatchTraits(traits WindowTraits) (*WindowRule, error) {
	l := logger.Sugar()
	var result *WindowRule
	var err error
	matchedClass, matchedTitle, matchedInstance, matchedRole := false, false, false, false
	for _, rule := range wr.parsed {
		l.Debugw("[WindowRules.MatchTraits]====================================================================")
		if rule.Class != "" {
			matchedClass, err = regexp.MatchString(rule.Class, traits.Class)
			l.Debugw("[WindowRules.MatchTraits]", "rule.Class", rule.Class, "traits.Class", traits.Class, "matchedClass", matchedClass)
			if err != nil {
				l.Warnw("[WindowRules.MatchTraits]", "err", err)
				return nil, err
			}
			if !matchedClass {
				continue
			}
		} else {
			matchedClass = true
		}
		if rule.Title != "" {
			matchedTitle, err = regexp.MatchString(rule.Title, traits.Title)
			l.Debugw("[WindowRules.MatchTraits]", "rule.Title", rule.Title, "traits.Title", traits.Title, "matchedTitle", matchedTitle)
			if err != nil {
				l.Warnw("[WindowRules.MatchTraits]", "err", err)
				return nil, err
			}
			if !matchedTitle {
				continue
			}
		} else {
			matchedTitle = true
		}
		if rule.Instance != "" {
			matchedInstance, err = regexp.MatchString(rule.Instance, traits.Instance)
			l.Debugw("[WindowRules.MatchTraits]", "rule.Instance", rule.Instance, "traits.Instance", traits.Instance, "matchedInstance", matchedInstance)
			if err != nil {
				l.Warnw("[WindowRules.MatchTraits]", "err", err)
				return nil, err
			}
			if !matchedInstance {
				continue
			}
		} else {
			matchedInstance = true
		}
		if rule.Role != "" {
			matchedRole, err = regexp.MatchString(rule.Role, traits.Role)
			l.Debugw("[WindowRules.MatchTraits]", "rule.Role", rule.Role, "traits.Role", traits.Role, "matchedRole", matchedRole)
			if err != nil {
				l.Warnw("[WindowRules.MatchTraits]", "err", err)
				return nil, err
			}
			if !matchedRole {
				continue
			}
		} else {
			matchedRole = true
		}
		if matchedClass && matchedTitle && matchedInstance && matchedRole {
			result = &rule
			break
		}
	}
	return result, nil
}

// TODO: consider code generation usage
func NewWorkspaces(data []byte) (*Workspaces, error) {
	var result Workspaces
	result.data = data
	err := jsoniter.Unmarshal(data, &result.parsed)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func WorkspacesFromRedis(key string) (*Workspaces, error) {
	workspacesData, err := r.GetValue(key)
	if err != nil {
		return nil, err
	}
	return NewWorkspaces(workspacesData)
}

func (w *Workspaces) List() []string {
	return w.parsed
}

func CurrentWorkspaceTitle() (string, error) {
	var result string
	impl.EnsureBinary("wmctrl", *logger)
	out, err := shell.ShellCmd("wmctrl -d", nil, nil, nil, true, true)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(strings.TrimSpace(*out), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if strings.TrimSpace(fields[1]) == "*" {
			result = strings.Join(fields[8:len(fields)], " ")
			break
		}
	}
	return result, nil
}

func HeadsFingerprint() (map[string]string, []string, error) {
	impl.EnsureBinary("xrandr", *logger)
	headEDIDs := make(map[string]string)
	xrandrOutput, err := shell.ShellCmd("xrandr --prop", nil, nil, nil, true, false)
	if err != nil {
		return nil, nil, err
	}
	xrandrOutputSlice := strings.Split(*xrandrOutput, "\n")
	var head, edid string
	var headNames []string
	for i, line := range xrandrOutputSlice {
		if strings.Contains(line, " connected ") {
			head = strings.Fields(line)[0]
			headNames = append(headNames, head)
		}
		if strings.Contains(line, "EDID:") {
			edid = strings.ReplaceAll(strings.ReplaceAll(strings.Join(xrandrOutputSlice[i+1:i+9], ""), "\n", ""), "\t", "")
		}
		if head != "" && edid != "" {
			headEDIDs[head] = edid
			head = ""
			edid = ""
		}
	}
	return headEDIDs, headNames, nil
}

func ReadClipboard(primary bool) (*string, error) {
	cbFlag := " -b"
	if primary {
		cbFlag = "" // NOTE: primary selection is default for `xsel`
	}
	return shell.ShellCmd(fmt.Sprintf("xsel -o%s", cbFlag), nil, nil, nil, true, false)
}

func WriteClipboard(data *string, primary bool) error {
	cbFlag := " -b"
	if primary {
		cbFlag = "" // NOTE: primary selection is default for `xsel`
	}
	_, err := shell.ShellCmd(fmt.Sprintf("xsel -i%s", cbFlag), data, nil, nil, false, false)
	return err
}
