package main

import (
	"errors"
	"fmt"
	"github.com/urfave/cli/v2"
	"os"
	"os/exec"
	"runtime"
)

const (
	ServiceName = "conter.service"
)

func service() *cli.Command {
	return &cli.Command{
		Name:    "service",
		Usage:   "Manage the systemctl service for Conter",
		Aliases: []string{"svc"},
		Before: func(c *cli.Context) error {
			if runtime.GOOS != "linux" {
				return errors.New("service utilities are only available for Linux")
			}
			return nil
		},
		Subcommands: []*cli.Command{
			{
				Name:  "logs",
				Usage: "Show the logs of Conter",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "follow",
						Aliases: []string{"f"},
						Usage:   "If the logs should be followed live",
					},
				},
				Action: serviceLogsHandler,
			},
			{
				Name:   "status",
				Usage:  "Show the current status of the Conter service",
				Action: statusCheckHandler,
			},
		},
	}
}

// serviceLogsHandler runs `journalctl -u` for the service and optionally follows the logs
func serviceLogsHandler(c *cli.Context) error {
	follow := c.Bool("follow")

	// Create the base command
	args := []string{"-u", ServiceName}
	if follow {
		args = append(args, "-f") // Add the follow option
	}

	cmd := exec.Command("journalctl", args...)
	cmd.Stdout = os.Stdout // Redirects the command's stdout to the program's stdout
	cmd.Stderr = os.Stderr // Redirects the command's stderr to the program's stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to show logs for service %s: %v", ServiceName, err)
	}
	return nil
}

// statusCheckHandler runs `systemctl status` for the service and outputs the content to stdout
func statusCheckHandler(c *cli.Context) error {
	cmd := exec.Command("systemctl", "status", ServiceName)
	cmd.Stdout = os.Stdout // Redirects the command's stdout to the program's stdout
	cmd.Stderr = os.Stderr // Redirects the command's stderr to the program's stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to get status of service %s: %v", ServiceName, err)
	}
	return nil
}
