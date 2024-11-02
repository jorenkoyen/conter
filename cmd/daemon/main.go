package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/jorenkoyen/conter/proxy"
	"github.com/jorenkoyen/conter/version"
	"io"
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

	legolog "github.com/go-acme/lego/v4/log"
	defaultLog "log"
)

func run(ctx context.Context, args []string) error {
	// disable default 'log' -> lego uses this internally
	legolog.Logger = defaultLog.New(io.Discard, "", defaultLog.LstdFlags)

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
	database := db.NewClient(opts.DatabaseFile)
	defer database.Close()

	// create docker client
	dckr := docker.NewClient()
	defer dckr.Close()

	// create certificate manager
	manager.LetsEncryptDirectoryUrl = "https://localhost:14000/dir" // TODO: change this based on configuration
	manager.InsecureDirectory = true                                // TODO: local test
	certificateManager := manager.NewCertificateManger(database, opts.ACME.Email)

	// create ingress manager
	ingressManager := manager.NewIngressManager()
	ingressManager.Database = database
	ingressManager.CertificateManager = certificateManager

	// create orchestrator
	containerManager := manager.NewContainerManager()
	containerManager.Database = database
	containerManager.Docker = dckr
	containerManager.IngressManager = ingressManager

	// create proxy
	rp := proxy.NewServer()
	rp.IngressManager = ingressManager
	rp.CertificateManager = certificateManager
	rp.SetLogLevel(logger.LevelInfo)

	// start HTTP proxy
	go func() {
		err := rp.ListenForHTTP(ctx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start HTTP proxy: %v", err)
		}
	}()

	// start HTTPS proxy
	go func() {
		err := rp.ListenForHTTPS(ctx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start HTTPS proxy: %v", err)
		}
	}()

	// create HTTP server
	srv := server.NewServer(opts.HTTP.ManagementAddress)
	srv.ContainerManager = containerManager
	srv.CertificateManager = certificateManager

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
