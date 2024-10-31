package manager

import (
	"context"
	"errors"
	"fmt"
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
	Ingress  *Ingress
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		logger: log.WithName("orchestrator"),
	}
}

// FindManifest will retrieve the stored manifest from the database if it exists.
func (o *Orchestrator) FindManifest(name string) *manifest.Project {
	_manifest, err := o.Database.GetManifestByName(name)
	if err != nil {
		if !errors.Is(err, db.ErrItemNotFound) {
			o.logger.Warningf("Failure when trying to retrieve manifest with name=%s: %v", name, err)
		}

		return nil
	}
	return _manifest
}

// ApplyManifest will create all the required resources for having the full manifest running.
func (o *Orchestrator) ApplyManifest(ctx context.Context, manifest *manifest.Project) error {
	// create network (if not exists)
	net, err := o.Docker.CreateNetworkIfNotExists(ctx, manifest.Name)
	if err != nil {
		return err
	}

	// create services
	routes := make([]string, 0, len(manifest.Services))
	for _, service := range manifest.Services {
		container, err := o.applyService(ctx, service, net)
		if err != nil {
			return fmt.Errorf("failed to create service %s: %w", service.Name, err)
		}

		// check if we need to register any routes
		if service.HasIngress() {
			ing := service.Ingress
			opts := RegisterRouteOptions{Challenge: ing.SslChallenge, Project: manifest.Name, Service: service.Name}
			err = o.Ingress.RegisterRoute(ctx, ing.Domain, container.Endpoint, opts)
			if err != nil {
				return fmt.Errorf("failed to register ingress route for %s: %w", ing.Domain, err)
			}

			// append domain to routes
			routes = append(routes, ing.Domain)
		}
	}

	// remove unused routes for manifest
	err = o.Ingress.RemoveUnusedRoutes(manifest.Name, routes)
	if err != nil {
		return fmt.Errorf("failed to remove unused routes for %s: %w", manifest.Name, err)
	}

	// save to database
	o.logger.Infof("Successfully applied manifest for project=%s", manifest.Name)
	return o.Database.SaveManifest(manifest)
}

// RemoveManifest will remove the resources associated to the manifest.
func (o *Orchestrator) RemoveManifest(ctx context.Context, manifest *manifest.Project) error {
	// delete services
	for _, service := range manifest.Services {
		err := o.removeService(ctx, service, manifest.Name)
		if err != nil {
			return fmt.Errorf("failed to remove service %s: %w", service.Name, err)
		}
	}

	// delete network
	err := o.Docker.DeleteNetwork(ctx, manifest.Name)
	if err != nil {
		return err
	}

	o.logger.Infof("Successfully removed manifest for project=%s", manifest.Name)
	return o.Database.RemoveManifest(manifest.Name)
}

// removeService will remove the specified service from the system.
func (o *Orchestrator) removeService(ctx context.Context, service manifest.Service, project string) error {
	canonical := fmt.Sprintf("%s_%s", project, service.Name)
	container := o.Docker.FindContainer(ctx, canonical)
	if container == nil {
		o.logger.Debugf("No container found with name=%s, nothing to remove", canonical)
		return nil
	}

	// remove container
	return o.Docker.RemoveContainer(ctx, container.ID)
}

// applyService will create or update the service.
func (o *Orchestrator) applyService(ctx context.Context, service manifest.Service, network *docker.Network) (*docker.Container, error) {
	// see if we can find an existing container
	canonical := fmt.Sprintf("%s_%s", network.Name, service.Name)
	container := o.Docker.FindContainer(ctx, canonical)
	if container != nil {
		if container.ConfigHash != service.CalculateConfigurationHash() {
			o.logger.Warningf("Recycling container with id=%s, configuration differs from manifest", container.ID)

			err := o.Docker.RemoveContainer(ctx, container.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to recylce container: %v", err)
			}

		} else if !container.IsRunning() {
			o.logger.Warningf("Container is not yet started for service=%s, starting now", service.Name)
			err := o.Docker.StartContainer(ctx, container.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to start container: %w", err)
			}

			return container, nil
		} else {
			o.logger.Debugf("Container already exists for service=%s", service.Name)
			return container, nil
		}
	}

	// translate source to valid docker image (build may be required)
	image, err := o.getDockerImageFromSource(ctx, service)
	if err != nil {
		return nil, fmt.Errorf("unable to extract docker image source: %w", err)
	}

	// create container image
	container, err = o.Docker.CreateContainer(ctx, service, network, canonical, image)
	if err != nil {
		return nil, fmt.Errorf("unable to create container: %w", err)
	}

	// start container
	err = o.Docker.StartContainer(ctx, container.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	o.logger.Infof("Successfully created docker container with id=%s for service=%s", container.ID, service.Name)
	return container, nil
}
