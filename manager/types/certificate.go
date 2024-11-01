package types

import (
	"crypto"
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
