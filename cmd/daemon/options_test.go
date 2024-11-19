package main

import (
	"bytes"
	"github.com/jorenkoyen/go-logger"
	"testing"
)

func AssertEquals(t *testing.T, expected interface{}, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Errorf("expected: %v, actual: %v", expected, actual)
	}
}

func TestCheckConfig_valid(t *testing.T) {
	valid := `
log_level  		= "info"
log_pretty 		= true
listen_address 	= "127.0.0.1:6440"

[acme]
email 			= "user@example.com"
directory_url 	= "https://acme.com/directory"
insecure 		= false

[data]
directory = "/var/lib/conter"

[proxy]
http_listen_address 	= "0.0.0.0:80"
https_listen_address 	= "0.0.0.0:443"
`
	buf := bytes.NewBufferString(valid)
	config, err := ReadConfig(buf)
	if err != nil {
		t.Errorf("Failed to read configuration file: %v", err)
		t.FailNow()
	}

	// general
	AssertEquals(t, logger.LevelInfoValue, config.LogLevel)
	AssertEquals(t, true, config.LogPretty)
	AssertEquals(t, "127.0.0.1:6440", config.ListenAddress)

	// acme
	AssertEquals(t, "user@example.com", config.Acme.Email)
	AssertEquals(t, "https://acme.com/directory", config.Acme.DirectoryUrl)
	AssertEquals(t, false, config.Acme.Insecure)

	// data
	AssertEquals(t, "/var/lib/conter", config.Data.Directory)

	// proxy
	AssertEquals(t, "0.0.0.0:80", config.Proxy.HttpListenAddress)
	AssertEquals(t, "0.0.0.0:443", config.Proxy.HttpsListenAddress)
}

func TestCheckConfig_invalid(t *testing.T) {
	invalid := `
log_level  		= "info"
log_pretty 		= true
listen_address 	= ""

[acme]
email 			= "user@example.com"
directory_url 	= ""
insecure 		= false

[data]
directory = ""

[proxy]
http_listen_address 	= ""
https_listen_address 	= ""
`
	buf := bytes.NewBufferString(invalid)
	_, err := ReadConfig(buf)
	if err == nil {
		t.Errorf("Configuration should not be considered valid: %v", err)
		t.FailNow()
	}
}

func TestCheckConfig_default(t *testing.T) {
	buf := bytes.NewBufferString("")
	config, err := ReadConfig(buf)
	if err != nil {
		t.Errorf("Failed to read configuration file: %v", err)
		t.FailNow()
	}

	// general
	AssertEquals(t, logger.LevelInfoValue, config.LogLevel)
	AssertEquals(t, false, config.LogPretty)
	AssertEquals(t, "127.0.0.1:6440", config.ListenAddress)

	// acme
	AssertEquals(t, "", config.Acme.Email)
	AssertEquals(t, "https://acme-staging-v02.api.letsencrypt.org/directory", config.Acme.DirectoryUrl)
	AssertEquals(t, false, config.Acme.Insecure)

	// data
	AssertEquals(t, "/var/lib/conter", config.Data.Directory)

	// proxy
	AssertEquals(t, "0.0.0.0:80", config.Proxy.HttpListenAddress)
	AssertEquals(t, "0.0.0.0:443", config.Proxy.HttpsListenAddress)
}

func TestParse(t *testing.T) {

	{
		// empty
		var args []string
		opts, err := Parse(args)
		if err != nil {
			t.Errorf("Should not have failed trying to parse empty args: %v", err)
		} else {
			AssertEquals(t, "/etc/conter/config.toml", opts.Config)
			AssertEquals(t, false, opts.ValidateConfig)
		}
	}

	{
		// with explicit config
		args := []string{"--config=/some/other/path/config.toml"}
		opts, err := Parse(args)
		if err != nil {
			t.Errorf("Should not have failed trying to parse explicit config location: %v", err)
		} else {
			AssertEquals(t, "/some/other/path/config.toml", opts.Config)
			AssertEquals(t, false, opts.ValidateConfig)
		}
	}

	{
		// with explicit validate
		args := []string{"--validate-config"}
		opts, err := Parse(args)
		if err != nil {
			t.Errorf("Should not have failed trying to parse explicit config validation: %v", err)
		} else {
			AssertEquals(t, "/etc/conter/config.toml", opts.Config)
			AssertEquals(t, true, opts.ValidateConfig)
		}
	}

	{
		// with all explicit
		args := []string{"--validate-config", "--config", "/some/path/config.toml"}
		opts, err := Parse(args)
		if err != nil {
			t.Errorf("Should not have failed trying to parse explicit options: %v", err)
		} else {
			AssertEquals(t, "/some/path/config.toml", opts.Config)
			AssertEquals(t, true, opts.ValidateConfig)
		}
	}

}
