package db

import (
	"encoding/json"
	"errors"

	"github.com/jorenkoyen/conter/manifest"
	"github.com/jorenkoyen/go-logger"
	"github.com/jorenkoyen/go-logger/log"
	"go.etcd.io/bbolt"
)

var (
	BucketManifest = []byte("manifests")

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

// SaveManifest will persist the manifest in the database.
func (c *Client) SaveManifest(manifest *manifest.Project) error {
	return c.bolt.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(BucketManifest)
		if err != nil {
			return err
		}

		content, err := json.Marshal(manifest)
		if err != nil {
			return err
		}

		c.logger.Tracef("Saving manifest with name=%s", manifest.Name)
		return bucket.Put([]byte(manifest.Name), content)
	})
}

// RemoveManifest will remove the manifest from the database.
func (c *Client) RemoveManifest(name string) error {
	return c.bolt.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(BucketManifest)
		if err != nil {
			return err
		}

		c.logger.Tracef("Removing manifest with name=%s", name)
		return bucket.Delete([]byte(name))
	})
}

// GetManifestByName will return the manifest with the matching name if present.
func (c *Client) GetManifestByName(name string) (*manifest.Project, error) {
	project := new(manifest.Project)
	err := c.bolt.View(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(BucketManifest)
		if err != nil {
			return err
		}

		content := bucket.Get([]byte(name))
		if content == nil {
			return ErrItemNotFound
		}

		c.logger.Tracef("Retrieving manifest with name=%s", name)
		return json.Unmarshal(content, project)
	})

	return project, err
}

// Close will cl
func (c *Client) Close() error {
	if c.bolt != nil {
		c.logger.Trace("Closing connection to database")
		return c.bolt.Close()
	}
	return nil
}
