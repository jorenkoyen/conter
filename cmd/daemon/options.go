package main

import (
	"flag"
	"github.com/jorenkoyen/go-logger"
)

type Options struct {
	Log struct {
		Pretty bool
		Level  logger.Level
	}
	HTTP struct {
		ManagementAddress string
	}
	ACME struct {
		Email     string
		Directory string
		Insecure  bool
	}

	DatabaseFile string
}

func ParseOptions(args []string) Options {
	opts := Options{}

	var lvl string // log level (needs to be parsed)

	fs := flag.NewFlagSet("conter", flag.ExitOnError)
	fs.BoolVar(&opts.Log.Pretty, "log-pretty", false, "If the log output should be pretty formatted")
	fs.StringVar(&lvl, "log-level", "info", "The log level that should be applied for the application")
	fs.StringVar(&opts.HTTP.ManagementAddress, "address", "127.0.0.1:6640", "The HTTP management address")
	fs.StringVar(&opts.ACME.Email, "acme-email", "", "The email address for the owner of the ACME certificates")
	fs.StringVar(&opts.ACME.Directory, "acme-directory", "", "The ACME directory URL to use when requesting certificates")
	fs.BoolVar(&opts.ACME.Insecure, "acme-insecure", false, "If the ACME directory is to be considered insecure")
	fs.StringVar(&opts.DatabaseFile, "database", "/var/lib/conter/state.db", "The path to the database file")
	_ = fs.Parse(args) // exit on error -> error can be ignored

	opts.Log.Level = logger.ParseLevel(lvl)
	return opts
}
