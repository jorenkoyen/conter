package docker

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/jorenkoyen/conter/manager/types"
	"github.com/jorenkoyen/go-logger"
	"github.com/jorenkoyen/go-logger/log"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type Client struct {
	logger *logger.Logger
	docker client.APIClient

	volumesDirectory string
}

func NewClient(directory string) *Client {
	c := &Client{logger: log.WithName("docker")}
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		c.logger.Fatalf("Failed to create docker client: %v", err)
	} else {
		c.docker = docker
	}

	if !strings.HasPrefix(directory, "/") {
		c.logger.Warningf("Resolving relative data directory to asbolute path for docker volume bindings")
		cwd, _ := os.Getwd()
		directory = filepath.Join(cwd, directory)
	}

	c.volumesDirectory = filepath.Join(directory, "volumes")
	return c
}

type Network struct {
	ID     string
	Name   string
	Labels map[string]string
}

// CreateNetworkIfNotExists will create a new docker network if not yet available.
func (c *Client) CreateNetworkIfNotExists(ctx context.Context, name string) (*Network, error) {
	inspect, err := c.docker.NetworkInspect(ctx, name, network.InspectOptions{})
	if err == nil {
		c.logger.Debugf("Network with name=%s already existed", name)
		return &Network{Name: name, ID: inspect.ID, Labels: inspect.Labels}, nil
	}

	c.logger.Tracef("Creating new network with name=%s", name)
	opts := network.CreateOptions{Labels: DefaultLabels()}
	create, err := c.docker.NetworkCreate(ctx, name, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %v", err)
	}

	return &Network{Name: name, ID: create.ID, Labels: opts.Labels}, nil
}

// DeleteNetwork will delete the network from the docker daemon.
func (c *Client) DeleteNetwork(ctx context.Context, name string) error {
	c.logger.Tracef("Removing network with name=%s", name)
	err := c.docker.NetworkRemove(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to remove network: %v", err)
	}
	return nil
}

type Container struct {
	ID         string
	Name       string
	State      string
	Endpoint   string
	ConfigHash string
}

// IsRunning will check if the current state of the container is marked as 'running'.
func (c *Container) IsRunning() bool {
	return c.State == "running"
}

// FindContainer will retrieve the container information for the service with the given name that belongs to the project.
func (c *Client) FindContainer(ctx context.Context, name string) *Container {
	inspect, err := c.docker.ContainerInspect(ctx, name)
	if err != nil {
		// unable to find container
		return nil
	}

	// find FIRST exposed port if any
	var endpoint string
	for _, bindings := range inspect.HostConfig.PortBindings {
		if len(bindings) > 0 {
			// port is exposed
			binding := bindings[0]
			endpoint = fmt.Sprintf("%s:%s", binding.HostIP, binding.HostPort)
			break // stop at first exposed port
		}
	}

	return &Container{
		ID:         inspect.ID,
		Name:       inspect.Name,
		State:      inspect.State.Status,
		ConfigHash: inspect.Config.Labels[LabelHash],
		Endpoint:   endpoint,
	}
}

