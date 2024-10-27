package main

import "github.com/jorenkoyen/go-logger"

type Options struct {
	Log struct {
		Pretty bool
		Level  logger.Level
	}
	HTTP struct {
		ManagementAddress string
	}

	DatabaseFile string
}

func ParseOptions(args []string) Options {
	opts := Options{}
	opts.Log.Pretty = true
	opts.Log.Level = logger.LevelTrace
	opts.HTTP.ManagementAddress = "127.0.0.1:6440"
	opts.DatabaseFile = "conter.db"
	return opts
}
