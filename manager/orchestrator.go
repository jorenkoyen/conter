package manager

import (
	"context"
	"errors"

	"github.com/jorenkoyen/conter/manager/db"
	"github.com/jorenkoyen/conter/manager/docker"
	"github.com/jorenkoyen/conter/manifest"
	"github.com/jorenkoyen/go-logger"
	"github.com/jorenkoyen/go-logger/log"
)

type Orchestrator struct {
	logger   *logger.Logger
	Database *db.Client
	Docker   *docker.Client
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		logger: log.WithName("orchestrator"),
	}
}

// ApplyManifest will create all the required resources for having the full manifest running.
func (o *Orchestrator) ApplyManifest(ctx context.Context, manifest *manifest.Project) error {

	// create network (if not exists)
	_, err := o.Docker.CreateNetworkIfNotExists(ctx, manifest.Name)
	if err != nil {
		o.logger.Warningf("Failed to create docker network: %v", err)
		return err
	}

	// create services

	// save to database
	o.logger.Infof("Successfully applied manifiest for project=%s", manifest.Name)
	return o.Database.SaveManifest(manifest)
}

// RemoveManifest will remove the resources associated to the manifest.
func (o *Orchestrator) RemoveManifest(ctx context.Context, manifest *manifest.Project) error {
	return errors.New("not implemented")
}
