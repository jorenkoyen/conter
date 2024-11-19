package main

import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/go-acme/lego/v4/lego"
	"github.com/jorenkoyen/go-logger"
	"io"
	"os"
	"strings"
)

// Options represents the CLI arguments.
type Options struct {
	// The location of the configuration file.
	Config string
	// Indicates if the daemon should only validate the configuration and exit.
	ValidateConfig bool
	// Indicates if the version should be printed.
	Version bool
}

// Config represents the application configuration.
type Config struct {
	LogLevel      string `toml:"log_level"`
	LogPretty     bool   `toml:"log_pretty"`
	ListenAddress string `toml:"listen_address"`

	Acme struct {
		Email        string `toml:"email"`
		DirectoryUrl string `toml:"directory_url"`
		Insecure     bool   `toml:"insecure"`
	} `toml:"acme"`

	Data struct {
		Directory string `toml:"directory"`
	} `toml:"data"`

	Proxy struct {
		HttpListenAddress  string `toml:"http_listen_address"`
		HttpsListenAddress string `toml:"https_listen_address"`
	} `toml:"proxy"`
}

// Parse will process the CLI arguments and return the parsed options.
func Parse(args []string) (*Options, error) {
	opts := &Options{}

	fs := flag.NewFlagSet("conter", flag.ContinueOnError)
	fs.StringVar(&opts.Config, "config", "/etc/conter/config.toml", "The location of the configuration file")
	fs.BoolVar(&opts.ValidateConfig, "validate-config", false, "Validate the configuration file")
	fs.BoolVar(&opts.Version, "version", false, "Print the version and exit.")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return opts, nil
}

// ReadConfigFromOpts will read the configuration based on the CLI options.
func ReadConfigFromOpts(opts *Options) (*Config, error) {
	file, err := os.Open(opts.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	return ReadConfig(file)
}

// ReadConfig will read the application configuration and return an error if not valid.
func ReadConfig(r io.Reader) (*Config, error) {
	config := new(Config)

	// configure defaults
	config.LogLevel = logger.LevelInfoValue
	config.LogPretty = false
	config.ListenAddress = "127.0.0.1:6440"
	config.Acme.Email = "" // empty by default
	config.Acme.DirectoryUrl = lego.LEDirectoryStaging
	config.Acme.Insecure = false
	config.Data.Directory = "/var/lib/conter"
	config.Proxy.HttpListenAddress = "0.0.0.0:80"
	config.Proxy.HttpsListenAddress = "0.0.0.0:443"

	_, err := toml.NewDecoder(r).Decode(config)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration: %w", err)
	}

	// check if all required fields are present
	warnings := make([]string, 0, 5)
	if config.ListenAddress == "" {
		warnings = append(warnings, "'listen_address' is required")
	}
	if config.Acme.DirectoryUrl == "" {
		warnings = append(warnings, "'acme.directory_url' is required")
	}
	if config.Data.Directory == "" {
		warnings = append(warnings, "'data.directory' is required")
	}
	if config.Proxy.HttpsListenAddress == "" {
		warnings = append(warnings, "'proxy.https_listen_address' is required")
	}
	if config.Proxy.HttpListenAddress == "" {
		warnings = append(warnings, "'proxy.http_listen_address' is required")
	}
	if len(warnings) > 0 {
		return nil, fmt.Errorf("missing properties: %s", strings.Join(warnings, ", "))
	}

	return config, nil
}
