package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"strand/internal/api"
	"strand/internal/engine"
	"strand/internal/logger"
	"strand/internal/settings"

	"github.com/urfave/cli/v2"
)

var StartCommand = &cli.Command{
	Name:    "start",
	Aliases: []string{"up"},
	Usage:   "启动消息通道、交易与 API 服务",
	Flags:   startFlags,
	Action:  startAction,
}

var startFlags = []cli.Flag{
	&cli.BoolFlag{
		Name:    "detach",
		Aliases: []string{"d"},
		Usage:   "后台运行（写入 pid / log）",
	},
	&cli.BoolFlag{
		Name:   "foreground",
		Hidden: true,
		Usage:  "internal: run in foreground after detach re-exec",
	},
}

func startAction(ctx *cli.Context) error {
	cfg, err := loadSettingsStrict(ctx)
	if err != nil {
		return finish("", err)
	}

	if ctx.Bool("detach") && !ctx.Bool("foreground") {
		return startDetached(ctx, cfg)
	}

	if alive, pid := runningPID(cfg); alive && pid != os.Getpid() {
		return finish(fmt.Sprintf("服务已在运行 (pid %d)。先执行 strand stop", pid), fmt.Errorf("already running"))
	}

	return runServer(ctx.Context, cfg, versionFrom(ctx))
}

func startDetached(ctx *cli.Context, cfg settings.Settings) error {
	if alive, pid := runningPID(cfg); alive {
		return finish(fmt.Sprintf("服务已在运行 (pid %d)", pid), fmt.Errorf("already running"))
	}
	if err := os.MkdirAll(cfg.Data.Dir, 0o755); err != nil {
		return finish("", err)
	}

	exe, err := os.Executable()
	if err != nil {
		return finish("", err)
	}

	// Global flags must come before the subcommand.
	args := make([]string, 0, 6)
	if cfg.Path != "" {
		args = append(args, "--config", cfg.Path)
	}
	args = append(args, "start", "--foreground")

	logFile, err := os.OpenFile(cfg.LogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return finish("", err)
	}

	cmd := exec.Command(exe, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	cmd.Dir = cfg.Data.Dir
	env := os.Environ()
	if cfg.Path != "" {
		env = append(env, "STRAND_CONFIG="+cfg.Path)
	}
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return finish("", err)
	}
	// Parent must NOT write the pid file — the child writes it in runServer.
	// Writing it here makes the child see its own pid and exit as "already running".

	// Parent no longer owns the log fd.
	_ = logFile.Close()

	// Reap early exit so we don't mistake a zombie for a live process.
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		_ = os.Remove(cfg.ProcPath())
		msg := fmt.Sprintf("后台进程立刻退出了，请查看日志: %s", cfg.LogPath())
		if err != nil {
			return finish(msg, err)
		}
		return finish(msg, fmt.Errorf("detached process exited"))
	case <-time.After(1500 * time.Millisecond):
		// still running — prefer pid from file if child already wrote it
	}

	pid := cmd.Process.Pid
	if p, err := readPID(cfg.ProcPath()); err == nil && p > 0 {
		pid = p
	}

	return finish(fmt.Sprintf("已在后台启动 (pid %d)\nAPI: http://%s\n日志: %s", pid, cfg.Addr(), cfg.LogPath()), nil)
}

func runServer(parent context.Context, cfg settings.Settings, version string) error {
	if err := os.MkdirAll(cfg.Data.Dir, 0o755); err != nil {
		return finish("", err)
	}

	// Foreground also writes pid so `strand stop/status` work.
	if err := writePID(cfg.ProcPath()); err != nil {
		return finish("", err)
	}
	defer func() { _ = os.Remove(cfg.ProcPath()) }()

	eng := engine.NewEngine(cfg)
	ctx, cancel := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := eng.Start(ctx); err != nil {
		return finish("", err)
	}
	defer func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 15*time.Second)
		defer c()
		_ = eng.Stop(shutdownCtx)
	}()

	logger.Infof("strand %s started — API http://%s", version, cfg.Addr())
	if err := api.Serve(ctx, cfg, eng, version); err != nil {
		return finish("", err)
	}
	return nil
}

func runningPID(cfg settings.Settings) (bool, int) {
	pid, err := readPID(cfg.ProcPath())
	if err != nil {
		return false, 0
	}
	if processAlive(pid) {
		return true, pid
	}
	_ = os.Remove(cfg.ProcPath())
	return false, pid
}

func versionFrom(ctx *cli.Context) string {
	if ctx != nil && ctx.App != nil && ctx.App.Version != "" {
		return ctx.App.Version
	}
	return "dev"
}
