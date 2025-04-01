package cli

import (
	"sort"

	"github.com/urfave/cli/v2"
)

// SetupCLI initialize cli
func SetupCLI() *cli.App {
	app := &cli.App{
		// List your commands here
		Name:    "mtcli",
		Usage:   "Mediatool CLI",
		Version: "1.0.5",
		Commands: []*cli.Command{
			BuildCommand(),
			WatchCommand(),
			ListCommand(),
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	// Return our cli
	return app
}
