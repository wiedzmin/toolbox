package browsers

import (
	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl/fs"
	"github.com/wiedzmin/toolbox/impl/ui"
	"github.com/wiedzmin/toolbox/impl/xserver/xkb"
)

var (
	RegexTimedSessionName = `session-(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})-[0-9]{2}-[0-9]{2}-[0-9]{2}`
)

// SelectSession collects session files and allows selecting one
func SelectSession(ctx *cli.Context, path, prompt string) (*string, error) {
	// TODO: should we use "org" session type here or somewhere else among the call stack?
	files, err := fs.CollectFiles(path, false, []string{"org$"})
	if err != nil {
		return nil, err
	}
	xkb.EnsureEnglishKeyboardLayout()
	sessionName, err := ui.GetSelection(ctx, files, prompt, true, false)

	if err != nil {
		return nil, err
	}
	return &sessionName, nil
}
