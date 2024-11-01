package db

import (
	"encoding/json"
	"errors"

	"github.com/jorenkoyen/conter/model"
	"github.com/jorenkoyen/go-logger"
	"github.com/jorenkoyen/go-logger/log"
	"go.etcd.io/bbolt"
)

var (
	BucketProjects   = []byte("projects")
	BucketRoutes     = []byte("routes")
	BucketChallenges = []byte("challenges")

	ErrItemNotFound = errors.New("item not found")
)

// Client acts as the interface between to communicate with our database system.
type Client struct {
	logger *logger.Logger
	bolt   *bbolt.DB
}

// NewClient will create a new database client for handling operations.
func NewClient(path string) *Client {
	l := log.WithName("database")
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		l.Fatalf("Failed to create new database client: %v", err)
	}

	l.Debugf("Successfully opened database at path=%s", path)
	return &Client{logger: l, bolt: db}
}

// SaveProject will persist the project in the database.
func (c *Client) SaveProject(project *model.Project) error {
	return c.bolt.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(BucketProjects)
		if err != nil {
			return err
		}

		content, err := json.Marshal(project)
		if err != nil {
			return err
		}

		c.logger.Tracef("Saving project with name=%s", project.Name)
		return bucket.Put([]byte(project.Name), content)
	})
}

// RemoveProject will remove the project from the database.
func (c *Client) RemoveProject(name string) error {
	return c.bolt.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(BucketProjects)
		if err != nil {
			return err
		}

		c.logger.Tracef("Removing project with name=%s", name)
		return bucket.Delete([]byte(name))
	})
}

// GetProjectByName will return the project with the matching name if present.
func (c *Client) GetProjectByName(name string) (*model.Project, error) {
	project := new(model.Project)
	err := c.bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketProjects)
		if bucket == nil {
			return ErrItemNotFound
		}

		content := bucket.Get([]byte(name))
		if content == nil {
			return ErrItemNotFound
		}

		c.logger.Tracef("Retrieving project with name=%s", name)
		return json.Unmarshal(content, project)
	})

	return project, err
}

// GetIngressRoute will return the ingress route if it exists.
func (c *Client) GetIngressRoute(domain string) (*model.IngressRoute, error) {
	route := new(model.IngressRoute)
	err := c.bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketRoutes)
		if bucket == nil {
			return ErrItemNotFound
		}

		content := bucket.Get([]byte(domain))
		if content == nil {
			return ErrItemNotFound
		}

		return json.Unmarshal(content, route)
	})
	return route, err
}

func (c *Client) SaveIngressRoute(route *model.IngressRoute) error {
	return c.bolt.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(BucketRoutes)
		if err != nil {
			return err
		}

		content, err := json.Marshal(route)
		if err != nil {
			return err
		}

		c.logger.Tracef("Saving ingress route for domain=%s", route.Domain)
		return bucket.Put([]byte(route.Domain), content)
	})
}

// GetIngressRoutesByProject returns all ingress routes related to the project.
func (c *Client) GetIngressRoutesByProject(project string) []model.IngressRoute {
	routes := make([]model.IngressRoute, 0)
	_ = c.bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketRoutes)
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(domain, content []byte) error {
			r := new(model.IngressRoute)
			if err := json.Unmarshal(content, r); err == nil && r.Project == project {
				// append to routes array
				routes = append(routes, *r)
			}
			return nil
		})
	})
	return routes
}

func (c *Client) RemoveIngressRoute(domain string) error {
	return c.bolt.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketRoutes)
		if bucket == nil {
			return nil
		}

		return bucket.Delete([]byte(domain))
	})
}

// Close will cl
func (c *Client) Close() error {
	if c.bolt != nil {
		c.logger.Trace("Closing connection to database")
		return c.bolt.Close()
	}
	return nil
}
