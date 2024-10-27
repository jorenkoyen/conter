package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jorenkoyen/conter/manager"
	"github.com/jorenkoyen/conter/manager/db"
	"github.com/jorenkoyen/conter/manager/docker"
	"github.com/jorenkoyen/conter/server"
	"github.com/jorenkoyen/go-logger"
	"github.com/jorenkoyen/go-logger/log"
)

func run(ctx context.Context, args []string) error {
	// parse CLI options
	opts := ParseOptions(args)

	var formatter logger.Formatter = logger.NewTextFormatter()
	if opts.Log.Pretty {
		// override formatter with pretty formatter
		formatter = logger.NewPrettyFormatter()
	}
	log.SetDefaultLogger(logger.NewWithOptions(logger.Options{
		Writer:    os.Stdout,
		Formatter: formatter,
		Level:     opts.Log.Level,
	}))

	// listen for ctrl+c notifies
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// create database client
	database := db.NewClient("conter.db")
	defer database.Close()

	// create docker client
	dckr := docker.NewClient()
	defer dckr.Close()

	// create orchestrator
	orchestrator := manager.NewOrchestrator()
	orchestrator.Database = database
	orchestrator.Docker = dckr

	// create HTTP server
	srv := server.NewServer(opts.HTTP.ManagementAddress)
	srv.Orchestrator = orchestrator
	if err := srv.Listen(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
