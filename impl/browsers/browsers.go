package browsers

import (
	"github.com/wiedzmin/toolbox/impl/fs"
	"github.com/wiedzmin/toolbox/impl/ui"
	"github.com/wiedzmin/toolbox/impl/xserver/xkb"
)

var (
	RegexTimedSessionName = `session-(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})-[0-9]{2}-[0-9]{2}-[0-9]{2}`
)

// SelectSession collects session files and allows selecting one
func SelectSession(path, prompt, tool, font string, regexpsWhitelist, regexpsBlacklist []string) (*string, error) {
	files := fs.NewFSCollection(path, regexpsWhitelist, regexpsBlacklist, false).Emit(false)
	xkb.EnsureEnglishKeyboardLayout()
	sessionName, err := ui.GetSelection(files, prompt, tool, font, true, false)

	if err != nil {
		return nil, err
	}
	return &sessionName, nil
}
