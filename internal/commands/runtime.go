package commands

import (
	"fmt"

	"strand/internal/settings"

	"github.com/urfave/cli/v2"
)

func configPathFrom(ctx *cli.Context) string {
	if ctx == nil {
		return ""
	}
	if ctx.IsSet("config") {
		return ctx.String("config")
	}
	if ctx.App != nil {
		return ctx.String("config")
	}
	return ""
}

func loadSettings(ctx *cli.Context) (settings.Settings, error) {
	return settings.Load(configPathFrom(ctx))
}

func loadSettingsStrict(ctx *cli.Context) (settings.Settings, error) {
	return settings.MustExistLoad(configPathFrom(ctx))
}

// finish prints a multiline reply itself. Returning the text as a normal error
// would collapse its line breaks in the top-level handler.
func finish(text string, err error) error {
	if text == "" && err != nil {
		text = err.Error()
	}
	if text != "" {
		fmt.Println(text)
	}
	if err != nil {
		return cli.Exit("", 1)
	}
	return nil
}
