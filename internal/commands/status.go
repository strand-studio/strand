package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"strand/internal/config"
	"strand/internal/engine"

	"github.com/urfave/cli/v2"
)

// StatusCommand reports whether the server is running.
var StatusCommand = &cli.Command{
	Name:   "status",
	Usage:  "查看服务是否在运行",
	Action: statusAction,
}

func statusAction(ctx *cli.Context) error {
	cfg, err := loadSettings(ctx)
	if err != nil {
		return finish("", err)
	}

	alive, pid := runningPID(cfg)
	var b strings.Builder
	if alive {
		fmt.Fprintf(&b, "运行中  pid=%d\n", pid)
		fmt.Fprintf(&b, "配置    %s\n", cfg.Path)
		fmt.Fprintf(&b, "数据目录 %s\n", cfg.Data.Dir)
		fmt.Fprintf(&b, "API     http://%s\n", cfg.Addr())

		if snap, err := fetchStatus(cfg.Addr(), cfg.API.Token); err == nil {
			fmt.Fprintf(&b, "项目    %d\n", len(snap.Projects))
			fmt.Fprintf(&b, "交易员  loaded=%d running=%d\n", snap.Trading.Loaded, snap.Trading.Running)
			if snap.UptimeSec > 0 {
				fmt.Fprintf(&b, "运行时长 %ds\n", snap.UptimeSec)
			}
		}
	} else {
		fmt.Fprintf(&b, "未运行\n")
		if cfg.Path != "" {
			fmt.Fprintf(&b, "配置 %s\n", cfg.Path)
		} else {
			fmt.Fprintf(&b, "提示: 先执行 strand config init\n")
		}
	}
	return finish(b.String(), nil)
}

func fetchStatus(addr, token string) (engine.Status, error) {
	url := fmt.Sprintf("http://%s%s/status", addr, config.ApiUri)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return engine.Status{}, err
	}
	if token != "" {
		req.Header.Set("X-Strand-Token", token)
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return engine.Status{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return engine.Status{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return engine.Status{}, fmt.Errorf("status %d", resp.StatusCode)
	}
	var snap engine.Status
	if err := json.Unmarshal(body, &snap); err != nil {
		return engine.Status{}, err
	}
	return snap, nil
}
