package config

import "github.com/urfave/cli/v2"

// Flags configures the global command-line interface (CLI) parameters.
var Flags = CliFlags{
	{
		Flag: &cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Path to config file (default: $STRAND_CONFIG or ~/.strand/config.toml)",
			EnvVars: []string{"STRAND_CONFIG"},
		},
	},
}
