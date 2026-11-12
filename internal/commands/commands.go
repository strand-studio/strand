// Package commands provides the CLI commands of Strand

package commands

import (
	"errors"
	"fmt"
	"os/exec"
	"strand/pkg/clean"
	"strand/pkg/log/status"

	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"
)

// Strand contains the strand CLI (sub-)commands
var Strand = []*cli.Command{
	StartCommand,
	StopCommand,
	StatusCommand,
	ConfigCommand,
	DaemonCommand,
}

// ExitCode returns the process exit status for an error that app.Run returned without exiting.
// A canceled operation or interrupted prompt exits 0; the status of an external tool is not passed on.
// urfave/cli reports a missing required flag with an unexported type, so its name is compared.
func ExitCode(err error) int {
	var exit cli.ExitCoder
	var execErr *exec.ExitError

	switch {
	case err == nil:
		return 0
	case errors.As(err, &execErr):
		return 1
	case errors.As(err, &exit):
		if code := exit.ExitCode(); code >= 0 && code <= 125 {
			return code
		}
		return 1
	case errors.Is(err, status.ErrCanceled), errors.Is(err, promptui.ErrInterrupt):
		return 0
	case fmt.Sprintf("%T", err) == "*cli.errRequiredFlags":
		return 2
	default:
		return 1
	}
}

// HandleExitErr prints the message of a cli.Exit error like cli.HandleExitCoder, with the message passed
// through clean.DisplayLines, and exists with its code.
func HandleExitErr(_ *cli.Context, err error) {
	switch e := err.(type) {
	case cli.ExitCoder:
		if err.Error() != "" {
			msg := err.Error()

			if _, ok := e.(cli.ErrorFormatter); ok {
				msg = fmt.Sprintf("%+v", err)
			}

			_, _ = fmt.Fprintln(cli.ErrWriter, clean.DisplayLines(msg))
		}

		cli.OsExiter(e.ExitCode())
	case cli.MultiError:
		cli.OsExiter(printMultiError(e))
	}
}

// printMultiError prints the errors of a cli.MultiError like cli.HandleExitCoder and returns the exit code
// of the last cli.ExitCoder, or 1.
func printMultiError(multiErr cli.MultiError) int {
	code := 1

	for _, err := range multiErr.Errors() {
		if nested, ok := err.(cli.MultiError); ok { //nolint:errorlint // matches cli.HandleExitCoder
			code = printMultiError(nested)
		} else if err != nil {
			_, _ = fmt.Fprintln(cli.ErrWriter, clean.DisplayLines(err.Error()))

			if exitErr, ok := err.(cli.ExitCoder); ok { //nolint:errorlint // matches cli.HandleExitCoder
				code = exitErr.ExitCode()
			}
		}
	}

	return code
}
