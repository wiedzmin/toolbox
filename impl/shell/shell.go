package shell

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

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
		return nil
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
