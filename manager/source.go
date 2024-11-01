package manager

import (
	"context"
	"fmt"
	"github.com/jorenkoyen/conter/manager/types"
)

const (
	SourceDocker = "docker"
	SourceGit    = "git"
)

// getDockerImageFromSource will extract the docker image required for the service based on the configured source.
func (o *Container) getDockerImageFromSource(ctx context.Context, service types.Service) (string, error) {
	// TODO: add support for GIT source

	if service.Source.Type == SourceDocker {
		return service.Source.URI, nil
	}

	return "", fmt.Errorf("source=%s is not supported", service.Source.Type)
}
