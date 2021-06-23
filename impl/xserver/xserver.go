package xserver

import (
	"fmt"
	"regexp"

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
	conn *xgbutil.XUtil
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

type ErrWindowNotFound struct {
	Query WindowQuery
}

func (e ErrWindowNotFound) Error() string {
	return fmt.Sprintf("no windows found by query: %v", e.Query)
}

func (q WindowQuery) Empty() bool {
	return q.Name == "" && q.Class == "" && q.Instance == ""
}

func (q WindowQuery) MatchWindow(X *xgbutil.XUtil, win xproto.Window) bool {
	if q.Empty() {
		return false
	}
	var wmClassData *icccm.WmClass
	var err error
	if q.Class != "" || q.Instance != "" {
		wmClassData, err = icccm.WmClassGet(X, win)
		if err != nil {
			return false
		}
	}
	match := true
	if q.Name != "" {
		if q.Fuzzy {
			if q.nameRegexp == nil {
				return false
			} else {
				windowName, err := ewmh.WmNameGet(X, win)
				if err != nil {
					return false
				}
				if !q.nameRegexp.MatchString(windowName) {
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
				if !q.classRegexp.MatchString(wmClassData.Class) {
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
				if !q.instanceRegexp.MatchString(wmClassData.Instance) {
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
	conn, err := xgbutil.NewConn()
	if err != nil {
		l.Warnw("[NewX]", "err", err)
		return nil, err
	}
	l.Debugw("[NewX]", "conn", fmt.Sprintf("%v", conn))
	return &X{conn}, nil
}

func (x *X) GetCurrentWindowName() (*string, error) {
	active, err := ewmh.ActiveWindowGet(x.conn)
	if err != nil {
		return nil, err
	}
	windowName, err := ewmh.WmNameGet(x.conn, active)
	return &windowName, err
}

func (x *X) FindWindow(query WindowQuery) (*xproto.Window, error) {
	l := logger.Sugar()
	query = prepareWindowQuery(query)
	l.Debugw("[FindWindow]", "query", query)
	windows, err := ewmh.ClientListGet(x.conn)
	if err != nil {
		return nil, err
	}
	for _, win := range windows {
		if query.MatchWindow(x.conn, win) {
			return &win, nil
		}
	}
	return nil, ErrWindowNotFound{query}
}

func (x *X) SetActiveWindow(query WindowQuery) error {
	l := logger.Sugar()
	win, err := x.FindWindow(query)
	l.Debugw("[SetActiveWindow]", "win", win, "err", err)
	if err != nil {
		return err
	}
	return ewmh.ActiveWindowSet(x.conn, *win)
}
