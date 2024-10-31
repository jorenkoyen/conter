package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/jorenkoyen/conter/proxy"
	"github.com/jorenkoyen/conter/version"
	"net/http"
	"os"
	"os/signal"
	"runtime"
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

	// create ingress manager
	ingress := manager.NewIngressManager()
	ingress.Database = database

	// create orchestrator
	orchestrator := manager.NewOrchestrator()
	orchestrator.Database = database
	orchestrator.Docker = dckr
	orchestrator.Ingress = ingress

	// create proxy
	rp := proxy.NewServer()
	rp.Ingress = ingress

	// start HTTP proxy
	go func() {
		err := rp.ListenForHTTP(ctx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start HTTP proxy: %v", err)
		}
	}()

	// TODO: start HTTPS proxy

	// create HTTP server
	srv := server.NewServer(opts.HTTP.ManagementAddress)
	srv.Orchestrator = orchestrator

	// start application
	log.Infof("Starting conter @ version=%s [ go=%s arch=%s ]", version.Version, version.GoVersion, runtime.GOARCH)
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
