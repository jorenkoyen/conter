package main

import (
	"encoding/json"
	"flag"
	"github.com/go-acme/lego/v4/lego"
	"github.com/jorenkoyen/go-logger"
	"os"
)

type Options struct {
	Log struct {
		Pretty bool         `json:"pretty"`
		Level  logger.Level `json:"level"`
	} `json:"log"`

	HTTP struct {
		ManagementAddress string `json:"mgmt"`
	} `json:"http"`

	ACME struct {
		Email     string `json:"email"`
		Directory string `json:"directory"`
		Insecure  bool   `json:"insecure"`
	} `json:"acme"`
}

func ParseOptions(args []string) (Options, error) {
	opts := Options{}
	opts.Log.Pretty = false
	opts.Log.Level = logger.LevelInfo
	opts.HTTP.ManagementAddress = "127.0.0.1:6440"
	opts.ACME.Directory = lego.LEDirectoryStaging // staging by default
	opts.ACME.Insecure = false

	var location string
	fs := flag.NewFlagSet("conter", flag.ExitOnError)
	fs.StringVar(&location, "config", "/etc/conter/config.json", "The location of the configuration file")
	_ = fs.Parse(args) // exit on error -> error can be ignored

	// read configuration file
	content, err := os.ReadFile(location)
	if err != nil {
		return opts, err
	}

	err = json.Unmarshal(content, &opts)
	return opts, err
}
