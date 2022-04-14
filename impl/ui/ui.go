package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/0xAX/notificator"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/shell"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

const (
	rofiOptionsSeparator  = "\n"
	dmenuOptionsSeparator = "\n"
	dmenuSelectionLinesCount = 15

	SelectorToolFlagName = "selector-tool"
	SelectorTool = "dmenu"
)

var notify *notificator.Notificator
var logger *zap.Logger

func init() {
	notify = notificator.New(notificator.Options{
		// DefaultIcon: "icon/default.png",
		AppName: "toolbox",
	})
	logger = impl.NewLogger()
}

// GetSelection returns users choice from list of options, using predefined selector tool
func GetSelection(ctx *cli.Context, seq []string, prompt string, caseInsensitive, normalWindow bool) (string, error) {
	l := logger.Sugar()
	tool := ctx.String(SelectorToolFlagName)
	switch tool {
	case "rofi":
		return GetSelectionRofi(seq, prompt, caseInsensitive, normalWindow, ctx.String(impl.SelectorFontFlagName))
	case "dmenu":
		return GetSelectionDmenu(seq, prompt, caseInsensitive, normalWindow, ctx.String(impl.SelectorFontFlagName))
	case "bemenu":
		return GetSelectionBemenu(seq, prompt, caseInsensitive, normalWindow, ctx.String(impl.SelectorFontFlagName))
	default:
		l.Debugw("[GetSelection]", "tool", tool, "summary", fmt.Sprintf("unknown selector tool '%s'...", tool))
		return "", fmt.Errorf("unknown selector tool: '%s'", tool)
	}
}

// GetSelectionRofi returns users choice from list of options, using Rofi selector tool
func GetSelectionRofi(seq []string, prompt string, caseInsensitive, normalWindow bool, font string/*ignored*/) (string, error) {
	impl.EnsureBinary("rofi", *logger)
	l := logger.Sugar()
	sort.Strings(seq)
	seqStr := strings.Join(seq, rofiOptionsSeparator)
	l.Debugw("[GetSelectionRofi]", "seq", seq, "seqStr", seqStr, "normalWindow", normalWindow)
	caseFlagStr := ""
	if caseInsensitive {
		caseFlagStr = " -i"
	}
	normalWindowStr := ""
	if normalWindow {
		normalWindowStr = " -normal-window"
	}
	result, err := shell.ShellCmd(fmt.Sprintf("rofi%s -dmenu%s -sep '%s' -p '%s'",
		normalWindowStr, caseFlagStr, rofiOptionsSeparator, prompt), &seqStr, nil, nil, true, false)
	return *result, err
}

// GetSelectionDmenu returns users choice from list of options, using Dmenu selector tool
func GetSelectionDmenu(seq []string, prompt string, caseInsensitive, normalWindow/*ignored*/ bool, font string) (string, error) {
	impl.EnsureBinary("dmenu", *logger)
	l := logger.Sugar()
	sort.Strings(seq)
	seqStr := strings.Join(seq, dmenuOptionsSeparator)
	l.Debugw("[GetSelectionDmenu]", "seq", seq, "seqStr", seqStr, "case-insensitive", caseInsensitive)
	caseFlagStr := ""
	if caseInsensitive {
		caseFlagStr = " -i"
	}
	lines := 1
	seqLen := len(seq)
	if seqLen > 0 {
		if seqLen < dmenuSelectionLinesCount{
			lines = seqLen
		} else {
			lines = dmenuSelectionLinesCount
		}
	}
	result, err := shell.ShellCmd(fmt.Sprintf("dmenu%s -p '%s' -l %d -fn '%s'", caseFlagStr, prompt, lines, font),
		&seqStr, nil, nil, true, false)
	return *result, err
}

// GetSelectionBemenu returns users choice from list of options, using Bemenu selector tool
func GetSelectionBemenu(seq []string, prompt string, caseInsensitive, normalWindow/*ignored*/ bool, font string) (string, error) {
	impl.EnsureBinary("bemenu", *logger)
	l := logger.Sugar()
	sort.Strings(seq)
	seqStr := strings.Join(seq, dmenuOptionsSeparator)
	l.Debugw("[GetSelectionBemenu]", "seq", seq, "seqStr", seqStr, "case-insensitive", caseInsensitive)
	caseFlagStr := ""
	if caseInsensitive {
		caseFlagStr = " -i"
	}
	lines := 1
	seqLen := len(seq)
	if seqLen > 0 {
		if seqLen < dmenuSelectionLinesCount{
			lines = seqLen
		} else {
			lines = dmenuSelectionLinesCount
		}
	}
	result, err := shell.ShellCmd(fmt.Sprintf("bemenu%s -p '%s' -l %d -fn '%s'", caseFlagStr, prompt, lines, font),
		&seqStr, nil, nil, true, false)
	return *result, err
}

// TODO: make selector function for fzf, see example(s): https://junegunn.kr/2016/02/using-fzf-in-your-program

func NotifyNormal(title, text string) {
	notify.Push(title, text, "", notificator.UR_NORMAL)
}

func NotifyCritical(title, text string) {
	notify.Push(title, text, "", notificator.UR_CRITICAL)
}
