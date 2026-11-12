package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func writePID(path string) error {
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o644)
}

func readPID(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return 0, fmt.Errorf("empty pid file")
	}
	return strconv.Atoi(s)
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, Signal(0) checks existence.
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

func stopPID(path string, timeout time.Duration) error {
	pid, err := readPID(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("服务未在运行（找不到 pid 文件）")
		}
		return err
	}
	if !processAlive(pid) {
		_ = os.Remove(path)
		return fmt.Errorf("服务未在运行（pid %d 已退出）", pid)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("发送 SIGTERM 失败: %w", err)
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			_ = os.Remove(path)
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	_ = proc.Signal(syscall.SIGKILL)
	_ = os.Remove(path)
	return nil
}
