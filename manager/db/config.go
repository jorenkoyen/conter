package db

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"github.com/go-acme/lego/v4/registration"
	"github.com/jorenkoyen/go-logger/log"
)

var (
	KeyAcmeEmail        = []byte("acme.email")
	KeyAcmePrivateKey   = []byte("acme.private_key")
	KeyAcmeRegistration = []byte("acme.registration")
)

type Config struct {
	client *Client
}

// NewConfigDatabase creates a new database that only interacts with the configuration bucket.
func NewConfigDatabase(c *Client) *Config {
	return &Config{client: c}
}

// GetAcmeEmail will return the email address of the ACME user.
func (c *Config) GetAcmeEmail() string {
	content, err := c.client.getConfigContent(KeyAcmeEmail)
	if err != nil {
		return ""
	}

	return string(content)
}

// SetAcmeEmail will configure the ACME user email.
func (c *Config) SetAcmeEmail(email string) {
	err := c.client.setConfigContent(KeyAcmeEmail, []byte(email))
	if err != nil {
		log.Panicf("Failed to set content for ACME email: %v", err)
	}
}

// GetAcmePrivateKey will return the private key used to register the user via ACME.
func (c *Config) GetAcmePrivateKey() crypto.PrivateKey {
	content, err := c.client.getConfigContent(KeyAcmePrivateKey)
	if err != nil {
		return nil
	}

	// Decode the PEM block
	block, _ := pem.Decode(content)
	if block == nil || block.Type != "PRIVATE KEY" && block.Type != "RSA PRIVATE KEY" && block.Type != "EC PRIVATE KEY" {
		log.Panicf("failed to decode PEM block containing private key")
		return nil
	}

	// Parse the private key based on its type
	var parsedKey crypto.PrivateKey
	parsedKey, err = x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		log.Panicf("failed to parse private key: %v", err)
	}

	switch key := parsedKey.(type) {
	case *ecdsa.PrivateKey:
		return key
	default:
		log.Panicf("unsupported private key type")
		return nil
	}
}

// SetAcmePrivateKey will persist the ACME private key into the configuration bucket.
func (c *Config) SetAcmePrivateKey(privateKey crypto.PrivateKey) {
	var keyBytes []byte
	var err error

	switch key := privateKey.(type) {
	case *ecdsa.PrivateKey:
		// Encode EC key in PKCS8 DER format
		keyBytes, err = x509.MarshalECPrivateKey(key)
		if err != nil {
			log.Panicf("failed to marshal EC private key: %v", err)
		}
	default:
		log.Panicf("unsupported key type %T", privateKey)
	}

	// Wrap the DER bytes in a PEM block
	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	}

	content := pem.EncodeToMemory(pemBlock)
	err = c.client.setConfigContent(KeyAcmePrivateKey, content)
	if err != nil {
		log.Panicf("failed to set content for ACME private key: %v", err)
	}
}

// GetAcmeRegistration will return the registration resource we got from the ACME authority.
func (c *Config) GetAcmeRegistration() *registration.Resource {
	content, err := c.client.getConfigContent(KeyAcmeRegistration)
	if err != nil {
		return nil
	}

	reg := new(registration.Resource)
	err = json.Unmarshal(content, reg)
	if err != nil {
		log.Panicf("failed to unmarshal registration: %v", err)
		return nil
	}
	return reg
}

// SetAcmeRegistration will persist the registration resources we got from the ACME authority.
func (c *Config) SetAcmeRegistration(registration *registration.Resource) {
	content, err := json.Marshal(registration)
	if err != nil {
		log.Panicf("failed to marshal registration: %v", err)
	}
	err = c.client.setConfigContent(KeyAcmeRegistration, content)
	if err != nil {
		log.Panicf("failed to set content for ACME registration: %v", err)
	}
}

func (c *Config) ClearAcme() {
	if err := c.client.removeConfigContent(KeyAcmeEmail); err != nil {
		log.Panicf("Faield to remove ACME email: %v", err)
	}
	if err := c.client.removeConfigContent(KeyAcmeRegistration); err != nil {
		log.Panicf("Faield to remove ACME registration: %v", err)
	}
	if err := c.client.removeConfigContent(KeyAcmePrivateKey); err != nil {
		log.Panicf("Faield to remove ACME private key: %v", err)
	}
}
