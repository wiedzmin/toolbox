package xserver

import (
	"github.com/jezek/xgbutil"
	"github.com/jezek/xgbutil/ewmh"
)

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
