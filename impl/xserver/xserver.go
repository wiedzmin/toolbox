package xserver

import (
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

func GetCurrentWindowName(X *xgb.Conn) (*string, error) {
	var err error
	if X == nil {
		X, err = xgb.NewConn()
		if err != nil {
			return nil, err
		}
	}
	setup := xproto.Setup(X)
	root := setup.DefaultScreen(X).Root
	activeAtom, err := xproto.InternAtom(X, true, uint16(len("_NET_ACTIVE_WINDOW")), "_NET_ACTIVE_WINDOW").Reply()
	if err != nil {
		return nil, err
	}
	nameAtom, err := xproto.InternAtom(X, true, uint16(len("_NET_WM_NAME")), "_NET_WM_NAME").Reply()
	if err != nil {
		return nil, err
	}
	reply, err := xproto.GetProperty(X, false, root, activeAtom.Atom, xproto.GetPropertyTypeAny, 0, (1<<32)-1).Reply()
	if err != nil {
		return nil, err
	}
	windowId := xproto.Window(xgb.Get32(reply.Value))
	reply, err = xproto.GetProperty(X, false, windowId, nameAtom.Atom, xproto.GetPropertyTypeAny, 0, (1<<32)-1).Reply()
	if err != nil {
		return nil, err
	}
	windowName := string(reply.Value)
	return &windowName, nil
}
