package commands

import (
	"fmt"
	"os"
	"strings"

	"strand/internal/settings"

	"github.com/urfave/cli/v2"
)

var ConfigCommand = &cli.Command{
	Name:  "config",
	Usage: "查看或生成配置",
	Subcommands: []*cli.Command{
		{
			Name:   "show",
			Usage:  "打印当前配置，密钥只显示前后几位",
			Action: configShowAction,
		},
		{
			Name:  "init",
			Usage:  "在默认路径生成配置与 .env",
			Flags: []cli.Flag{
				&cli.BoolFlag{Name: "force", Usage: "覆盖已经存在的配置"},
				&cli.StringFlag{Name: "dir", Usage: "数据目录（默认 ~/.strand）"},
			},
			Action: configInitAction,
		},
		{
			Name:   "path",
			Usage:  "打印将要使用的配置文件路径",
			Action: configPathAction,
		},
	},
}

func configShowAction(ctx *cli.Context) error {
	cfg, err := loadSettings(ctx)
	if err != nil {
		return finish("", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "config:   %s\n", emptyDash(cfg.Path))
	fmt.Fprintf(&b, "data.dir: %s\n", cfg.Data.Dir)
	fmt.Fprintf(&b, "api:      enabled=%v addr=%s token=%s\n",
		cfg.API.Enabled, cfg.Addr(), secretLabel(cfg.API.Token))
	fmt.Fprintf(&b, "database: type=%s path=%s\n", cfg.Database.Type, cfg.Database.Path)
	if cfg.Database.Type == "postgres" {
		fmt.Fprintf(&b, "          host=%s port=%d user=%s name=%s\n",
			cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Name)
	}
	fmt.Fprintf(&b, "runtime:  lang=%s log=%s auto_traders=%v\n",
		cfg.Runtime.Language, cfg.Runtime.LogLevel, cfg.Runtime.AutoStartTraders)
	fmt.Fprintf(&b, "trading:  enabled=%v\n", cfg.Trading.Enabled)
	fmt.Fprintf(&b, "market:   alpaca=%s twelve=%s\n",
		secretLabel(cfg.Market.AlpacaAPIKey), secretLabel(cfg.Market.TwelveDataKey))
	fmt.Fprintf(&b, "projects: %d\n", len(cfg.Projects))
	for _, p := range cfg.Projects {
		plats := make([]string, 0, len(p.Platforms))
		for _, raw := range p.Platforms {
			if t, _ := raw["type"].(string); t != "" {
				plats = append(plats, t)
			}
		}
		fmt.Fprintf(&b, "  - %s agent=%s platforms=%s\n",
			p.Name, p.Agent.Type, strings.Join(plats, ","))
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		fmt.Fprintf(&b, "jwt:      %s\n", secretLabel(v))
	} else {
		fmt.Fprintf(&b, "jwt:      (未设置 JWT_SECRET)\n")
	}
	return finish(b.String(), nil)
}

func configInitAction(ctx *cli.Context) error {
	opt := settings.InitOptions{
		Force:   ctx.Bool("force"),
		DataDir: ctx.String("dir"),
	}
	if p := configPathFrom(ctx); p != "" {
		opt.ConfigPath = p
	}
	res, err := settings.WriteInit(opt)
	if err != nil {
		return finish("", err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "已初始化 Strand 配置\n")
	fmt.Fprintf(&b, "  数据目录  %s\n", res.DataDir)
	fmt.Fprintf(&b, "  配置文件  %s\n", res.ConfigPath)
	fmt.Fprintf(&b, "  密钥文件  %s\n", res.EnvPath)
	fmt.Fprintf(&b, "\n下一步:\n")
	fmt.Fprintf(&b, "  1. 编辑配置，按需取消注释 [[projects]] 接入微信等通道\n")
	fmt.Fprintf(&b, "  2. 在 .env 中填写平台 Token\n")
	fmt.Fprintf(&b, "  3. 运行: strand start\n")
	return finish(b.String(), nil)
}

func configPathAction(ctx *cli.Context) error {
	resolved, err := settings.ResolveConfigPath(configPathFrom(ctx))
	if err != nil {
		return finish("", err)
	}
	if resolved == "" {
		resolved = settings.DefaultConfigPath()
	}
	return finish(resolved, nil)
}

func secretLabel(s string) string {
	m := settings.Mask(s)
	if m == "" {
		return "(空)"
	}
	return m
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(未找到文件，使用默认值)"
	}
	return s
}
