package commands

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

// DaemonCommand provides background helpers around start --detach.
var DaemonCommand = &cli.Command{
	Name:  "daemon",
	Usage: "后台进程：start / stop / status / logs（等价于 start --detach）",
	Subcommands: []*cli.Command{
		{
			Name:   "start",
			Usage:  "后台启动服务",
			Action: daemonStartAction,
		},
		{
			Name:   "stop",
			Usage:  "停止后台服务",
			Action: daemonStopAction,
		},
		{
			Name:   "restart",
			Usage:  "重启后台服务",
			Action: daemonRestartAction,
		},
		{
			Name:   "status",
			Usage:  "查看后台状态",
			Action: daemonStatusAction,
		},
		{
			Name:   "logs",
			Usage:  "打印服务日志（strand.log）",
			Flags: []cli.Flag{
				&cli.IntFlag{Name: "n", Value: 80, Usage: "显示最后 N 行"},
			},
			Action: daemonLogsAction,
		},
	},
}

func daemonStartAction(ctx *cli.Context) error {
	cfg, err := loadSettingsStrict(ctx)
	if err != nil {
		return finish("", err)
	}
	return startDetached(ctx, cfg)
}

func daemonStopAction(ctx *cli.Context) error {
	return stopAction(ctx)
}

func daemonRestartAction(ctx *cli.Context) error {
	cfg, err := loadSettings(ctx)
	if err != nil {
		return finish("", err)
	}
	if alive, _ := runningPID(cfg); alive {
		_ = stopAction(ctx)
	}
	cfg2, err := loadSettingsStrict(ctx)
	if err != nil {
		return finish("", err)
	}
	return startDetached(ctx, cfg2)
}

func daemonStatusAction(ctx *cli.Context) error {
	return statusAction(ctx)
}

func daemonLogsAction(ctx *cli.Context) error {
	cfg, err := loadSettings(ctx)
	if err != nil {
		return finish("", err)
	}
	path := cfg.LogPath()
	b, err := os.ReadFile(path)
	if err != nil {
		return finish("", fmt.Errorf("无法打开日志 %s: %w", path, err))
	}
	n := ctx.Int("n")
	if n <= 0 {
		n = 80
	}
	return finish(trimLastLines(string(b), n), nil)
}

func trimLastLines(s string, n int) string {
	if n <= 0 || s == "" {
		return s
	}
	lines := splitKeep(s)
	if len(lines) <= n {
		return s
	}
	return joinLines(lines[len(lines)-n:])
}

func splitKeep(s string) []string {
	out := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	out := lines[0]
	for i := 1; i < len(lines); i++ {
		out += "\n" + lines[i]
	}
	return out
}
