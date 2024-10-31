package manifest

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
