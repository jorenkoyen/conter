package source

import (
	"context"
	"fmt"
	"github.com/jorenkoyen/conter/manager/types"
)

const (
	Docker = "docker"
	Git    = "git"
)

// GetImageFromSource will return the container image name to use when creating the service.
func GetImageFromSource(ctx context.Context, service types.Service) (string, error) {
	switch service.Source.Type {
	case Docker:
		return service.Source.URI, nil
	case Git:
		return NewBuilder().Build(ctx, service)
	default:
		return "", fmt.Errorf("source=%s is not supported", service.Source.Type)
	}
}
