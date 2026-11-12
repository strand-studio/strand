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
	Usage:   "停止后台或前台的 strand 服务",
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
	return finish(fmt.Sprintf("已停止（%s）", cfg.ProcPath()), nil)
}
