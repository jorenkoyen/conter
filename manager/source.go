package manager

import (
	"context"
	"fmt"
	"github.com/jorenkoyen/conter/manifest"
)

const (
	SourceDocker = "docker"
	SourceGit    = "git"
)

// getDockerImageFromSource will extract the docker image required for the service based on the configured source.
func (o *Orchestrator) getDockerImageFromSource(ctx context.Context, service manifest.Service) (string, error) {
	// TODO: add support for GIT source

	if service.Source.Type == SourceDocker {
		return service.Source.URI, nil
	}

	return "", fmt.Errorf("source=%s is not supported", service.Source.Type)
}
