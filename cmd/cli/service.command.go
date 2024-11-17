package main

import "github.com/urfave/cli/v2"

func service() *cli.Command {
	return &cli.Command{
		Name:  "service",
		Usage: "Manage the systemctl service for Conter",
		Subcommands: []*cli.Command{
			{
				Name:  "logs",
				Usage: "Show the logs of Conter",
			},
			{
				Name:  "status",
				Usage: "Show the current status of the Conter service",
			},
			{
				Name:  "install",
				Usage: "Install the Conter systemctl service",
			},
		},
	}
}
