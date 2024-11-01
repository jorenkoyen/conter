package manager

import (
	"context"
	"errors"
	"fmt"
	"github.com/jorenkoyen/conter/manager/db"
	"github.com/jorenkoyen/conter/manager/docker"
	"github.com/jorenkoyen/conter/model"
	"github.com/jorenkoyen/go-logger"
	"github.com/jorenkoyen/go-logger/log"
)

type Container struct {
	logger         *logger.Logger
	Database       *db.Client
	Docker         *docker.Client
	IngressManager *IngressManager
}

func NewContainerManager() *Container {
	return &Container{
		logger: log.WithName("container-mgr"),
	}
}

// FindProject will retrieve the stored project from the database if it exists.
func (o *Container) FindProject(name string) *model.Project {
	p, err := o.Database.GetProjectByName(name)
	if err != nil {
		if !errors.Is(err, db.ErrItemNotFound) {
			o.logger.Warningf("Failure when trying to retrieve project with name=%s: %v", name, err)
		}

		return nil
	}
	return p
}

// ApplyProject will create all the required resources for having the full project running.
func (o *Container) ApplyProject(ctx context.Context, project *model.Project) error {
	// create network (if not exists)
	net, err := o.Docker.CreateNetworkIfNotExists(ctx, project.Name)
	if err != nil {
		return err
	}

	// create services
	routes := make([]string, 0, len(project.Services))
	for _, service := range project.Services {
		container, err := o.applyService(ctx, service, net)
		if err != nil {
			return fmt.Errorf("failed to create service %s: %w", service.Name, err)
		}

		// check if we need to register any routes
		if service.HasIngress() {
			ing := service.Ingress
			opts := RegisterRouteOptions{Challenge: ing.SslChallenge, Project: project.Name, Service: service.Name}
			err = o.IngressManager.RegisterRoute(ctx, ing.Domain, container.Endpoint, opts)
			if err != nil {
				return fmt.Errorf("failed to register ingress route for %s: %w", ing.Domain, err)
			}

			// append domain to routes
			routes = append(routes, ing.Domain)
		}
	}

	// remove unused routes for project
	err = o.IngressManager.RemoveUnusedRoutes(project.Name, routes)
	if err != nil {
		return fmt.Errorf("failed to remove unused routes for %s: %w", project.Name, err)
	}

	// save to database
	o.logger.Infof("Successfully applied project for project=%s", project.Name)
	return o.Database.SaveProject(project)
}

// RemoveProject will remove the resources associated to the project.
func (o *Container) RemoveProject(ctx context.Context, project *model.Project) error {
	// delete services
	for _, service := range project.Services {
		err := o.removeService(ctx, service, project.Name)
		if err != nil {
			return fmt.Errorf("failed to remove service %s: %w", service.Name, err)
		}
	}

	// delete network
	err := o.Docker.DeleteNetwork(ctx, project.Name)
	if err != nil {
		return err
	}

	o.logger.Infof("Successfully removed project for project=%s", project.Name)
	return o.Database.RemoveProject(project.Name)
}

// removeService will remove the specified service from the system.
func (o *Container) removeService(ctx context.Context, service model.Service, project string) error {
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
func (o *Container) applyService(ctx context.Context, service model.Service, network *docker.Network) (*docker.Container, error) {
	// see if we can find an existing container
	canonical := fmt.Sprintf("%s_%s", network.Name, service.Name)
	container := o.Docker.FindContainer(ctx, canonical)
	if container != nil {
		if container.ConfigHash != service.CalculateConfigurationHash() {
			o.logger.Warningf("Recycling container with id=%s, configuration differs from service", container.ID)

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

			// inspect again -> required to now deployed port
			container = o.Docker.FindContainer(ctx, container.ID)
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
