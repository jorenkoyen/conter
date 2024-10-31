package model

// ChallengeType defines the ACME challenge to use when requesting a new SSL certificate.
type ChallengeType string

const (
	ChallengeHttp01 ChallengeType = "http01"
	ChallengeDns01  ChallengeType = "dns01"
)

// IngressRoute contains all the required details to proxy through a request based on the incoming domain name.
type IngressRoute struct {
	Domain   string `json:"domain"`
	Endpoint string `json:"endpoint"`
	Service  string `json:"service"`
	Project  string `json:"project"`

	// TODO: certificates.
}

// HasValidCertificates will return true if the ingress route has SSL certificates that are still valid.
func (r *IngressRoute) HasValidCertificates() bool {
	return false
}
