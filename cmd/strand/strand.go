package main

import (
	"fmt"
	"os"
	"strand/internal/commands"
	"strand/internal/config"
	"strand/pkg/clean"

	"github.com/urfave/cli/v2"
)

var version = "0.1.0"

var appName = "strand"
var appAbout = "Self-hosted AI trading terminal — messaging, markets, Autopilot, and HTTP API"
var appEdition = "preview"
var appDescription = `Strand runs on your machine: Web console + API, exchange trading, and chat alerts.
Keys and funds stay local — nothing is custodial.

Quick start:
  strand config init          Create ~/.strand/config.toml and .env
  strand start                Run API (default http://127.0.0.1:8080)
  strand start -d             Detach to background (pid + log under data dir)
  strand status               Check whether the service is running
  strand messaging supported  List chat / alert platform types

Web console (separate process):
  cd web && pnpm install && pnpm run dev   → http://127.0.0.1:3000

Docs: README.md · README.zh-CN.md · docs/`
var appCopyright = ""

var Metadata = map[string]any{
	"Name":        appName,
	"About":       appAbout,
	"Edition":     appEdition,
	"Description": appDescription,
	"Copyright":   appCopyright,
	"Version":     version,
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			_, _ = fmt.Fprintf(os.Stderr, "panic: %v\n", r)
			os.Exit(1)
		}
	}()

	app := cli.NewApp()
	app.Name = appName
	app.Usage = appAbout
	app.Description = appDescription
	app.Version = version
	app.Copyright = appCopyright
	app.EnableBashCompletion = true
	app.Flags = config.Flags.Cli()
	app.Commands = commands.Strand
	app.Metadata = Metadata
	app.ExitErrHandler = commands.HandleExitErr

	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, clean.DisplayLines(fmt.Sprint(err)))
		os.Exit(commands.ExitCode(err))
	}
}
