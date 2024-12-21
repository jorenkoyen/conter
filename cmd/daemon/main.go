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

func init() {
	// disable default 'log' -> lego uses this internally
	legolog.Logger = defaultLog.New(io.Discard, "", defaultLog.LstdFlags)
}

func run(ctx context.Context, args []string) error {
	opts, err := Parse(args)
	if err != nil {
		return err
	}

	if opts.Version {
		fmt.Fprintln(os.Stdout, version.Version)
		return nil
	}

	// read config
	config, err := ReadConfigFromOpts(opts)
	if err != nil {
		return err
	}

	// exit if validate config only
	if opts.ValidateConfig {
		return nil
	}

	var formatter logger.Formatter = logger.NewTextFormatter()
	if config.LogPretty {
		// override formatter with pretty formatter
		formatter = logger.NewPrettyFormatter()
	}

	log.SetDefaultLogger(logger.NewWithOptions(logger.Options{
		Writer:    os.Stdout,
		Formatter: formatter,
		Level:     logger.ParseLevel(config.LogLevel),
	}))

	// create data directory if not exists
	if err = os.MkdirAll(config.Data.Directory, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	log.Infof("Starting conter @ version=%s [ go=%s arch=%s ]", version.Version, version.GoVersion, runtime.GOARCH)

	// listen for ctrl+c notifies
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// create database client
	database := db.NewClient(config.Data.Directory)
	defer database.Close()

	// create docker client
	dckr := docker.NewClient(config.Data.Directory)
	defer dckr.Close()

	// create certificate manager
	certificateManager := manager.NewCertificateManger(database, config.Acme.Email, config.Acme.DirectoryUrl, config.Acme.Insecure)

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

	// start HTTP proxy
	go func() {
		err := rp.ListenForHTTP(ctx, config.Proxy.HttpListenAddress)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start HTTP proxy: %v", err)
		}
	}()

	// start HTTPS proxy
	go func() {
		err := rp.ListenForHTTPS(ctx, config.Proxy.HttpsListenAddress)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start HTTPS proxy: %v", err)
		}
	}()

	// create HTTP server
	srv := server.NewServer(config.ListenAddress)
	srv.ContainerManager = containerManager
	srv.CertificateManager = certificateManager

	// start application
	if err := srv.Listen(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