// CreateContainer will create the container based on the service configuration.
func (c *Client) CreateContainer(ctx context.Context, service types.Service, net *Network) (*Container, error) {
	err := c.PullImageIfNotExists(ctx, service.ContainerImage)
	if err != nil {
		return nil, fmt.Errorf("failed to pull image: %w", err)
	}

	cfg := &container.Config{
		Image:    service.ContainerImage,
		Labels:   GenerateServiceLabels(service),
		Env:      TransformEnvironment(service.Environment),
		Hostname: service.Name,
	}

	hostCfg := &container.HostConfig{
		NetworkMode: container.NetworkMode(net.ID),
		RestartPolicy: container.RestartPolicy{
			Name: container.RestartPolicyAlways,
		},
		Resources: container.Resources{
			Memory: ToBytes(128), // default limit is 128MB
		},
	}

	// create volume mounts
	if len(service.Volumes) > 0 {
		mounts := make([]mount.Mount, len(service.Volumes))

		for i, volume := range service.Volumes {
			mounts[i] = mount.Mount{
				Type:   mount.TypeBind,
				Target: volume.Path,
				Source: filepath.Join(c.volumesDirectory, service.Ingress.TargetProject, service.Name, volume.Name),
				BindOptions: &mount.BindOptions{
					CreateMountpoint: true,
				},
			}
		}

		hostCfg.Mounts = mounts
	}

	if service.Quota.MemoryLimit > 0 {
		hostCfg.Resources.Memory = ToBytes(service.Quota.MemoryLimit)
	}

	var ingress string
	if service.Ingress.ContainerPort > 0 {
		// container should be exposed for networking
		internal := nat.Port(fmt.Sprintf("%d/tcp", service.Ingress.ContainerPort))
		exposed := GetAvailablePort(PortStartRange, PortEndRange)
		if exposed <= 0 {
			return nil, errors.New("no more available ports to assign")
		}

		ingress = fmt.Sprintf("127.0.0.1:%d", exposed)
		hostCfg.PortBindings = nat.PortMap{
			internal: []nat.PortBinding{{
				HostIP:   "127.0.0.1",
				HostPort: strconv.Itoa(exposed),
			}},
		}
	}

	c.logger.Tracef("Creating new container with name=%s [ image=%s ]", service.ContainerName, service.ContainerImage)
	resp, err := c.docker.ContainerCreate(ctx, cfg, hostCfg, nil, nil, service.ContainerName)
	if err != nil {
		return nil, err
	}

	return &Container{
		ID:         resp.ID,
		Name:       service.ContainerName,
		State:      "created",
		Endpoint:   ingress,
		ConfigHash: service.Hash,
	}, nil
}

// StartContainer will start up the container with the given ID.
func (c *Client) StartContainer(ctx context.Context, containerId string) error {
	c.logger.Tracef("Starting container with id=%s", containerId)
	return c.docker.ContainerStart(ctx, containerId, container.StartOptions{})
}

func (c *Client) RemoveContainer(ctx context.Context, containerId string) error {
	c.logger.Tracef("Removing container with id=%s", containerId)
	return c.docker.ContainerRemove(ctx, containerId, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: false,
	})
}

// PullImageIfNotExists will retrieve the image from the internet if it does not yet exist on the system.
func (c *Client) PullImageIfNotExists(ctx context.Context, img string) error {
	_, _, err := c.docker.ImageInspectWithRaw(ctx, img)
	if err == nil {
		// image already exists no pull required.
		c.logger.Tracef("Image with name=%s already exists, not pulling", img)
		return nil
	}

	c.logger.Tracef("Pulling image with name=%s", img)
	out, err := c.docker.ImagePull(ctx, img, image.PullOptions{})
	if err != nil {
		return err
	}

	// discard output
	_, err = io.Copy(io.Discard, out)
	if err != nil {
		return err
	}

	return out.Close()
}

// RemoveUnusedContainers will clean up all the containers for the project that are not mentioned in the excluded containers list.
func (c *Client) RemoveUnusedContainers(ctx context.Context, project string, excludedContainers []string) (int, error) {
	containers, err := c.docker.ContainerList(ctx, container.ListOptions{Filters: ProjectFilter(project)})
	if err != nil {
		return 0, err
	}

	removed := 0
	for _, _container := range containers {
		excluded := false

		// check to see if the container name is not excluded.
		if len(excludedContainers) > 0 {
			for _, name := range _container.Names {
				name = strings.TrimPrefix(name, "/")

				if slices.Index(excludedContainers, name) != -1 {
					// container should be excluded!
					excluded = true
					break
				}
			}
		}

		if excluded {
			// skip container deletion.
			continue
		}

		// remove container
		if err = c.RemoveContainer(ctx, _container.ID); err != nil {
			return removed, err
		}

		removed++
	}

	return removed, nil
}

// RemoveAllContainersForProject will purge all containers from the system that are linked to the given project.
func (c *Client) RemoveAllContainersForProject(ctx context.Context, project string) (int, error) {
	return c.RemoveUnusedContainers(ctx, project, []string{})
}

// Close will close the open connection to the docker daemon.
func (c *Client) Close() error {
	if c.docker != nil {
		return c.docker.Close()
	}
	return nil
}
