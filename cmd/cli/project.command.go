package main

import "github.com/urfave/cli/v2"

func project() *cli.Command {
	return &cli.Command{
		Name:  "project",
		Usage: "Manage projects",
		Subcommands: []*cli.Command{
			{
				Name:  "ls",
				Usage: "List projects",
			},
			{
				Name:  "apply",
				Usage: "Apply a project configuration to the system",
			},
			{
				Name:  "rm",
				Usage: "Remove a project",
			},
			{
				Name:  "inspect",
				Usage: "Inspect the information of a project",
			},
		},
	}
}
