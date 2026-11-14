package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

// ResourceCommand probes model/agent/platform/exchange via the running API.
var ResourceCommand = &cli.Command{
	Name:        "resource",
	Usage:       "Test unified resources (model / agent / platform / exchange)",
	Description: "Calls the running API. Requires STRAND_TOKEN or --token (same as the console JWT / API token).",
	Subcommands: []*cli.Command{
		{
			Name:        "test",
			Usage:        "POST /api/resources/{kind}/{id}/test",
			Description:  "Kinds: model | agent | platform | exchange",
			ArgsUsage:   "<kind> <id>",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "api", Value: envOr("STRAND_API", "http://127.0.0.1:8080"), Usage: "API base URL"},
				&cli.StringFlag{Name: "token", Value: envOr("STRAND_TOKEN", ""), Usage: "Bearer token (or STRAND_TOKEN)"},
			},
			Action: resourceTestAction,
		},
	},
}

func resourceTestAction(c *cli.Context) error {
	kind := strings.ToLower(strings.TrimSpace(c.Args().Get(0)))
	id := strings.TrimSpace(c.Args().Get(1))
	if kind == "" || id == "" {
		return cli.Exit("usage: strand resource test <model|agent|platform|exchange> <id>", 2)
	}
	base := strings.TrimRight(c.String("api"), "/")
	token := strings.TrimSpace(c.String("token"))
	if token == "" {
		return cli.Exit("token required (--token or STRAND_TOKEN)", 2)
	}
	url := fmt.Sprintf("%s/api/resources/%s/%s/test", base, kind, id)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var pretty bytes.Buffer
	if json.Indent(&pretty, body, "", "  ") == nil {
		fmt.Println(pretty.String())
	} else {
		fmt.Println(string(body))
	}
	if resp.StatusCode >= 300 {
		return cli.Exit(fmt.Sprintf("test failed (http %d)", resp.StatusCode), 1)
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
