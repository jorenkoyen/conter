package main

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
)

func certificate() *cli.Command {
	return &cli.Command{
		Name:    "certificate",
		Aliases: []string{"cert"},
		Usage:   "Manage certificates",
		Subcommands: []*cli.Command{
			{
				Name:   "ls",
				Usage:  "List certificates",
				Action: listCertificateHandler,
			},
			{
				Name:      "renew",
				Usage:     "Renew the certificate for the specified domain",
				Action:    renewCertificateHandler,
				Args:      true,
				ArgsUsage: "[domain]",
			},
			{
				Name:      "inspect",
				Usage:     "Inspect the information of a certificate",
				Action:    inspectCertificateHandler,
				Args:      true,
				ArgsUsage: "[domain]",
			},
		},
	}
}

func listCertificateHandler(c *cli.Context) error {
	client, err := clientFromContext(c)
	if err != nil {
		return err
	}

	certificates, err := client.CertificateList(c.Context)
	if err != nil {
		return fmt.Errorf("failed to list certificates: %w", err)
	}

	// write certificates output
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"DOMAIN", "CHALLENGE", "EXPIRY", "ISSUER"})
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetNoWhiteSpace(true)
	table.SetTablePadding("    ")

	for _, cert := range certificates {
		data := []string{
			cert.Domain,
			string(cert.Challenge),
			cert.Meta.Expiry.Format(time.RFC1123),
			cert.Meta.Issuer,
		}

		expired := time.Now().After(cert.Meta.Expiry)
		if expired {
			table.Rich(data, []tablewriter.Colors{
				{}, // domain
				{}, // challenge
				{tablewriter.Bold, tablewriter.FgHiRedColor}, // expiry
				{}, // issuer
			})
		} else {
			table.Append(data)
		}
	}

	table.Render()
	return nil
}

func renewCertificateHandler(c *cli.Context) error {
	domain := c.Args().First()
	if domain == "" {
		return errors.New("domain argument is required")
	}

	client, err := clientFromContext(c)
	if err != nil {
		return err
	}

	err = client.CertificateRenew(c.Context, domain)
	if err != nil {
		return fmt.Errorf("failed to renew certificate: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Certificate for %s is being renewed\n", domain)
	return nil
}

func inspectCertificateHandler(c *cli.Context) error {
	domain := c.Args().First()
	if domain == "" {
		return errors.New("domain argument is required")
	}

	client, err := clientFromContext(c)
	if err != nil {
		return err
	}

	cert, err := client.CertificateInspect(c.Context, domain)
	if err != nil {
		return fmt.Errorf("failed to retrieve certificate: %w", err)
	}

	// write certificate information
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "%s:\t%s\n", "Domain", cert.Domain)
	fmt.Fprintf(writer, "%s:\t%s\n", "Challenge", string(cert.Challenge))
	fmt.Fprintf(writer, "%s:\t%s\n", "Since", cert.Meta.Since.Format(time.RFC1123))
	fmt.Fprintf(writer, "%s:\t%s\n", "Expiry", cert.Meta.Expiry.Format(time.RFC1123))
	fmt.Fprintf(writer, "\nMeta:\n")
	fmt.Fprintf(writer, "  %s:\t%s\n", "Subject", cert.Meta.Subject)
	fmt.Fprintf(writer, "  %s:\t%s\n", "Issuer", cert.Meta.Issuer)
	fmt.Fprintf(writer, "  %s:\t%s\n", "Serial Number", cert.Meta.SerialNumber)
	fmt.Fprintf(writer, "  %s:\t%s\n", "Signature Algorithm", cert.Meta.SignatureAlgorithm)
	fmt.Fprintf(writer, "  %s:\t%s\n", "Public Algorithm", cert.Meta.PublicAlgorithm)

	return writer.Flush()
}
