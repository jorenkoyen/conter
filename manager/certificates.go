package manager

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/jorenkoyen/conter/manager/db"
	"github.com/jorenkoyen/conter/manager/types"
	"github.com/jorenkoyen/conter/version"
	"github.com/jorenkoyen/go-logger"
	"github.com/jorenkoyen/go-logger/log"
	"net/http"
)

var (
	// LetsEncryptDirectoryUrl will point to the staging environment by default.
	LetsEncryptDirectoryUrl = lego.LEDirectoryStaging
	// InsecureDirectory informs the application that the ACME directory does not have valid TLS certificates.
	InsecureDirectory = false
)

type CertificateManager struct {
	logger *logger.Logger
	config *db.Config
	acme   *lego.Client
}

func NewCertificateManger(database *db.Client, email string) *CertificateManager {
	mgr := &CertificateManager{
		logger: log.WithName("certificate-mgr"),
		config: db.NewConfigDatabase(database),
	}

	// check if user is initialized (if not already done)
	mgr.acme = mgr.init(email, false)

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
	if user.Email != email || !user.IsValid() {
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
	config.CADirURL = LetsEncryptDirectoryUrl
	config.UserAgent = fmt.Sprintf("conter/%s", version.Version)
	config.HTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: InsecureDirectory,
			},
		},
	}

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

	return client
}
