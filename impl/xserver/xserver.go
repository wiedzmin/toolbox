package xserver

import (
	"fmt"
	"regexp"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/jezek/xgbutil"
	"github.com/jezek/xgbutil/ewmh"
	"github.com/jezek/xgbutil/icccm"
	"github.com/wiedzmin/toolbox/impl"
	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	logger = impl.NewLogger()
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
	return &WindowTraits{
		Title:    title,
		Class:    wmClassData.Class,
		Instance: wmClassData.Instance,
	}, nil
}

func (x *X) FindWindow(query WindowQuery) (*xproto.Window, error) {
	l := logger.Sugar()
	l.Debugw("[FindWindow]", "query", query)
	windows, err := ewmh.ClientListGet(x.connXU)
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
