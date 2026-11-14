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
	Usage: "Show or create configuration",
	Subcommands: []*cli.Command{
		{
			Name:        "show",
			Usage:        "Print the active config (secrets masked)",
			Description: "Shows data dir, API bind, database, runtime, and project/channel summary.",
			Action:      configShowAction,
		},
		{
			Name:  "init",
			Usage:  "Create config.toml and .env under the data directory",
			Description: "Default data dir is ~/.strand. Use --force to overwrite an existing config.",
			Flags: []cli.Flag{
				&cli.BoolFlag{Name: "force", Usage: "Overwrite an existing config"},
				&cli.StringFlag{Name: "dir", Usage: "Data directory (default ~/.strand)"},
			},
			Action: configInitAction,
		},
		{
			Name:   "path",
			Usage:  "Print the resolved config file path",
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
		fmt.Fprintf(&b, "jwt:      (JWT_SECRET not set)\n")
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
	fmt.Fprintf(&b, "Initialized Strand config\n")
	fmt.Fprintf(&b, "  data dir   %s\n", res.DataDir)
	fmt.Fprintf(&b, "  config     %s\n", res.ConfigPath)
	fmt.Fprintf(&b, "  secrets    %s\n", res.EnvPath)
	fmt.Fprintf(&b, "\nNext steps:\n")
	fmt.Fprintf(&b, "  1. Review config.toml; prefer `strand messaging …` or Settings → Messaging for channels\n")
	fmt.Fprintf(&b, "  2. Fill tokens in .env (JWT_SECRET, STRAND_API_TOKEN, platform tokens)\n")
	fmt.Fprintf(&b, "  3. Run: strand start\n")
	fmt.Fprintf(&b, "  4. Web: cd web && pnpm install && pnpm run dev\n")
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
		return "(empty)"
	}
	return m
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(no file found; using defaults)"
	}
	return s
}
