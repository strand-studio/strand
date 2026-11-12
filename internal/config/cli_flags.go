package config

import (
	"strings"

	"github.com/urfave/cli/v2"
)

func firstName(names string) string {
	if n := strings.Split(names, ","); len(n) > 0 {
		return strings.TrimSpace(n[0])
	}
	return ""
}

// DeprecatedFlagMarker is what a flag's usage text carries when the option still works but is
// no longer offered. It is rendered as emphasis in the generated reference.
const DeprecatedFlagMarker = "*deprecated*"

// CliFlags represents a list of command-line parameters.
type CliFlags []CliFlag

func (f CliFlags) Cli() (result []cli.Flag) {
	result = make([]cli.Flag, 0, len(f))

	for _, flag := range f {
		result = append(result, flag.Flag)
	}

	return result
}
