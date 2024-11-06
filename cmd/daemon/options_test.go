package main

import (
	"github.com/jorenkoyen/go-logger"
	"testing"
)

func AssertEquals(t *testing.T, expected interface{}, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Errorf("expected: %v, actual: %v", expected, actual)
	}
}

func TestParseOptions(t *testing.T) {
	args := []string{
		"--log-level", "debug",
		"--log-pretty",
		"--address", "127.0.0.1:1234",
		"--database", "/var/lib/conter/state.db",
		"--acme-email", "user@example.com",
		"--acme-directory", "https://localhost:14000/dir",
		"--acme-insecure",
	}

	opts := ParseOptions(args)
	AssertEquals(t, logger.LevelDebug, opts.Log.Level)
	AssertEquals(t, true, opts.Log.Pretty)
	AssertEquals(t, "127.0.0.1:1234", opts.HTTP.ManagementAddress)
	AssertEquals(t, "/var/lib/conter/state.db", opts.DatabaseFile)
	AssertEquals(t, "user@example.com", opts.ACME.Email)
	AssertEquals(t, "https://localhost:14000/dir", opts.ACME.Directory)
	AssertEquals(t, true, opts.ACME.Insecure)
}

func TestParseOptionsDefaults(t *testing.T) {
	args := make([]string, 0)
	opts := ParseOptions(args)
	AssertEquals(t, logger.LevelInfo, opts.Log.Level)
	AssertEquals(t, false, opts.Log.Pretty)
	AssertEquals(t, "127.0.0.1:6640", opts.HTTP.ManagementAddress)
	AssertEquals(t, "/var/lib/conter/state.db", opts.DatabaseFile)
	AssertEquals(t, "", opts.ACME.Email)
	AssertEquals(t, "https://acme-staging-v02.api.letsencrypt.org/directory", opts.ACME.Directory)
	AssertEquals(t, false, opts.ACME.Insecure)
}
