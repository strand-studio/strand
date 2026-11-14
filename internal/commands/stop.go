package commands

import (
	"fmt"
	"time"

	"github.com/urfave/cli/v2"
)

// StopCommand stops the server started with strand start.
var StopCommand = &cli.Command{
	Name:    "stop",
	Aliases: []string{"down"},
	Usage:   "Stop a running Strand process (foreground or detached)",
	Action:  stopAction,
}

func stopAction(ctx *cli.Context) error {
	cfg, err := loadSettings(ctx)
	if err != nil {
		return finish("", err)
	}
	if err := stopPID(cfg.ProcPath(), 12*time.Second); err != nil {
		return finish("", err)
	}
	return finish(fmt.Sprintf("stopped (%s)", cfg.ProcPath()), nil)
}
