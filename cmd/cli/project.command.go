package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jorenkoyen/conter/api"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
	"os"
	"strings"
	"text/tabwriter"
)

func project() *cli.Command {
	return &cli.Command{
		Name:  "project",
		Usage: "Manage projects",
		Subcommands: []*cli.Command{
			{
				Name:   "ls",
				Usage:  "List projects",
				Action: listProjectHandler,
			},
			{
				Name:  "apply",
				Usage: "Apply a project configuration to the system",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "file",
						Aliases: []string{"f"},
						Usage:   "Specifies the `project.json` file to use when applying the configuration",
						Value:   "project.json",
					},
				},
				Action: applyProjectHandler,
			},
			{
				Name:      "rm",
				Usage:     "Remove a project",
				Action:    removeProjectHandler,
				Args:      true,
				ArgsUsage: "[name]",
			},
			{
				Name:      "inspect",
				Usage:     "Inspect the information of a project",
				Action:    inspectProjectHandler,
				Args:      true,
				ArgsUsage: "[name]",
			},
		},
	}
}

func listProjectHandler(c *cli.Context) error {
	client, err := clientFromContext(c)
	if err != nil {
		return err
	}

	projects, err := client.ProjectList(c.Context)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	// write certificates output
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"NAME", "RUNNING", "SERVICES"})
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetNoWhiteSpace(true)
	table.SetTablePadding("    ")

	for _, p := range projects {
		table.Append([]string{
			p.Name,
			fmt.Sprint(p.Running),
			strings.Join(p.Services, ","),
		})
	}

	table.Render()
	return nil
}

func applyProjectHandler(c *cli.Context) error {
	content, err := os.ReadFile(c.String("file"))
	if err != nil {
		return fmt.Errorf("failed to read project file: %w", err)
	}

	var cmd api.ProjectApplyCommand
	if err = json.Unmarshal(content, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal project.json: %w", err)
	}

	client, err := clientFromContext(c)
	if err != nil {
		return err
	}

	p, err := client.ProjectApply(c.Context, cmd)
	if err != nil {
		return fmt.Errorf("failed to apply project: %w", err)
	}

	// write project information
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "%s:\t%s\n", "Name", p.Name)
	fmt.Fprintf(writer, "Services:\n")

	// write each service
	for _, s := range p.Services {
		fmt.Fprintf(writer, "  %s:\n", s.Name)
		fmt.Fprintf(writer, "    %s:\t%s\n", "Status", s.Status)
		fmt.Fprintf(writer, "    %s:\t%s\n", "Hash", s.Hash)

		if s.Ingress.Domain != "" {
			fmt.Fprintf(writer, "    %s:\t%s\n", "Domain", s.Ingress.Domain)
			fmt.Fprintf(writer, "    %s:\t%s\n", "Endpoint", s.Ingress.InternalEndpoint)
		}
		fmt.Fprintf(writer, "\n")
	}

	return nil
}

func removeProjectHandler(c *cli.Context) error {
	name := c.Args().First()
	if name == "" {
		return errors.New("name argument is required")
	}

	client, err := clientFromContext(c)
	if err != nil {
		return err
	}

	err = client.ProjectRemove(c.Context, name)
	if err != nil {
		return fmt.Errorf("failed to remove project: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Project %s has been removed\n", name)
	return nil
}

func inspectProjectHandler(c *cli.Context) error {
	name := c.Args().First()
	if name == "" {
		return errors.New("name argument is required")
	}

	client, err := clientFromContext(c)
	if err != nil {
		return err
	}

	p, err := client.ProjectInspect(c.Context, name)
	if err != nil {
		return fmt.Errorf("failed to inspect project: %w", err)
	}

	// write project information
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "%s:\t%s\n", "Name", p.Name)
	fmt.Fprintf(writer, "Services:\n")

	// write each service
	for _, s := range p.Services {
		fmt.Fprintf(writer, "  %s:\n", s.Name)
		fmt.Fprintf(writer, "    %s:\t%s\n", "Status", s.Status)
		fmt.Fprintf(writer, "    %s:\t%s\n", "Hash", s.Hash)

		if s.Ingress.Domain != "" {
			fmt.Fprintf(writer, "    %s:\t%s\n", "Domain", s.Ingress.Domain)
			fmt.Fprintf(writer, "    %s:\t%s\n", "Endpoint", s.Ingress.InternalEndpoint)
		}
		fmt.Fprintf(writer, "\n")
	}

	return nil
}
