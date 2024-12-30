package manager

import (
	"context"
	"errors"
	"fmt"
	"github.com/jorenkoyen/conter/manager/db"
	"github.com/jorenkoyen/conter/manager/docker"
	"github.com/jorenkoyen/conter/manager/source"
	"github.com/jorenkoyen/conter/manager/types"
	"github.com/jorenkoyen/go-logger"
	"github.com/jorenkoyen/go-logger/log"
	"strings"
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

// DoesProjectExist checks whether the project with the given name actually exists in the system.
func (o *Container) DoesProjectExist(name string) bool {
	services := o.Database.GetServicesForProject(name)
	return len(services) > 0
}

type ApplyProjectOptions struct {
	ProjectName string `json:"project_name"`
	Services    []struct {
		Name           string              `json:"name"`
		Source         types.Source        `json:"source"`
		Environment    map[string]string   `json:"environment"`
		IngressDomains []string            `json:"ingress_domains"`
		ContainerPort  int                 `json:"container_port"`
		Volumes        []types.Volume      `json:"volumes"`
		ChallengeType  types.ChallengeType `json:"challenge_type"`
		Quota          types.Quota         `json:"quota"`
	} `json:"services"`
}

// validate will perform the basic validation required for applying a project configuration.
func (opts *ApplyProjectOptions) validate() *types.ValidationError {
	err := new(types.ValidationError)

	if opts.ProjectName == "" {
		err.Append("project_name", "Project name is required")
	}

	if len(opts.Services) == 0 {
		err.Append("services", "At least one service is required")
		return err
	}

	for i, service := range opts.Services {
		prefix := fmt.Sprintf("services[%d].", i)

		if service.Name == "" {
			err.Append(prefix+"name", "Services name is required")
		}

		if service.Source.Type == "" {
			err.Append(prefix+"source.type", "Source type is required")

		} else if service.Source.Type != "docker" {
			err.Appendf(prefix+"source.type", "Source type=%s is not supported", service.Source.Type)
		}

		if service.Source.URI == "" {
			err.Append(prefix+"source.uri", "Source URI is required")
		}

		if len(service.IngressDomains) > 0 {
			// indication that service should be exposed
			if service.ChallengeType != types.ChallengeTypeHTTP && service.ChallengeType != types.ChallengeTypeNone {
				err.Appendf(prefix+"challenge_type", "Challenge type=%s is not supported", service.ChallengeType)
			}
			if service.ContainerPort <= 0 {
				err.Append(prefix+"container_port", "A valid container port is required to expose a service")
			}
		}

		if service.Quota.MemoryLimit > 0 {
			// explicitly specified memory limit
			if service.Quota.MemoryLimit < 128 {
				err.Appendf(prefix+"quota.memory_limit", "The minimum memory limit is 128MB")
			}
		}

		// check volumes
		if len(service.Volumes) > 0 {
			for j, volume := range service.Volumes {
				volumePrefix := fmt.Sprintf("%svolumes[%d].", prefix, j)

				if volume.Name == "" {
					err.Append(volumePrefix+"name", "Volume name is required")
				}

				if strings.Contains(volume.Name, " ") {
					err.Append(volumePrefix+"name", "Volume name must not contain spaces")
				}

				if volume.Path == "" {
					err.Append(volumePrefix+"path", "Volume path is required")
				}

				if !strings.HasPrefix(volume.Path, "/") {
					err.Append(volumePrefix+"path", "Volume path must be absolute")
				}
			}
		}
	}

	if err.HasFailures() {
		return err
	} else {
		return nil
	}
}

// ApplyProject will apply the configuration changes for the specified project.
// It will create the required resources and clean up the no longer referenced resources.
func (o *Container) ApplyProject(ctx context.Context, opts *ApplyProjectOptions) ([]types.Service, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}

	// translate apply options to requested services
	domains := make([]string, 0, len(opts.Services))
	containers := make([]string, 0, len(opts.Services))
	services := make([]types.Service, len(opts.Services))
	for i, service := range opts.Services {
		services[i] = types.Service{
			Name:           service.Name,
			Hash:           "", // calculated below
			ContainerName:  fmt.Sprintf("%s_%s", opts.ProjectName, service.Name),
			ContainerImage: "", // retrieved below
			Source:         service.Source,
			Environment:    service.Environment,
			Quota:          service.Quota,
			Volumes:        service.Volumes,
			Ingress: types.Ingress{
				Domains:        service.IngressDomains,
				ContainerPort:  service.ContainerPort,
				TargetEndpoint: "", // will be supplied by docker (if exposed)
				TargetService:  service.Name,
				TargetProject:  opts.ProjectName,
				ChallengeType:  service.ChallengeType,
			},
		}

		// append domains for project
		if len(service.IngressDomains) > 0 {
			domains = append(domains, service.IngressDomains...)
		}

		// append container names
		containers = append(containers, services[i].ContainerName)

		// build or set container image
		img, err := source.GetImageFromSource(ctx, services[i])
		if err != nil {
			return nil, fmt.Errorf("failed to get image: %w", err)
		}
		services[i].ContainerImage = img

		// calculate hash
		services[i].Hash = types.CalculateHash(&services[i])
	}

	// 1. create docker network (if not exists)
	// 2. remove unused routes
	// 3. remove no longer referenced services
	// 4. apply changes for each service
	// 	-> image creation step (only docker supported for now)
	//	-> create docker container
	//	-> setup ingress route
	// 5. save changes to database
	o.logger.Infof("Preparing to apply project=%s with %d services", opts.ProjectName, len(services))
	net, err := o.Docker.CreateNetworkIfNotExists(ctx, opts.ProjectName)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker network: %w", err)
	}

	removed, err := o.IngressManager.RemoveUnusedRoutes(opts.ProjectName, domains)
	if err != nil {
		return nil, fmt.Errorf("failed to remove unused routes: %w", err)
	}
	if removed > 0 {
		o.logger.Debugf("Successfully removed %d unused routes for project=%s", removed, opts.ProjectName)
	}

	removed, err = o.Docker.RemoveUnusedContainers(ctx, opts.ProjectName, containers)
	if err != nil {
		return nil, fmt.Errorf("failed to remove unused containers: %w", err)
	}
	if removed > 0 {
		o.logger.Debugf("Successfully removed %d unused containers for project=%s", removed, opts.ProjectName)
	}

	// apply changes for each service
	for i, service := range services {
		var applied *types.Service
		applied, err = o.ApplyService(ctx, service, net)
		if err != nil {
			return nil, fmt.Errorf("failed to create service %s: %w", service.Name, err)
		}

		// overwrite
		services[i] = *applied
	}

	return services, o.Database.SaveProject(opts.ProjectName, services)
}

