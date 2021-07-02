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
	Fuzzy          bool
}

type WindowTraits struct {
	Title    string
	Class    string
	Instance string
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

func (q WindowQuery) MatchWindow(t WindowTraits) bool {
	if q.Empty() {
		return false
	}
	match := true
	if q.Name != "" {
		if q.Fuzzy {
			if q.nameRegexp == nil {
				return false
			} else {
				if !q.nameRegexp.MatchString(t.Title) {
					return false
				}
			}
		}
	}
	if q.Class != "" {
		if q.Fuzzy {
			if q.classRegexp == nil {
				return false
			} else {
				if !q.classRegexp.MatchString(t.Class) {
					return false
				}
			}
		}
	}
	if q.Instance != "" {
		if q.Fuzzy {
			if q.instanceRegexp == nil {
				return false
			} else {
				if !q.instanceRegexp.MatchString(t.Instance) {
					return false
				}
			}
		}
	}
	return match
}

func prepareWindowQuery(query WindowQuery) WindowQuery {
	if query.Fuzzy {
		if query.Name != "" {
			query.nameRegexp = regexp.MustCompile(query.Name)
		}
		if query.Class != "" {
			query.classRegexp = regexp.MustCompile(query.Class)
		}
		if query.Instance != "" {
			query.instanceRegexp = regexp.MustCompile(query.Instance)
		}
	}
	return query
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
	query = prepareWindowQuery(query)
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
		if query.MatchWindow(*traits) {
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
