package config

import "github.com/urfave/cli/v2"

// Flags configures the global command-line interface (CLI) parameters.
var Flags = CliFlags{
	{
		Flag: &cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "配置文件路径（默认: $STRAND_CONFIG 或 ~/.strand/config.toml）",
			EnvVars: []string{"STRAND_CONFIG"},
		},
	},
}
