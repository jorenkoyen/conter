package main

import (
	"fmt"
	"github.com/jorenkoyen/conter/api"
	"github.com/urfave/cli/v2"
	"os"
)

func main() {
	app := cli.App{
		Name:  "conctl",
		Usage: "CLI tool for Conter, a minimal container management system for small scale web deployments.",
		Commands: []*cli.Command{
			// [conterctl] service logs -f
			// [conterctl] service status
			// [conterctl] service install
			service(),
			// [conterctl] certificate ls
			// [conterctl] certificate renew :domain
			// [conterctl] certificate inspect :domain
			certificate(),
			// [conterctl] project ls
			// [conterctl] project apply -f :file
			// [conterctl] project rm :name
			// [conterctl] project inspect :name
			project(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

// clientFromContext will create the client for communicating with Conter based on the CLI context.
func clientFromContext(c *cli.Context) (*api.Client, error) {
	client := api.NewClientFromEnv()
	return client, nil
}
