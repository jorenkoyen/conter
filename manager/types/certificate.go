package types

import (
	"crypto"
	"crypto/tls"
	"encoding/base64"
	"github.com/go-acme/lego/v4/registration"
)

// AcmeRegistration contains all the information in relation to a complete ACME registration.
type AcmeRegistration struct {
	Email        string
	PrivateKey   crypto.PrivateKey
	Registration *registration.Resource
}

func (u *AcmeRegistration) GetEmail() string {
	return u.Email
}
func (u *AcmeRegistration) GetRegistration() *registration.Resource {
	return u.Registration
}
func (u *AcmeRegistration) GetPrivateKey() crypto.PrivateKey {
	return u.PrivateKey
}

// IsValid will return true when all required fields are available in the ACME registration.
func (u *AcmeRegistration) IsValid() bool {
	return u.Registration != nil && u.PrivateKey != nil && u.Email != ""
}

// AcmeChallenge represents an ACME challenge.
type AcmeChallenge struct {
	Token string
	Auth  string
}

type Certificate struct {
	Key           string        `json:"key"`
	Certificate   string        `json:"certificate"`
	ChallengeType ChallengeType `json:"challenge_type"`
}

// CertificateBytes will return the bytes of the certificate.
func (c *Certificate) CertificateBytes() ([]byte, error) {
	return base64.StdEncoding.DecodeString(c.Certificate)
}

// PrivateKeyBytes will return the bytes of the private key.
func (c *Certificate) PrivateKeyBytes() ([]byte, error) {
	return base64.StdEncoding.DecodeString(c.Key)
}

// X509KeyPair will return the X509 key pair extracted from the certificate and private key.
func (c *Certificate) X509KeyPair() (*tls.Certificate, error) {
	certificate, err := c.CertificateBytes()
	if err != nil {
		return nil, err
	}

	key, err := c.PrivateKeyBytes()
	if err != nil {
		return nil, err
	}

	pair, err := tls.X509KeyPair(certificate, key)
	if err != nil {
		return nil, err
	}

	return &pair, nil
}
