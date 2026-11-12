package config

import "strand/pkg/clean"

// Develop indicates whether the application is running in development mode.
var Develop = false

const (
	EnvProd    = "prod"
	EnvUnsafe  = "unsafe"
	EnvDebug   = "debug"
	EnvTrace   = "trace"
	EnvDemo    = "demo"
	EnvSponsor = "sponsor"
	EnvDevelop = "develop"
	EnvTest    = "test"
)

func EnvVar(flag string) string {
	return clean.EnvVar(flag)
}

func EnvVars(flags ...string) []string {
	return clean.EnvVars(flags...)
}
