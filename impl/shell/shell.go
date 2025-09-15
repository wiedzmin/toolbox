package shell

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/shell/tmux"
	"go.uber.org/zap"
)

const (
	TerminalCommandFlagName = "term-command"
	TerminalBackendFlagName = "term-backend"
	TerminalBackendDefault  = "kitty"

	grepMaxFileSize = 65536
	osMetadataPath  = "/etc/os-release"
	pkexecPathNixos = "/run/wrappers/bin/pkexec"
)

var (
	logger                *zap.Logger
	KittySocketEnvVarName = fmt.Sprintf("%s_KITTY_SOCKET", impl.EnvPrefix)
)

type TerminalTraits struct {
	Backend, VTermCmd, TmuxSession string
}

type ErrInvalidCmd struct {
	Cmd string
}

func (e ErrInvalidCmd) Error() string {
	return fmt.Sprintf("invalid shell command: '%s'", e.Cmd)
}

type ErrNoEnvVar struct {
	Name string
}

func (e ErrNoEnvVar) Error() string {
	return fmt.Sprintf("no value found for: %s", e.Name)
}

func init() {
	logger = impl.NewLogger()
}

func TermTraitsFromContext(ctx *cli.Context) TerminalTraits {
	return TerminalTraits{
		Backend:     ctx.String(TerminalBackendFlagName),
		VTermCmd:    ctx.String(TerminalCommandFlagName),
		TmuxSession: ctx.String("tmux-session"),
	}
}

// ShellCmd executes shell commands
// environment variables are provided as string slice of "<name>=<value>" entries
func ShellCmd(cmd string, input *string, cwd *string, env []string, needOutput, combineOutput bool) (*string, error) {
	l := logger.Sugar()
	c := exec.Command("sh", "-c", cmd)
	c.Env = append(os.Environ(), env...)
	if input != nil {
		reader := strings.NewReader(*input)
		c.Stdin = reader
	}
	if cwd != nil {
		c.Dir = *cwd
	}

	l.Debugw("[ShellCmd]", "cmd", cmd, "env", env, "input", input)
	if needOutput {
		var out []byte
		var err error
		if combineOutput {
			out, err = c.CombinedOutput()
		} else {
			out, err = c.Output()
		}
		result := strings.TrimRight(string(out), "\n")
		l.Debugw("[ShellCmd]", "cmd", cmd, "result", result, "err", err)
		return &result, err
	} else {
		err := c.Run()
		return nil, err
	}
}

// FIXME: remove duplication below
func OpenTerminal(path string, traits TerminalTraits) error {
	l := logger.Sugar()
	if len(traits.VTermCmd) == 0 && traits.Backend != "kitty" {
		return ErrInvalidCmd{Cmd: traits.VTermCmd}
	}
	if impl.HasSpaces(path) {
		path = impl.Quote(path)
	}
	switch traits.Backend {
	case "kitty":
		l.Debugw("[OpenTerminal]", "backend", "kitty")
		return OpenKitty(path)
	// TODO: implement opening tmux pane (or Zellij, or something alike)
	default:
		// NOTE: because VT programs has no agreement on syntax for just opening new
		// window/pane with particular CWD, we could not relay on any defaults for this
		// So it would be more correct to implement this functionality for each VT tool
		// being put into use over time (such as for Kitty over here).
		l.Debugw("[OpenTerminal]", "error", "unsupported terminal")
		return ErrInvalidCmd{Cmd: traits.VTermCmd}
	}
}

func RunInTerminal(cmd, title string, traits TerminalTraits) error {
	l := logger.Sugar()
	if len(traits.VTermCmd) == 0 && traits.Backend != "kitty" {
		return ErrInvalidCmd{Cmd: traits.VTermCmd}
	}
	switch traits.Backend {
	case "kitty":
		return RunInKitty(cmd, title)
	case "tmux":
		return RunInTmux(cmd, traits.TmuxSession, title, traits.VTermCmd)
	default:
		l.Debugw("[OpenInTerminal]", "backend", traits.Backend, "summary", "unknown terminal backend")
		return RunInBareTerminal(cmd, traits.VTermCmd)
	}
}

func OpenKitty(path string) error {
	l := logger.Sugar()
	impl.EnsureBinary("kitty", *logger)
	l.Debugw("[OpenKitty]", "path", path)
	socket := os.Getenv(KittySocketEnvVarName)
	if socket == "" {
		return ErrNoEnvVar{Name: KittySocketEnvVarName}
	}
	_, err := ShellCmd(fmt.Sprintf("kitty @ --to %s launch --cwd %s --type os-window", socket, path), nil, nil, nil, false, false)
	if err != nil {
		// NOTE: most likely, kitty is not running, hence no socket listening - let's start new instance with required CWD
		_, err = ShellCmd(fmt.Sprintf("kitty --working-directory %s", path), nil, nil, nil, false, false)
		if err != nil {
			return err
		}
	}
	return nil
}

func RunInKitty(cmd, title string) error {
	impl.EnsureBinary("kitty", *logger)
	socket := os.Getenv(KittySocketEnvVarName)
	if socket == "" {
		return ErrNoEnvVar{Name: KittySocketEnvVarName}
	}
	_, err := ShellCmd(fmt.Sprintf("kitty @ --to %s launch --type os-window sh -c \"%s\"", socket, cmd), nil, nil, nil, false, false)
	if err != nil {
		return err
	}
	return nil
}

func RunInTmux(cmd, title, session, vtermCmd string) error {
	if len(session) > 0 {
		impl.EnsureBinary("tmux", *logger)
		session, err := tmux.GetSession(session, false, true)
		switch err.(type) {
		case tmux.ErrSessionNotFound:
			return RunInBareTerminal(cmd, vtermCmd)
		default:
			if err != nil {
				return err
			}
			return session.NewWindow(cmd, title, "", true)
		}
	} else {
		return RunInBareTerminal(cmd, vtermCmd)
	}
}

func RunInBareTerminal(cmd, vtermCmd string) error {
	// TODO: elaborate/ensure `transient` commands proper handling, i.e. those who need "; read" thereafter
	_, err := ShellCmd(fmt.Sprintf("%s \"sh -c %s\"", vtermCmd, cmd), nil, nil, nil, false, false)
	return err
}

func Grep(path, token string) (bool, error) {
	r, err := regexp.Compile(token)
	if err != nil {
		return false, err
	}

	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, grepMaxFileSize)
	scanner.Buffer(buf, grepMaxFileSize)

	for scanner.Scan() {
		if r.MatchString(scanner.Text()) {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	return false, nil
}

func PkexecPath() string {
	found, _ := Grep(osMetadataPath, "NixOS")
	if found {
		return pkexecPathNixos
	}
	return "pkexec"
}

// RunDetached runs shell command in new process group, effectively unwiring it from parent's one
// so this command won't be killed on parent exit
func RunDetached(command string) error {
	l := logger.Sugar()
	parts := strings.Split(command, " ")
	cmd, err := exec.LookPath(parts[0])
	if err != nil {
		return err
	}
	attr := &os.ProcAttr{Sys: &syscall.SysProcAttr{Setpgid: true}}
	argv := append([]string{cmd}, parts[1:]...)
	process, err := os.StartProcess(cmd, argv, attr)
	if err != nil {
		return err
	}
	l.Debugw("[RunDetached]", "process.Pid", process.Pid)
	process.Release()
	return nil
}