// ApplyService will create the resources required for starting the service.
func (o *Container) ApplyService(ctx context.Context, service types.Service, net *docker.Network) (*types.Service, error) {
	// PRE. check if container already exists
	// 	-> compare hash (remove container if it's different)
	//	-> start container if it's not running
	//	-> no action required, container already exists
	container := o.Docker.FindContainer(ctx, service.ContainerName)
	if container != nil {
		o.logger.Debugf("Services with name=%s already exists, checking status (container_id=%s)", service.Name, container.ID)

		// we should remap the endpoint here.
		// docker decides what endpoint the container is exposed on
		service.Ingress.TargetEndpoint = container.Endpoint

		if container.ConfigHash != service.Hash {
			o.logger.Warningf("Configuration hash does not match for service=%s with container_id=%s, rebuilding", service.Name, container.ID)
			err := o.Docker.RemoveContainer(ctx, container.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to remove old container: %w", err)
			}

			// reset target endpoint -> we have removed it.
			service.Ingress.TargetEndpoint = ""
		} else {
			if container.IsRunning() {
				o.logger.Tracef("Services with name=%s is already running, no action required", service.Name)
			} else {
				o.logger.Warningf("Container with id=%s for service=%s is not running, restarting", container.ID, service.Name)
				err := o.Docker.StartContainer(ctx, container.ID)
				if err != nil {
					return nil, fmt.Errorf("unable to start container: %w", err)
				}
			}

			// make sure that routes are registered
			err := o.IngressManager.RegisterRoute(service.Ingress)
			if err != nil {
				return nil, fmt.Errorf("failed to register routes: %w", err)
			}

			return &service, nil
		}
	}

	// 1. create + start container from service
	// 2. register ingress route to container endpoint
	container, err := o.Docker.CreateContainer(ctx, service, net)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	err = o.Docker.StartContainer(ctx, container.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// service is configured to be exposed
	service.Ingress.TargetEndpoint = container.Endpoint
	err = o.IngressManager.RegisterRoute(service.Ingress)
	if err != nil {
		return nil, fmt.Errorf("failed to register ingress route: %w", err)
	}

	o.logger.Debugf("Successfully created container=%s for service=%s (project=%s)", container.ID, service.Name, service.Ingress.TargetProject)
	return &service, nil
}

// RemoveProject will remove the resources associated to the project.
func (o *Container) RemoveProject(ctx context.Context, project string) error {
	// 1. remove ingress routes
	// 2. remove services
	// 3. remove network
	// 4. remove project from database

	routes, err := o.IngressManager.RemoveAllRoutes(project)
	if err != nil {
		return fmt.Errorf("failed to remove routes: %w", err)
	}

	containers, err := o.Docker.RemoveAllContainersForProject(ctx, project)
	if err != nil {
		return fmt.Errorf("failed to remove containers: %w", err)
	}

	// delete network
	err = o.Docker.DeleteNetwork(ctx, project)
	if err != nil {
		// only log warning
		o.logger.Warningf("Failed to remove network for project=%s (rsn=%v)", project, err)
	}

	o.logger.Infof("Successfully removed project=%s from the system (containers=%d, routes=%d)", project, containers, routes)
	return o.Database.RemoveProject(project)
}

const (
	StatusNotAvailable = "not_available"
	StatusRunning      = "running"
	StatusStopped      = "stopped"
)

type Status struct {
	Services []types.Service
	statuses map[string]string
}

// GetState retrieves the current state of the service.
func (s *Status) GetState(service string) string {
	current, ok := s.statuses[service]
	if !ok {
		return StatusNotAvailable
	}

	return current
}

// GetProjectStatus will return the actual status of the container running on the system.
func (o *Container) GetProjectStatus(ctx context.Context, project string) (*Status, error) {
	status := &Status{
		statuses: make(map[string]string),
	}

	status.Services = o.Database.GetServicesForProject(project)
	if len(status.Services) == 0 {
		return nil, errors.New("no services found")
	}

	// go over each service and inspect both ingress + container
	for _, service := range status.Services {
		container := o.Docker.FindContainer(ctx, service.ContainerName)
		if container != nil {
			if container.IsRunning() {
				status.statuses[service.Name] = StatusRunning
			} else {
				status.statuses[service.Name] = StatusStopped
			}
		}
	}

	return status, nil
}

// IsProjectRunning will return true if all services within the project are running.
func (o *Container) IsProjectRunning(ctx context.Context, project string) bool {
	status, err := o.GetProjectStatus(ctx, project)
	if err != nil {
		return false
	}

	for _, service := range status.Services {
		if status.GetState(service.Name) != StatusRunning {
			return false
		}
	}

	return true
}

// FindAllProjects will return all projects currently available.
func (o *Container) FindAllProjects() map[string][]types.Service {
	return o.Database.GetAllProjects()
}
