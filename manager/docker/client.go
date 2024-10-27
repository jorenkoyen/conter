package docker

import (
	"context"
	"errors"

	"github.com/jorenkoyen/go-logger"
	"github.com/jorenkoyen/go-logger/log"
)

type Client struct {
	logger *logger.Logger
}

func NewClient() *Client {
	return &Client{
		logger: log.WithName("docker"),
	}
}

type Network struct {
}

// CreateNetworkIfNotExists will create a new docker network if not yet available.
func (c *Client) CreateNetworkIfNotExists(ctx context.Context, name string) (*Network, error) {
	return nil, errors.New("not implemented")
}

// Close will close the open connection to the docker daemon.
func (c *Client) Close() error {
	// TODO: implement
	return nil
}
