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
var appAbout = "AI 交易助手：消息通道 + 行情交易 + HTTP API"
var appEdition = "preview"
var appDescription = "通过微信等通讯软件查看市场、接收建议与通知，并提供客户端可用的 REST API。"
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
