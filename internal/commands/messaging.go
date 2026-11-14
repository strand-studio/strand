package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"strand/internal/settings"
	"strand/internal/store"

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

// MessagingCommand manages social platform bindings (chat + alerts).
var MessagingCommand = &cli.Command{
	Name:        "messaging",
	Aliases:     []string{"channel", "channels"},
	Usage:       "Configure messaging channels (chat + alerts) without editing TOML",
	Description: "Preferred over [[projects]] in config.toml. Bindings are stored in the database and can also be managed in the Web console under Settings → Messaging.",
	Subcommands: []*cli.Command{
		{
			Name:  "supported",
			Usage: "List common platform types",
			Action: func(c *cli.Context) error {
				fmt.Println("telegram  weixin  wecom  discord  slack  feishu  lark  dingtalk")
				fmt.Println("qq  qqbot  line  matrix  googlechat  webex  weibo  tuitui  max  yuanbao")
				fmt.Println("wps-xiezuo  wps-agentspace  cloud_web")
				fmt.Println("Full field schema: GET /api/messaging/supported (while strand is running)")
				return nil
			},
		},
		{
			Name:  "workspace",
			Usage: "Show or set the conversation Agent workspace",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "agent", Usage: "Agent type (codex, claudecode, …)"},
				&cli.StringFlag{Name: "model", Usage: "Optional model override"},
				&cli.StringFlag{Name: "work-dir", Usage: "Agent work directory"},
				&cli.StringFlag{Name: "language", Usage: "zh|en|…"},
				&cli.StringFlag{Name: "user", Value: "admin", Usage: "Owner user id"},
			},
			Action: messagingWorkspaceAction,
		},
		{
			Name:  "list",
			Usage: "List bound channels",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "user", Value: "admin", Usage: "Owner user id"},
			},
			Action: messagingListAction,
		},
		{
			Name:        "add",
			Usage:       "Add a channel",
			Description: "Example: strand messaging add --type telegram --name tg1 --set token=…",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "type", Required: true, Usage: "Platform type"},
				&cli.StringFlag{Name: "name", Required: true, Usage: "Display name"},
				&cli.StringSliceFlag{Name: "set", Usage: "Credential key=value (repeatable)"},
				&cli.BoolFlag{Name: "chat", Value: true, Usage: "Enable conversation"},
				&cli.BoolFlag{Name: "alerts", Value: true, Usage: "Enable alerts"},
				&cli.StringFlag{Name: "allow-from", Usage: "Optional sender allowlist"},
				&cli.StringFlag{Name: "bound-peer", Usage: "Alert peer / chat id"},
				&cli.StringFlag{Name: "user", Value: "admin", Usage: "Owner user id"},
			},
			Action: messagingAddAction,
		},
		{
			Name:      "rm",
			Usage:     "Remove a channel",
			ArgsUsage: "<id>",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "user", Value: "admin", Usage: "Owner user id"},
			},
			Action: messagingRmAction,
		},
		{
			Name:  "reload",
			Usage: "How to hot-reload channels on a running instance",
			Action: func(c *cli.Context) error {
				fmt.Println("Hot reload: POST /api/messaging/reload (authenticated)")
				fmt.Println("Or: strand stop && strand start")
				return nil
			},
		},
	},
}

func openMessagingStore(c *cli.Context) (*store.Store, settings.Settings, error) {
	cfg, err := loadSettings(c)
	if err != nil {
		return nil, cfg, err
	}
	if err := os.MkdirAll(cfg.Data.Dir, 0o755); err != nil {
		return nil, cfg, err
	}
	dbCfg := store.DBConfig{
		Type:     store.DBType(cfg.Database.Type),
		Path:     cfg.Database.Path,
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.Name,
		SSLMode:  cfg.Database.SSLMode,
	}
	if dbCfg.Type == store.DBTypeSQLite {
		_ = os.MkdirAll(filepath.Dir(dbCfg.Path), 0o755)
	}
	st, err := store.NewWithConfig(dbCfg)
	return st, cfg, err
}

func messagingWorkspaceAction(c *cli.Context) error {
	st, cfg, err := openMessagingStore(c)
	if err != nil {
		return err
	}
	defer st.Close()
	userID := c.String("user")
	ws, err := st.Messaging().GetWorkspace(userID)
	if err != nil {
		return err
	}
	changed := false
	if v := c.String("agent"); v != "" {
		ws.AgentType = v
		changed = true
	}
	if v := c.String("model"); c.IsSet("model") {
		ws.AgentModel = v
		changed = true
	}
	if v := c.String("work-dir"); v != "" {
		ws.WorkDir = v
		changed = true
	}
	if v := c.String("language"); v != "" {
		ws.Language = v
		changed = true
	}
	if ws.WorkDir == "" {
		ws.WorkDir = filepath.Join(cfg.Data.Dir, "workspace", "messaging")
	}
	if changed {
		if err := st.Messaging().SaveWorkspace(ws); err != nil {
			return err
		}
		fmt.Println("workspace saved")
	}
	b, _ := json.MarshalIndent(ws, "", "  ")
	fmt.Println(string(b))
	return nil
}

func messagingListAction(c *cli.Context) error {
	st, _, err := openMessagingStore(c)
	if err != nil {
		return err
	}
	defer st.Close()
	rows, err := st.Messaging().ListChannels(c.String("user"))
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		fmt.Println("(no channels)")
		return nil
	}
	for _, row := range rows {
		fmt.Printf("%s  %-12s  %-16s  chat=%v alerts=%v enabled=%v  status=%s\n",
			row.ID, row.PlatformType, row.Name, row.ChatEnabled, row.AlertsEnabled, row.Enabled, row.Status)
	}
	return nil
}

func messagingAddAction(c *cli.Context) error {
	st, _, err := openMessagingStore(c)
	if err != nil {
		return err
	}
	defer st.Close()
	creds := map[string]string{}
	for _, pair := range c.StringSlice("set") {
		k, v, ok := strings.Cut(pair, "=")
		if !ok || strings.TrimSpace(k) == "" {
			return fmt.Errorf("invalid --set %q (want key=value)", pair)
		}
		creds[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	ch := &store.MessagingChannel{
		ID:            uuid.NewString(),
		UserID:        c.String("user"),
		Name:          c.String("name"),
		PlatformType:  c.String("type"),
		ChatEnabled:   c.Bool("chat"),
		AlertsEnabled: c.Bool("alerts"),
		Enabled:       true,
		AllowFrom:     c.String("allow-from"),
		BoundPeer:     c.String("bound-peer"),
		Status:        "stopped",
	}
	if err := ch.SetCredentialMap(creds); err != nil {
		return err
	}
	if err := st.Messaging().CreateChannel(ch); err != nil {
		return err
	}
	fmt.Printf("created %s (%s)\n", ch.ID, ch.PlatformType)
	fmt.Println("Apply with: strand stop && strand start   (or POST /api/messaging/reload)")
	return nil
}

func messagingRmAction(c *cli.Context) error {
	id := c.Args().First()
	if id == "" {
		return fmt.Errorf("channel id required")
	}
	st, _, err := openMessagingStore(c)
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.Messaging().DeleteChannel(c.String("user"), id); err != nil {
		return err
	}
	fmt.Println("deleted", id)
	return nil
}
