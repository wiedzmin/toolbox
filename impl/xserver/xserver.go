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

func GetCurrentWindowName(X *xgbutil.XUtil) (*string, error) {
	var err error
	if X == nil {
		X, err = xgbutil.NewConn()
		if err != nil {
			return nil, err
		}
	}
	active, err := ewmh.ActiveWindowGet(X)
	if err != nil {
		return nil, err
	}
	windowName, err := ewmh.WmNameGet(X, active)
	return &windowName, err
}

func FindWindow(X *xgbutil.XUtil, query WindowQuery) (*xproto.Window, error) {
	l := logger.Sugar()
	var err error
	if X == nil {
		X, err = xgbutil.NewConn()
		if err != nil {
			return nil, err
		}
	}
	query = prepareWindowQuery(query)
	l.Debugw("[FindWindow]", "query", query)
	windows, err := ewmh.ClientListGet(X)
	if err != nil {
		return nil, err
	}
	for _, win := range windows {
		if query.MatchWindow(X, win) {
			return &win, nil
		}
	}
	return nil, ErrWindowNotFound{query}
}

func SetActiveWindow(X *xgbutil.XUtil, query WindowQuery) error {
	l := logger.Sugar()
	var err error
	if X == nil {
		X, err = xgbutil.NewConn()
		if err != nil {
			return err
		}
	}
	win, err := FindWindow(X, query)
	l.Debugw("[SetActiveWindow]", "win", win, "err", err)
	if err != nil {
		return err
	}
	return ewmh.ActiveWindowSet(X, *win)
}
