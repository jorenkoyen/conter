package manager

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/jorenkoyen/conter/manager/db"
	"github.com/jorenkoyen/conter/manager/types"
	"github.com/jorenkoyen/conter/version"
	"github.com/jorenkoyen/go-logger"
	"github.com/jorenkoyen/go-logger/log"
	"net/http"
)

type CertificateManager struct {
	logger *logger.Logger
	config *db.Config
	data   *db.Client
	acme   *lego.Client

	directory string
	insecure  bool
}

func NewCertificateManger(database *db.Client, email string, directory string, insecure bool) *CertificateManager {
	mgr := &CertificateManager{
		logger:    log.WithName("certificate-mgr"),
		config:    db.NewConfigDatabase(database),
		data:      database,
		directory: directory,
		insecure:  insecure,
	}

	if email == "" {
		mgr.logger.Warningf("No ACME email address set, please update configuration before requesting certificates")
	} else {
		// check if user is initialized (if not already done)
		mgr.acme = mgr.init(email, false)
	}

	return mgr
}

// registration will return the current ACME registration resource based on the stored data.
func (c *CertificateManager) registration() *types.AcmeRegistration {
	return &types.AcmeRegistration{
		Email:        c.config.GetAcmeEmail(),
		PrivateKey:   c.config.GetAcmePrivateKey(),
		Registration: c.config.GetAcmeRegistration(),
	}
}

// init will initialize the certificate manager by registering the user with the ACME registry.
// If the user is already registered no action will be undertaken.
func (c *CertificateManager) init(email string, isRetry bool) *lego.Client {
	user := c.registration()
	if user.Email != email || !user.IsValid() || c.directory != c.config.GetAcmeDirectory() {
		c.logger.Infof("Initializing ACME user for email=%s", email)
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			c.logger.Fatalf("Failed to generate private key: %v", err)
		}

		user = &types.AcmeRegistration{
			Email:      email,
			PrivateKey: privateKey,
		}
	}

	// continue LEGO configuration
	config := lego.NewConfig(user)
	config.CADirURL = c.directory
	config.UserAgent = fmt.Sprintf("conter/%s", version.Version)
	config.HTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: c.insecure,
			},
		},
	}

	c.logger.Tracef("Creating ACME client for directory=%s (email=%s)", config.CADirURL, user.Email)
	client, err := lego.NewClient(config)
	if err != nil {
		c.logger.Fatalf("Unable to create LEGO client: %v", err)
	}

	// check if we require registration
	if user.GetRegistration() == nil {
		reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			c.logger.Fatalf("Failed to register with ACME: %v", err)
		}

		c.logger.Debugf("Successfully registered with ACME registry (uri=%s)", reg.URI)
		user.Registration = reg

		// persist all information
		c.config.SetAcmeEmail(user.Email)
		c.config.SetAcmePrivateKey(user.PrivateKey)
		c.config.SetAcmeRegistration(user.Registration)
		c.config.SetAcmeDirectory(c.directory) // persist directory URL to compare registration
	} else {
		// validate registration
		reg, err := client.Registration.QueryRegistration()
		if err != nil {
			c.logger.Errorf("Failed to query current ACME registration: %v", err)
			c.config.ClearAcme() // clear ACME as it is no longer valid.
			if !isRetry {
				// retry client initialization
				return c.init(email, true)
			}
		}

		c.logger.Tracef("Current active ACME registration on uri=%s", reg.URI)
	}

	// set challenge providers
	err = client.Challenge.SetHTTP01Provider(c)
	if err != nil {
		c.logger.Fatalf("Failed to register HTTP-01 provider: %v", err)
	}

	return client
}

func (c *CertificateManager) Present(domain string, token string, auth string) error {
	c.logger.Tracef("Presenting new ACME challenge for domain=%s (token=%s)", domain, token)
	return c.data.SetAcmeChallenge(domain, token, auth)
}

func (c *CertificateManager) CleanUp(domain string, token string, auth string) error {
	c.logger.Tracef("Removing ACME challenge for domain=%s (token=%s)", domain, token)
	return c.data.RemoveAcmeChallenge(domain, token, auth)
}

// Authorize will return the authorization if available for the given domain.
func (c *CertificateManager) Authorize(domain string, token string) (string, error) {
	challenge := c.data.GetDomainChallenge(domain)
	if challenge == nil {
		return "", errors.New("no challenge available")
	}

	if challenge.Token != token {
		return "", errors.New("invalid token")
	}

	return challenge.Auth, nil
}

// ChallengeCreate will create a new challenge request for the ingress domain.
func (c *CertificateManager) ChallengeCreate(domain string, challenge types.ChallengeType) {
	if c.acme == nil {
		c.logger.Errorf("Unable to request certificate for domain=%s, ACME email is not configured", domain)
		return
	}

	if challenge == types.ChallengeTypeNone {
		c.logger.Tracef("Ignoring challenge creation for domain=%s", domain)
		return
	}

	if challenge != types.ChallengeTypeHTTP {
		c.logger.Errorf("Challenge type=%s is not supported", challenge)
		return
	}

	if c.data.GetDomainChallenge(domain) != nil {
		c.logger.Infof("Challenge for domain=%s already exists, skipping", domain)
		return
	}

	go func() {
		req := certificate.ObtainRequest{
			Domains: []string{domain},
			Bundle:  true,
		}

		c.logger.Infof("Requesting certificates for domain=%s", domain)
		resource, err := c.acme.Certificate.Obtain(req)
		if err != nil {
			c.logger.Errorf("Failed to obtain certificates for domain=%s: %v", domain, err)
			return
		}

		c.logger.Tracef("Successfully obtained certificates for domain=%s (uri=%s)", domain, resource.CertURL)

		cert := &types.Certificate{
			Certificate:   base64.StdEncoding.EncodeToString(resource.Certificate),
			Key:           base64.StdEncoding.EncodeToString(resource.PrivateKey),
			ChallengeType: challenge,
		}

		// persist certificate for domain
		err = c.data.SetCertificate(domain, cert)
		if err != nil {
			c.logger.Errorf("Failed to save certificate for domain=%s: %v", domain, err)
		}
	}()
}

// Get will retrieve the active certificate for the given domain if available.
// If no certificate is available it will return nil.
func (c *CertificateManager) Get(domain string) *types.Certificate {
	cert, err := c.data.GetCertificate(domain)
	if err != nil {
		if !errors.Is(err, db.ErrItemNotFound) {
			c.logger.Warningf("Failed to retrieve certificate for domain=%s: %v", domain, err)
		}

		return nil
	}

	return cert
}

// GetAll will retrieve all certificates currently known to the system.
func (c *CertificateManager) GetAll() map[string]*types.Certificate {
	return c.data.GetAllCertificates()
}
