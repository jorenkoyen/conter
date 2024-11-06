package db

import (
	"encoding/json"
	"errors"
	"github.com/jorenkoyen/conter/manager/types"
	"time"

	"github.com/jorenkoyen/go-logger"
	"github.com/jorenkoyen/go-logger/log"
	"go.etcd.io/bbolt"
)

var (
	BucketProjects     = []byte("projects")
	BucketRoutes       = []byte("routes")
	BucketConfig       = []byte("config")
	BucketChallenges   = []byte("challenges")
	BucketCertificates = []byte("certificates")

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
	db, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: time.Second * 2})
	if err != nil {
		l.Fatalf("Failed to create new database client: %v", err)
	}

	l.Debugf("Successfully opened database at path=%s", path)
	return &Client{logger: l, bolt: db}
}

// SaveProject will persist the project in the database.
func (c *Client) SaveProject(project string, services []types.Service) error {
	return c.bolt.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(BucketProjects)
		if err != nil {
			return err
		}

		content, err := json.Marshal(services)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(project), content)
	})
}

// RemoveProject will remove the project from the database.
func (c *Client) RemoveProject(name string) error {
	return c.bolt.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(BucketProjects)
		if err != nil {
			return err
		}
		return bucket.Delete([]byte(name))
	})
}

// GetServicesForProject will return the services associated for the project.
// If no services are available or the project does not exist it will return nil.
func (c *Client) GetServicesForProject(project string) []types.Service {
	var services []types.Service
	_ = c.bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketProjects)
		if bucket == nil {
			return ErrItemNotFound
		}

		content := bucket.Get([]byte(project))
		if content == nil {
			return ErrItemNotFound
		}

		return json.Unmarshal(content, &services)
	})
	return services
}

// GetAllProjects will return a map of all projects known by the system in combination with their services.
func (c *Client) GetAllProjects() map[string][]types.Service {
	output := make(map[string][]types.Service)
	_ = c.bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketProjects)
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(project, content []byte) error {
			var services []types.Service
			if err := json.Unmarshal(content, &services); err != nil {
				return nil // ignore
			}
			output[string(project)] = services
			return nil
		})
	})
	return output
}

// GetIngressRoute will return the ingress route if it exists.
func (c *Client) GetIngressRoute(domain string) (*types.Ingress, error) {
	route := new(types.Ingress)
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

func (c *Client) SaveIngressRoute(route *types.Ingress) error {
	return c.bolt.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(BucketRoutes)
		if err != nil {
			return err
		}

		content, err := json.Marshal(route)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(route.Domain), content)
	})
}

// GetIngressRoutesByProject returns all ingress routes related to the project.
func (c *Client) GetIngressRoutesByProject(project string) []types.Ingress {
	routes := make([]types.Ingress, 0)
	_ = c.bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketRoutes)
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(domain, content []byte) error {
			r := new(types.Ingress)
			if err := json.Unmarshal(content, r); err == nil && r.TargetProject == project {
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

// GetDomainChallenge will return the latest known ACME challenge.
// If no challenge exists it will return nil.
func (c *Client) GetDomainChallenge(domain string) *types.AcmeChallenge {
	challenge := new(types.AcmeChallenge)
	err := c.bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketChallenges)
		if bucket == nil {
			return ErrItemNotFound
		}

		content := bucket.Get([]byte(domain))
		if content == nil {
			return ErrItemNotFound
		}

		return json.Unmarshal(content, challenge)
	})

	if err != nil {
		return nil
	}

	return challenge
}

// SetAcmeChallenge will persist the ACME challenge for validating a certificate request.
func (c *Client) SetAcmeChallenge(domain string, token string, auth string) error {
	challenge := &types.AcmeChallenge{
		Token: token,
		Auth:  auth,
	}

	return c.bolt.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(BucketChallenges)
		if err != nil {
			return err
		}

		content, err := json.Marshal(challenge)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(domain), content)
	})
}

// RemoveAcmeChallenge will remove the ACME challenge if all parameters match.
func (c *Client) RemoveAcmeChallenge(domain string, token string, auth string) error {
	return c.bolt.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketChallenges)
		if bucket == nil {
			return nil // no action required
		}

		content := bucket.Get([]byte(domain))
		if content == nil {
			return nil // no action required
		}

		challenge := new(types.AcmeChallenge)
		err := json.Unmarshal(content, challenge)
		if err != nil {
			return err
		}

		if challenge.Token == token && challenge.Auth == auth {
			return bucket.Delete([]byte(domain))
		} else {
			return nil // no action required
		}
	})
}

// GetCertificate retrieves the certificate for the specified domain.
func (c *Client) GetCertificate(domain string) (*types.Certificate, error) {
	certificate := new(types.Certificate)
	err := c.bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketCertificates)
		if bucket == nil {
			return ErrItemNotFound
		}

		content := bucket.Get([]byte(domain))
		if content == nil {
			return ErrItemNotFound
		}

		return json.Unmarshal(content, certificate)
	})

	return certificate, err
}

// GetAllCertificates will return all certificates with the key being the domain name.
func (c *Client) GetAllCertificates() map[string]*types.Certificate {
	output := make(map[string]*types.Certificate)
	_ = c.bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketCertificates)
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(domain, content []byte) error {
			var cert *types.Certificate
			if err := json.Unmarshal(content, &cert); err != nil {
				return nil // ignore errors
			}

			output[string(domain)] = cert
			return nil
		})
	})

	return output
}

// RemoveCertificate removes the certificate from the bucket.
func (c *Client) RemoveCertificate(domain string) error {
	return c.bolt.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketCertificates)
		if bucket == nil {
			return nil
		}

		return bucket.Delete([]byte(domain))
	})
}

// SetCertificate persists the certificate configuration for the domain.
func (c *Client) SetCertificate(domain string, cert *types.Certificate) error {
	return c.bolt.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(BucketCertificates)
		if err != nil {
			return err
		}

		content, err := json.Marshal(cert)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(domain), content)
	})
}

// getConfigContent returns the byte content from 'config' bucket.
func (c *Client) getConfigContent(key []byte) ([]byte, error) {
	var content []byte
	err := c.bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketConfig)
		if bucket == nil {
			return ErrItemNotFound
		}

		content = bucket.Get(key)
		if content == nil {
			return ErrItemNotFound
		}

		return nil
	})

	return content, err
}

// setConfigContent updates a key inside the 'config' bucket.
func (c *Client) setConfigContent(key []byte, content []byte) error {
	return c.bolt.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(BucketConfig)
		if err != nil {
			return err
		}

		return bucket.Put(key, content)
	})
}

// removeConfigContent will remove the configuration content key from the 'config' bucket.
func (c *Client) removeConfigContent(key []byte) error {
	return c.bolt.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketConfig)
		if bucket == nil {
			return nil
		}
		return bucket.Delete(key)
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
